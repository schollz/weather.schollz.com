package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"wthrtxt.com/internal/store"
)

func TestSlugHelpersMatchBrowserRoutes(t *testing.T) {
	if actual := slugify("São José’s Weather"); actual != "sao-joses-weather" {
		t.Fatalf("unexpected slug: %q", actual)
	}
	terms := searchTermsForSlug("hillsboro-or")
	if strings.Join(terms, "|") != "hillsboro or|hillsboro" {
		t.Fatalf("unexpected search terms: %#v", terms)
	}

	candidates := []placeResult{
		{Name: "Hillsboro", Admin1: "Oregon", CountryCode: "US"},
		{Name: "Hillsboro", Admin1: "Ohio", CountryCode: "US"},
	}
	if actual := locationSlug(candidates[0], candidates); actual != "hillsboro-or" {
		t.Fatalf("unexpected ambiguous slug: %q", actual)
	}
}

func TestGeocoderUsesFixturesAndCache(t *testing.T) {
	requests := 0
	fixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests++
		if request.Header.Get("User-Agent") != upstreamUserAgent {
			t.Errorf("unexpected User-Agent: %q", request.Header.Get("User-Agent"))
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"results":[{"name":"Seattle","admin1":"Washington","country":"United States","country_code":"US","feature_code":"PPLA2","latitude":47.6062,"longitude":-122.3321,"timezone":"America/Los_Angeles"}]}`))
	}))
	defer fixture.Close()

	httpClient := &HTTPClient{Client: fixture.Client(), Retries: 0}
	geocoder := NewGeocoder(httpClient, store.NewMemory(10))
	geocoder.GeocodingURL = fixture.URL

	first, err := geocoder.ResolveSlug(context.Background(), "seattle")
	if err != nil {
		t.Fatal(err)
	}
	second, err := geocoder.ResolveSlug(context.Background(), "seattle")
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Seattle" || second.TimeZone != "America/Los_Angeles" {
		t.Fatalf("unexpected locations: %#v %#v", first, second)
	}
	if first.CanonicalSlug != "seattle" {
		t.Fatalf("unexpected canonical slug: %q", first.CanonicalSlug)
	}
	if requests != 1 {
		t.Fatalf("expected cached lookup, got %d requests", requests)
	}
}

func TestGeocoderFallsBackToNominatimForwardSearch(t *testing.T) {
	geocodingRequests := 0
	geocodingFixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		geocodingRequests++
		response.Header().Set("Content-Type", "application/json")
		if request.URL.Query().Get("name") == "portland" {
			_, _ = response.Write([]byte(`{"results":[
				{"name":"Portland","admin1":"Oregon","country":"United States","country_code":"US","feature_code":"PPLA2","latitude":45.52345,"longitude":-122.67621},
				{"name":"Portland","admin1":"Maine","country":"United States","country_code":"US","feature_code":"PPLA2","latitude":43.65737,"longitude":-70.2589}
			]}`))
			return
		}
		_, _ = response.Write([]byte(`{"results":[]}`))
	}))
	defer geocodingFixture.Close()

	forwardRequests := 0
	forwardFixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		forwardRequests++
		if request.URL.Query().Get("q") != "portland oregon" ||
			request.URL.Query().Get("featureType") != "settlement" {
			t.Errorf("unexpected forward query: %s", request.URL.RawQuery)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"features":[{
			"properties":{"geocoding":{
				"name":"Portland","state":"Oregon","country":"United States","country_code":"us"
			}},
			"geometry":{"coordinates":[-122.674194,45.5202471]}
		}]}`))
	}))
	defer forwardFixture.Close()

	geocoder := NewGeocoder(
		&HTTPClient{Client: geocodingFixture.Client(), Retries: 0},
		store.NewMemory(10),
	)
	geocoder.GeocodingURL = geocodingFixture.URL
	geocoder.ForwardURL = forwardFixture.URL
	geocoder.ReverseDelay = 0

	first, err := geocoder.ResolveSlug(context.Background(), "portland oregon")
	if err != nil {
		t.Fatal(err)
	}
	second, err := geocoder.ResolveSlug(context.Background(), "portland-oregon")
	if err != nil {
		t.Fatal(err)
	}

	if first.Name != "Portland" || first.Region != "Oregon" ||
		first.Source != "OpenStreetMap Nominatim" ||
		first.CanonicalSlug != "portland-or" ||
		first.Latitude != 45.5202471 || first.Longitude != -122.674194 {
		t.Fatalf("unexpected location: %#v", first)
	}
	if second != first {
		t.Fatalf("expected normalized slug cache hit: %#v %#v", first, second)
	}
	if geocodingRequests != 2 || forwardRequests != 1 {
		t.Fatalf(
			"unexpected request counts: geocoding=%d forward=%d",
			geocodingRequests,
			forwardRequests,
		)
	}
}

func TestReverseGeocoderParsesNominatim(t *testing.T) {
	fixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"features":[{"properties":{"geocoding":{"city":"Portland","state":"Oregon","country":"United States","country_code":"us"}}}]}`))
	}))
	defer fixture.Close()

	geocoder := NewGeocoder(
		&HTTPClient{Client: fixture.Client(), Retries: 0},
		store.NewMemory(10),
	)
	geocoder.ReverseURL = fixture.URL
	geocoder.ReverseDelay = 0
	location, err := geocoder.Reverse(context.Background(), 45.52, -122.68)
	if err != nil {
		t.Fatal(err)
	}
	if location.Name != "Portland" || location.CountryCode != "US" ||
		location.Latitude != 45.52 || location.Longitude != -122.68 {
		t.Fatalf("unexpected reverse location: %#v", location)
	}
}
