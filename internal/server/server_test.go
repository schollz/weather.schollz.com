package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"wthrtxt.com/internal/weather"
)

type fakeReporter struct {
	calls    int
	location weather.Location
	report   weather.WeatherReport
	err      error
}

func (f *fakeReporter) Report(_ context.Context, location weather.Location) (weather.WeatherReport, error) {
	f.calls++
	f.location = location
	report := f.report
	report.Location = location
	return report, f.err
}

type fakeGeocoder struct {
	slugLocation    weather.Location
	reverseLocation weather.Location
	slug            string
}

func (f *fakeGeocoder) ResolveSlug(_ context.Context, slug string) (weather.Location, error) {
	f.slug = slug
	return f.slugLocation, nil
}

func (f *fakeGeocoder) Reverse(_ context.Context, _, _ float64) (weather.Location, error) {
	return f.reverseLocation, nil
}

type fakeIPResolver struct {
	address  netip.Addr
	location weather.Location
}

func (f *fakeIPResolver) Resolve(address netip.Addr) (weather.Location, error) {
	f.address = address
	return f.location, nil
}

func testServer(t *testing.T) (*Server, *fakeReporter, *fakeGeocoder, *fakeIPResolver) {
	t.Helper()
	assets := fstest.MapFS{
		"index.html":             {Data: []byte("<!doctype html><title>wthrtxt.com</title><body></body>")},
		"about/index.html":       {Data: []byte("<!doctype html><title>About wthrtxt.com</title><body></body>")},
		"assets/app-abc123.js":   {Data: []byte("console.log('ok')")},
		"favicon.svg":            {Data: []byte("<svg></svg>")},
		"og.png":                 {Data: []byte("social preview")},
		"robots.txt":             {Data: []byte("User-agent: *")},
		"assets/not-a-directory": {Data: []byte("ok")},
	}
	reporter := &fakeReporter{report: sampleServerReport()}
	geocoder := &fakeGeocoder{
		slugLocation: weather.Location{
			Name: "Seattle", Region: "WA", CountryCode: "US",
			TimeZone: "America/Los_Angeles", Latitude: 47.6062, Longitude: -122.3321,
		},
		reverseLocation: weather.Location{
			Name: "Portland", Region: "OR", CountryCode: "US",
			TimeZone: "America/Los_Angeles", Latitude: 45.52, Longitude: -122.68,
		},
	}
	ipResolver := &fakeIPResolver{location: geocoder.slugLocation}
	handler := New(assets, reporter, geocoder, ipResolver, true, nil)
	handler.Now = func() time.Time {
		return time.Date(2026, 7, 27, 9, 35, 0, 0, time.UTC)
	}
	return handler, reporter, geocoder, ipResolver
}

func sampleServerReport() weather.WeatherReport {
	temperature, humidity, rain := 61.0, 82.0, 2.0
	start := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	return weather.WeatherReport{
		Current: weather.CurrentConditions{
			ObservedAt:  start,
			Temperature: &temperature,
			Humidity:    &humidity,
			Sky:         "Cloudy",
			Wind:        "5 mph",
			Source:      "observed",
		},
		Hourly: []weather.HourlyReading{{
			StartTime: start, EndTime: start.Add(time.Hour), Temperature: &temperature,
			Humidity: &humidity, PrecipChance: &rain, Sky: "Cloudy",
		}},
		Daily: []weather.DailyForecast{{
			Date: start, High: &temperature, Low: &temperature, PrecipChance: &rain, Sky: "Cloudy",
		}},
		Provider:     "NOAA",
		RecordsState: "warming",
	}
}

func TestNegotiatesBrowserAndTerminalResponses(t *testing.T) {
	handler, reporter, geocoder, _ := testServer(t)

	browserRequest := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/seattle", nil)
	browserRequest.Header.Set("Accept", "text/html")
	browserRequest.Header.Set("User-Agent", "Mozilla/5.0")
	browserResponse := httptest.NewRecorder()
	handler.ServeHTTP(browserResponse, browserRequest)

	if browserResponse.Code != http.StatusOK ||
		!strings.Contains(browserResponse.Body.String(), "<title>wthrtxt.com</title>") {
		t.Fatalf("unexpected browser response: %d %q", browserResponse.Code, browserResponse.Body.String())
	}
	if reporter.calls != 0 {
		t.Fatal("browser request should not fetch server-side weather")
	}

	aboutRequest := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/about/", nil)
	aboutRequest.Header.Set("Accept", "text/html")
	aboutResponse := httptest.NewRecorder()
	handler.ServeHTTP(aboutResponse, aboutRequest)
	if aboutResponse.Code != http.StatusOK ||
		!strings.Contains(aboutResponse.Body.String(), "About wthrtxt.com") {
		t.Fatalf("unexpected about response: %d %q", aboutResponse.Code, aboutResponse.Body.String())
	}

	curlRequest := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/seattle", nil)
	curlRequest.Header.Set("Accept", "*/*")
	curlRequest.Header.Set("User-Agent", "curl/8.7.1")
	curlResponse := httptest.NewRecorder()
	handler.ServeHTTP(curlResponse, curlRequest)

	if curlResponse.Code != http.StatusOK {
		t.Fatalf("unexpected curl status: %d", curlResponse.Code)
	}
	if !strings.Contains(curlResponse.Body.String(), "Weather for Seattle, WA") {
		t.Fatalf("unexpected terminal body: %q", curlResponse.Body.String())
	}
	if geocoder.slug != "seattle" || reporter.calls != 1 {
		t.Fatalf("terminal route was not resolved: slug=%q calls=%d", geocoder.slug, reporter.calls)
	}
	if value := curlResponse.Header().Get("Cache-Control"); value != "public, max-age=300" {
		t.Fatalf("unexpected explicit-location cache policy: %q", value)
	}
	if value := curlResponse.Header().Get("Vary"); value != "User-Agent, Accept" {
		t.Fatalf("unexpected Vary header: %q", value)
	}
}

func TestUmamiTrackerRequiresBothEnvironmentValues(t *testing.T) {
	handler, _, _, _ := testServer(t)
	request := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/", nil)
	request.Header.Set("Accept", "text/html")

	for _, testCase := range []struct {
		name      string
		url       string
		websiteID string
		tracked   bool
	}{
		{name: "unset"},
		{name: "URL only", url: "https://umami.schollz.com"},
		{name: "website ID only", websiteID: "website-uuid"},
		{
			name:      "both values",
			url:       "https://umami.schollz.com/",
			websiteID: "website-uuid",
			tracked:   true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler.UmamiURL = testCase.url
			handler.UmamiWebsiteID = testCase.websiteID
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			script := `<script defer data-website-id="website-uuid" src="https://umami.schollz.com/script.js"></script>`
			if strings.Contains(response.Body.String(), script) != testCase.tracked {
				t.Fatalf("unexpected tracker response: %q", response.Body.String())
			}
		})
	}
}

func TestFormatOverrideAndStaticRoutes(t *testing.T) {
	handler, reporter, _, _ := testServer(t)

	curlHTML := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/?format=html", nil)
	curlHTML.Header.Set("User-Agent", "curl/8.7.1")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, curlHTML)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "<!doctype html>") {
		t.Fatalf("format=html did not override curl: %d %q", response.Code, response.Body.String())
	}

	assetRequest := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/assets/app-abc123.js", nil)
	assetRequest.Header.Set("User-Agent", "curl/8.7.1")
	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, assetRequest)
	if assetResponse.Code != http.StatusOK || assetResponse.Body.String() != "console.log('ok')" {
		t.Fatalf("asset was not served directly: %d %q", assetResponse.Code, assetResponse.Body.String())
	}
	if !strings.Contains(assetResponse.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("asset is missing immutable caching: %q", assetResponse.Header().Get("Cache-Control"))
	}

	ogRequest := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/og.png", nil)
	ogResponse := httptest.NewRecorder()
	handler.ServeHTTP(ogResponse, ogRequest)
	if ogResponse.Code != http.StatusOK || ogResponse.Body.String() != "social preview" {
		t.Fatalf("social preview was not served directly: %d %q", ogResponse.Code, ogResponse.Body.String())
	}
	if reporter.calls != 0 {
		t.Fatal("static routes should not fetch weather")
	}
}

func TestBrowserWeatherCacheWarmsTextViewForTheSameSession(t *testing.T) {
	handler, reporter, geocoder, _ := testServer(t)
	now := time.Now()
	handler.Now = func() time.Time { return now }
	report := sampleServerReport()
	report.Location = geocoder.slugLocation
	report.Current.ObservedAt = now
	report.Hourly[0].StartTime = now
	report.Hourly[0].EndTime = now.Add(time.Hour)
	report.Daily[0].Date = now
	body, err := json.Marshal(browserWeatherCacheRequest{
		MaxAgeSeconds: 3600,
		Path:          "/seattle/",
		Report:        report,
	})
	if err != nil {
		t.Fatal(err)
	}

	cacheRequest := httptest.NewRequest(
		http.MethodPost,
		"http://wthrtxt.com/api/weather-cache",
		bytes.NewReader(body),
	)
	cacheRequest.Header.Set("Content-Type", "application/json")
	cacheResponse := httptest.NewRecorder()
	handler.ServeHTTP(cacheResponse, cacheRequest)
	if cacheResponse.Code != http.StatusNoContent {
		t.Fatalf("unexpected cache status: %d %q", cacheResponse.Code, cacheResponse.Body.String())
	}
	cookies := cacheResponse.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != browserWeatherCacheCookie {
		t.Fatalf("browser cache session cookie was not set: %#v", cookies)
	}

	textRequest := httptest.NewRequest(
		http.MethodGet,
		"http://wthrtxt.com/seattle?format=text",
		nil,
	)
	textRequest.AddCookie(cookies[0])
	textResponse := httptest.NewRecorder()
	handler.ServeHTTP(textResponse, textRequest)
	if textResponse.Code != http.StatusOK ||
		!strings.Contains(textResponse.Body.String(), "Weather for Seattle, WA") {
		t.Fatalf("unexpected cached text response: %d %q", textResponse.Code, textResponse.Body.String())
	}
	if reporter.calls != 0 || geocoder.slug != "" {
		t.Fatalf("cached text view reached upstreams: reporter=%d slug=%q", reporter.calls, geocoder.slug)
	}
	if value := textResponse.Header().Get("Vary"); !strings.Contains(value, "Cookie") {
		t.Fatalf("cached text response does not vary by cookie: %q", value)
	}

	otherSessionRequest := httptest.NewRequest(
		http.MethodGet,
		"http://wthrtxt.com/seattle?format=text",
		nil,
	)
	otherSessionResponse := httptest.NewRecorder()
	handler.ServeHTTP(otherSessionResponse, otherSessionRequest)
	if otherSessionResponse.Code != http.StatusOK || reporter.calls != 1 {
		t.Fatalf(
			"uncached session did not use normal reporting: status=%d calls=%d",
			otherSessionResponse.Code,
			reporter.calls,
		)
	}
}

func TestTerminalLocationAliasRedirectsToCanonicalSlug(t *testing.T) {
	handler, reporter, geocoder, _ := testServer(t)
	geocoder.slugLocation = weather.Location{
		Name: "Portland", Region: "Oregon", CountryCode: "US",
		Latitude: 45.5202471, Longitude: -122.674194,
		CanonicalSlug: "portland-or",
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"http://wthrtxt.com/portland%20oregon?format=text",
		nil,
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusPermanentRedirect {
		t.Fatalf("unexpected status: %d %q", response.Code, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/portland-or/?format=text" {
		t.Fatalf("unexpected redirect location: %q", location)
	}
	if reporter.calls != 0 {
		t.Fatal("alias redirect should not fetch weather")
	}
}

func TestRootUsesTrustedForwardedIP(t *testing.T) {
	handler, _, _, resolver := testServer(t)
	request := httptest.NewRequest(http.MethodGet, "http://wthrtxt.com/", nil)
	request.Header.Set("User-Agent", "curl/8.7.1")
	request.Header.Set("X-Forwarded-For", "8.8.8.8")
	request.RemoteAddr = "10.0.0.2:1234"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d %q", response.Code, response.Body.String())
	}
	if resolver.address.String() != "8.8.8.8" {
		t.Fatalf("unexpected resolved IP: %s", resolver.address)
	}
	if value := response.Header().Get("Cache-Control"); value != "private, no-store" {
		t.Fatalf("unexpected IP cache policy: %q", value)
	}
	if !strings.Contains(response.Body.String(), "MaxMind GeoLite2 City (IP estimate)") {
		t.Fatal("IP attribution missing")
	}
}

func TestRejectsBadInputsAndMethods(t *testing.T) {
	handler, _, _, _ := testServer(t)
	cases := []struct {
		method string
		target string
		status int
	}{
		{http.MethodGet, "http://wthrtxt.com/999,999", http.StatusBadRequest},
		{http.MethodGet, "http://wthrtxt.com/?format=json", http.StatusBadRequest},
		{http.MethodPost, "http://wthrtxt.com/seattle", http.StatusMethodNotAllowed},
	}
	for _, testCase := range cases {
		t.Run(testCase.method+" "+testCase.target, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.target, nil)
			request.Header.Set("User-Agent", "curl/8.7.1")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.status {
				t.Fatalf("got %d, want %d: %q", response.Code, testCase.status, response.Body.String())
			}
			if !strings.HasPrefix(response.Body.String(), "wthrtxt.com:") {
				t.Fatalf("unexpected error body: %q", response.Body.String())
			}
		})
	}
}

func TestHealthAndHeadHaveNoBody(t *testing.T) {
	handler, _, _, _ := testServer(t)
	for _, target := range []string{"/healthz", "/seattle"} {
		request := httptest.NewRequest(http.MethodHead, "http://wthrtxt.com"+target, nil)
		request.Header.Set("User-Agent", "curl/8.7.1")
		if target == "/seattle" {
			request.Header.Set("Accept", "*/*")
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.Len() != 0 {
			t.Fatalf("%s: status=%d body=%q", target, response.Code, response.Body.String())
		}
	}
}

func TestEmbeddedTestFSIsValid(t *testing.T) {
	handler, _, _, _ := testServer(t)
	if _, err := fs.Stat(handler.Assets, "index.html"); err != nil {
		t.Fatal(err)
	}
}
