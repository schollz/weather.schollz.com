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
	if requests != 1 {
		t.Fatalf("expected cached lookup, got %d requests", requests)
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
