package weather

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/oschwald/maxminddb-golang"
	"golang.org/x/sync/singleflight"
	"golang.org/x/text/unicode/norm"

	"wthrtxt.com/internal/store"
)

const (
	defaultGeocodingURL = "https://geocoding-api.open-meteo.com/v1/search"
	defaultForwardURL   = "https://nominatim.openstreetmap.org/search"
	defaultReverseURL   = "https://nominatim.openstreetmap.org/reverse"
)

var slugSeparator = regexp.MustCompile(`[^a-z0-9]+`)

type locationCache interface {
	Get(string, any) bool
	Put(string, any, time.Time) error
}

type Geocoder struct {
	HTTP          *HTTPClient
	Cache         locationCache
	GeocodingURL  string
	ForwardURL    string
	ReverseURL    string
	ReverseDelay  time.Duration
	now           func() time.Time
	group         singleflight.Group
	nominatimMu   sync.Mutex
	lastNominatim time.Time
}

func NewGeocoder(httpClient *HTTPClient, cache *store.Store) *Geocoder {
	return &Geocoder{
		HTTP:         httpClient,
		Cache:        cache,
		GeocodingURL: defaultGeocodingURL,
		ForwardURL:   defaultForwardURL,
		ReverseURL:   defaultReverseURL,
		ReverseDelay: time.Second,
		now:          time.Now,
	}
}

type geocodingResponse struct {
	Results []placeResult `json:"results"`
}

type placeResult struct {
	Admin1      string  `json:"admin1"`
	Admin2      string  `json:"admin2"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	FeatureCode string  `json:"feature_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Name        string  `json:"name"`
	Timezone    string  `json:"timezone"`
}

type nominatimResponse struct {
	Features []struct {
		Geometry struct {
			Coordinates []float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties struct {
			Geocoding nominatimPlace `json:"geocoding"`
		} `json:"properties"`
	} `json:"features"`
}

type nominatimPlace struct {
	City         string `json:"city"`
	Country      string `json:"country"`
	CountryCode  string `json:"country_code"`
	County       string `json:"county"`
	District     string `json:"district"`
	Locality     string `json:"locality"`
	Municipality string `json:"municipality"`
	Name         string `json:"name"`
	State        string `json:"state"`
	Town         string `json:"town"`
	Village      string `json:"village"`
}

func (g *Geocoder) ResolveSlug(ctx context.Context, slug string) (Location, error) {
	slug = slugify(slug)
	if slug == "" || len(slug) > 100 {
		return Location{}, errors.New("invalid location slug")
	}

	cacheKey := "geocode:slug:" + slug
	var cached Location
	if g.Cache != nil && g.Cache.Get(cacheKey, &cached) {
		return cached, nil
	}

	value, err, _ := g.group.Do(cacheKey, func() (any, error) {
		var resultSets [][]placeResult
		var geocodingErr error
		for _, query := range searchTermsForSlug(slug) {
			results, fetchErr := g.search(ctx, query)
			if fetchErr != nil {
				geocodingErr = fetchErr
				break
			}
			resultSets = append(resultSets, results)
			for _, place := range results {
				if locationSlug(place, results) != slug {
					continue
				}
				location := locationFromPlace(place)
				location.CanonicalSlug = slug
				if g.Cache != nil {
					_ = g.Cache.Put(cacheKey, location, g.now().Add(30*24*time.Hour))
				}
				return location, nil
			}
		}

		location, forwardErr := g.forward(ctx, slug, resultSets)
		if forwardErr == nil {
			if g.Cache != nil {
				_ = g.Cache.Put(cacheKey, location, g.now().Add(30*24*time.Hour))
			}
			return location, nil
		}
		if geocodingErr != nil {
			return nil, geocodingErr
		}
		return nil, forwardErr
	})
	if err != nil {
		return Location{}, err
	}
	return value.(Location), nil
}

func (g *Geocoder) search(ctx context.Context, query string) ([]placeResult, error) {
	endpoint, err := url.Parse(g.GeocodingURL)
	if err != nil {
		return nil, err
	}
	values := endpoint.Query()
	values.Set("name", query)
	values.Set("count", "10")
	values.Set("language", "en")
	values.Set("format", "json")
	endpoint.RawQuery = values.Encode()

	var response geocodingResponse
	if err := g.HTTP.JSON(ctx, "Open-Meteo geocoding", "GET", endpoint.String(), "", nil, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func (g *Geocoder) forward(
	ctx context.Context,
	slug string,
	resultSets [][]placeResult,
) (Location, error) {
	if err := g.waitForNominatim(ctx); err != nil {
		return Location{}, err
	}

	endpoint, err := url.Parse(g.ForwardURL)
	if err != nil {
		return Location{}, err
	}
	values := endpoint.Query()
	values.Set("q", strings.ReplaceAll(slug, "-", " "))
	values.Set("format", "geocodejson")
	values.Set("addressdetails", "1")
	values.Set("featureType", "settlement")
	values.Set("limit", "5")
	values.Set("accept-language", "en")
	endpoint.RawQuery = values.Encode()

	var response nominatimResponse
	if err := g.HTTP.JSON(
		ctx,
		"Nominatim",
		"GET",
		endpoint.String(),
		"",
		nil,
		&response,
	); err != nil {
		return Location{}, err
	}

	for _, feature := range response.Features {
		if len(feature.Geometry.Coordinates) < 2 {
			continue
		}
		place := feature.Properties.Geocoding
		name := firstNonEmpty(
			place.Name,
			place.City,
			place.Town,
			place.Village,
			place.Municipality,
			place.Locality,
		)
		countryCode := strings.ToUpper(place.CountryCode)
		longitude := feature.Geometry.Coordinates[0]
		latitude := feature.Geometry.Coordinates[1]
		if name == "" || countryCode == "" || !validCoordinates(latitude, longitude) {
			continue
		}

		location := Location{
			Name:        name,
			Region:      firstNonEmpty(place.State, place.County, place.District),
			Country:     firstNonEmpty(place.Country, countryCode),
			CountryCode: countryCode,
			Latitude:    latitude,
			Longitude:   longitude,
			Source:      "OpenStreetMap Nominatim",
		}
		location.CanonicalSlug = canonicalSlugForLocation(location, resultSets)
		if location.CanonicalSlug == "" {
			location.CanonicalSlug = slug
		}
		return location, nil
	}

	return Location{}, errors.New("location not found")
}

func (g *Geocoder) Reverse(ctx context.Context, latitude, longitude float64) (Location, error) {
	if !validCoordinates(latitude, longitude) {
		return Location{}, errors.New("invalid coordinates")
	}

	cacheKey := fmt.Sprintf("geocode:reverse:%.3f,%.3f", latitude, longitude)
	var cached Location
	if g.Cache != nil && g.Cache.Get(cacheKey, &cached) {
		return cached, nil
	}

	value, err, _ := g.group.Do(cacheKey, func() (any, error) {
		if err := g.waitForNominatim(ctx); err != nil {
			return nil, err
		}

		endpoint, parseErr := url.Parse(g.ReverseURL)
		if parseErr != nil {
			return nil, parseErr
		}
		values := endpoint.Query()
		values.Set("lat", fmt.Sprintf("%.4f", latitude))
		values.Set("lon", fmt.Sprintf("%.4f", longitude))
		values.Set("format", "geocodejson")
		values.Set("zoom", "10")
		values.Set("addressdetails", "1")
		values.Set("accept-language", "en")
		endpoint.RawQuery = values.Encode()

		var response nominatimResponse
		if fetchErr := g.HTTP.JSON(ctx, "Nominatim", "GET", endpoint.String(), "", nil, &response); fetchErr != nil {
			return nil, fetchErr
		}
		if len(response.Features) == 0 {
			return nil, errors.New("location not found")
		}

		place := response.Features[0].Properties.Geocoding
		name := firstNonEmpty(
			place.City,
			place.Town,
			place.Village,
			place.Municipality,
			place.Locality,
			place.District,
			place.Name,
		)
		if name == "" || place.CountryCode == "" {
			return nil, errors.New("location not found")
		}

		location := Location{
			Name:        name,
			Region:      firstNonEmpty(place.State, place.District),
			Country:     firstNonEmpty(place.Country, strings.ToUpper(place.CountryCode)),
			CountryCode: strings.ToUpper(place.CountryCode),
			Latitude:    latitude,
			Longitude:   longitude,
			Source:      "OpenStreetMap Nominatim",
		}
		if g.Cache != nil {
			_ = g.Cache.Put(cacheKey, location, g.now().Add(30*24*time.Hour))
		}
		return location, nil
	})
	if err != nil {
		return Location{}, err
	}
	return value.(Location), nil
}

func (g *Geocoder) waitForNominatim(ctx context.Context) error {
	g.nominatimMu.Lock()
	defer g.nominatimMu.Unlock()

	wait := g.ReverseDelay - time.Since(g.lastNominatim)
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	g.lastNominatim = time.Now()
	return nil
}

func locationFromPlace(place placeResult) Location {
	return Location{
		Name:        place.Name,
		Region:      firstNonEmpty(place.Admin1, place.Admin2),
		Country:     firstNonEmpty(place.Country, strings.ToUpper(place.CountryCode)),
		CountryCode: strings.ToUpper(place.CountryCode),
		TimeZone:    place.Timezone,
		Latitude:    place.Latitude,
		Longitude:   place.Longitude,
		Source:      "Open-Meteo geocoding",
	}
}

func slugify(value string) string {
	normalized := norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var builder strings.Builder
	for _, character := range normalized {
		if unicode.Is(unicode.Mn, character) {
			continue
		}
		if character == '\'' || character == '’' {
			continue
		}
		builder.WriteRune(character)
	}
	return strings.Trim(slugSeparator.ReplaceAllString(builder.String(), "-"), "-")
}

func searchTermsForSlug(slug string) []string {
	parts := strings.Fields(strings.ReplaceAll(slugify(slug), "-", " "))
	terms := make([]string, 0, len(parts))
	for length := len(parts); length > 0; length-- {
		term := strings.Join(parts[:length], " ")
		if len(term) >= 2 {
			terms = append(terms, term)
		}
	}
	return terms
}

func locationSlug(place placeResult, candidates []placeResult) string {
	citySlug := slugify(place.Name)
	if citySlug == "" {
		return ""
	}

	duplicates := 0
	for _, candidate := range candidates {
		if candidate.FeatureCode == "PPLX" {
			continue
		}
		if strings.EqualFold(candidate.CountryCode, place.CountryCode) &&
			slugify(candidate.Name) == citySlug {
			duplicates++
		}
	}
	if duplicates <= 1 {
		return citySlug
	}

	region := slugify(firstNonEmpty(place.Admin1, place.Admin2))
	if strings.EqualFold(place.CountryCode, "US") {
		if code := usStateCodes[region]; code != "" {
			region = code
		}
	} else if region == "" {
		region = strings.ToLower(place.CountryCode)
	}
	if region == "" {
		return citySlug
	}
	return citySlug + "-" + region
}

func canonicalSlugForLocation(location Location, resultSets [][]placeResult) string {
	locationName := slugify(location.Name)
	locationRegion := slugify(location.Region)

	for _, candidates := range resultSets {
		for _, candidate := range candidates {
			if !strings.EqualFold(candidate.CountryCode, location.CountryCode) ||
				slugify(candidate.Name) != locationName {
				continue
			}
			candidateRegion := slugify(firstNonEmpty(candidate.Admin1, candidate.Admin2))
			if locationRegion != "" &&
				candidateRegion != "" &&
				locationRegion != candidateRegion {
				continue
			}
			return locationSlug(candidate, candidates)
		}
	}
	return ""
}

func validCoordinates(latitude, longitude float64) bool {
	return latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type GeoIP struct {
	reader *maxminddb.Reader
}

func OpenGeoIP(path string) (*GeoIP, error) {
	reader, err := maxminddb.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open GeoLite2 database: %w", err)
	}
	if !strings.Contains(strings.ToLower(reader.Metadata.DatabaseType), "city") {
		_ = reader.Close()
		return nil, fmt.Errorf("GeoLite2 database has unexpected type %q", reader.Metadata.DatabaseType)
	}
	return &GeoIP{reader: reader}, nil
}

func (g *GeoIP) Resolve(address netip.Addr) (Location, error) {
	if g == nil || g.reader == nil {
		return Location{}, errors.New("GeoLite2 database is unavailable")
	}
	var record struct {
		City struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"city"`
		Country struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
		Location struct {
			Latitude  float64 `maxminddb:"latitude"`
			Longitude float64 `maxminddb:"longitude"`
			TimeZone  string  `maxminddb:"time_zone"`
		} `maxminddb:"location"`
		Subdivisions []struct {
			ISOCode string            `maxminddb:"iso_code"`
			Names   map[string]string `maxminddb:"names"`
		} `maxminddb:"subdivisions"`
	}
	if err := g.reader.Lookup(address.AsSlice(), &record); err != nil {
		return Location{}, err
	}

	region := ""
	if len(record.Subdivisions) > 0 {
		region = firstNonEmpty(record.Subdivisions[0].ISOCode, record.Subdivisions[0].Names["en"])
	}
	location := Location{
		Name:        record.City.Names["en"],
		Region:      region,
		Country:     record.Country.Names["en"],
		CountryCode: record.Country.ISOCode,
		TimeZone:    record.Location.TimeZone,
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
		Source:      "MaxMind GeoLite2 City",
	}
	if location.Name == "" || location.CountryCode == "" ||
		!validCoordinates(location.Latitude, location.Longitude) {
		return Location{}, errors.New("IP address has no usable city location")
	}
	return location, nil
}

func (g *GeoIP) Close() error {
	if g == nil || g.reader == nil {
		return nil
	}
	return g.reader.Close()
}

var usStateCodes = map[string]string{
	"alabama": "al", "alaska": "ak", "arizona": "az", "arkansas": "ar",
	"california": "ca", "colorado": "co", "connecticut": "ct", "delaware": "de",
	"district-of-columbia": "dc", "florida": "fl", "georgia": "ga", "hawaii": "hi",
	"idaho": "id", "illinois": "il", "indiana": "in", "iowa": "ia",
	"kansas": "ks", "kentucky": "ky", "louisiana": "la", "maine": "me",
	"maryland": "md", "massachusetts": "ma", "michigan": "mi", "minnesota": "mn",
	"mississippi": "ms", "missouri": "mo", "montana": "mt", "nebraska": "ne",
	"nevada": "nv", "new-hampshire": "nh", "new-jersey": "nj", "new-mexico": "nm",
	"new-york": "ny", "north-carolina": "nc", "north-dakota": "nd", "ohio": "oh",
	"oklahoma": "ok", "oregon": "or", "pennsylvania": "pa", "rhode-island": "ri",
	"south-carolina": "sc", "south-dakota": "sd", "tennessee": "tn", "texas": "tx",
	"utah": "ut", "vermont": "vt", "virginia": "va", "washington": "wa",
	"west-virginia": "wv", "wisconsin": "wi", "wyoming": "wy",
}
