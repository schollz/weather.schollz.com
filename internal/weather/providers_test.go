package weather

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenMeteoParsesForecastFixture(t *testing.T) {
	fixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("User-Agent") != upstreamUserAgent {
			t.Errorf("unexpected User-Agent: %q", request.Header.Get("User-Agent"))
		}
		query := request.URL.Query()
		if query.Get("temperature_unit") != "fahrenheit" ||
			query.Get("wind_speed_unit") != "mph" ||
			query.Get("forecast_days") != "7" {
			t.Errorf("missing forecast units: %s", request.URL.RawQuery)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{
			"timezone":"America/Los_Angeles",
			"current":{"time":1785144000,"temperature_2m":61,"relative_humidity_2m":82,"weather_code":3,"wind_speed_10m":5,"wind_direction_10m":315},
			"hourly":{"time":[1785144000,1785147600],"temperature_2m":[61,62],"relative_humidity_2m":[82,80],"precipitation_probability":[2,3],"precipitation":[0,0.01],"weather_code":[3,2],"wind_speed_10m":[5,6],"wind_direction_10m":[315,300]},
			"daily":{"time":[1785110400],"weather_code":[3],"temperature_2m_max":[79],"temperature_2m_min":[58],"precipitation_probability_max":[4]}
		}`))
	}))
	defer fixture.Close()

	provider := NewOpenMeteo(&HTTPClient{Client: fixture.Client(), Retries: 0})
	provider.ForecastURL = fixture.URL
	report, err := provider.Fetch(context.Background(), Location{
		Name: "Vancouver", CountryCode: "CA", Latitude: 49.28, Longitude: -123.12,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Provider != "Open-Meteo" || report.Location.TimeZone != "America/Los_Angeles" {
		t.Fatalf("unexpected report metadata: %#v", report)
	}
	if report.Current.Sky != "Overcast" || report.Current.Wind != "5 mph NW" {
		t.Fatalf("unexpected current conditions: %#v", report.Current)
	}
	if len(report.Hourly) != 2 || len(report.Daily) != 1 ||
		report.Hourly[1].Sky != "Partly cloudy" || *report.Daily[0].High != 79 {
		t.Fatalf("unexpected forecast shape: %#v", report)
	}
}

func TestNOAAJoinsForecastObservationAndGridFixtures(t *testing.T) {
	var fixtureURL string
	fixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Header.Get("User-Agent") != upstreamUserAgent {
			t.Errorf("unexpected User-Agent: %q", request.Header.Get("User-Agent"))
		}
		response.Header().Set("Content-Type", "application/geo+json")
		switch {
		case strings.HasPrefix(request.URL.Path, "/points/"):
			_, _ = fmt.Fprintf(response, `{"properties":{"forecast":"%s/daily","forecastGridData":"%s/grid","forecastHourly":"%s/hourly","observationStations":"%s/stations","timeZone":"America/Los_Angeles","relativeLocation":{"properties":{"city":"Seattle","state":"WA"}}}}`, fixtureURL, fixtureURL, fixtureURL, fixtureURL)
		case request.URL.Path == "/hourly":
			_, _ = response.Write([]byte(`{"properties":{"periods":[
				{"startTime":"2026-07-27T05:00:00-07:00","endTime":"2026-07-27T06:00:00-07:00","isDaytime":true,"temperature":60,"temperatureUnit":"F","relativeHumidity":{"unitCode":"wmoUnit:percent","value":85},"probabilityOfPrecipitation":{"unitCode":"wmoUnit:percent","value":2},"shortForecast":"Cloudy","windSpeed":"5 mph","windDirection":"NW"},
				{"startTime":"2026-07-27T06:00:00-07:00","endTime":"2026-07-27T07:00:00-07:00","isDaytime":true,"temperature":61,"temperatureUnit":"F","relativeHumidity":{"unitCode":"wmoUnit:percent","value":84},"probabilityOfPrecipitation":{"unitCode":"wmoUnit:percent","value":3},"shortForecast":"Partly Cloudy","windSpeed":"6 mph","windDirection":"NW"}
			]}}`))
		case request.URL.Path == "/daily":
			_, _ = response.Write([]byte(`{"properties":{"periods":[
				{"startTime":"2026-07-27T06:00:00-07:00","endTime":"2026-07-27T18:00:00-07:00","isDaytime":true,"temperature":79,"temperatureUnit":"F","probabilityOfPrecipitation":{"unitCode":"wmoUnit:percent","value":4},"shortForecast":"Cloudy"},
				{"startTime":"2026-07-27T18:00:00-07:00","endTime":"2026-07-28T06:00:00-07:00","isDaytime":false,"temperature":58,"temperatureUnit":"F","probabilityOfPrecipitation":{"unitCode":"wmoUnit:percent","value":2},"shortForecast":"Cloudy"},
				{"startTime":"2026-07-28T06:00:00-07:00","endTime":"2026-07-28T18:00:00-07:00","isDaytime":true,"temperature":76,"temperatureUnit":"F","probabilityOfPrecipitation":{"unitCode":"wmoUnit:percent","value":18},"shortForecast":"Rain"},
				{"startTime":"2026-07-28T18:00:00-07:00","endTime":"2026-07-29T06:00:00-07:00","isDaytime":false,"temperature":57,"temperatureUnit":"F","probabilityOfPrecipitation":{"unitCode":"wmoUnit:percent","value":8},"shortForecast":"Cloudy"}
			]}}`))
		case request.URL.Path == "/stations":
			_, _ = fmt.Fprintf(response, `{"features":[{"id":"%s/stations/KBFI"}]}`, fixtureURL)
		case request.URL.Path == "/stations/KBFI/observations":
			_, _ = response.Write([]byte(`{"features":[{"properties":{"timestamp":"2026-07-27T12:55:00+00:00","textDescription":"Light Rain","temperature":{"unitCode":"wmoUnit:degC","value":10},"relativeHumidity":{"unitCode":"wmoUnit:percent","value":90},"precipitationLastHour":{"unitCode":"wmoUnit:m","value":0.002},"rawMessage":"KBFI P0008"}}]}`))
		case request.URL.Path == "/grid":
			_, _ = response.Write([]byte(`{"properties":{"quantitativePrecipitation":{"uom":"wmoUnit:mm","values":[{"validTime":"2026-07-27T12:00:00+00:00/PT2H","value":25.4}]}}}`))
		default:
			http.NotFound(response, request)
		}
	}))
	fixtureURL = fixture.URL
	defer fixture.Close()

	provider := NewNOAA(&HTTPClient{Client: fixture.Client(), Retries: 0})
	provider.BaseURL = fixture.URL
	provider.Now = func() time.Time {
		return time.Date(2026, 7, 27, 13, 30, 0, 0, time.UTC)
	}
	report, err := provider.Fetch(context.Background(), Location{
		CountryCode: "US", Latitude: 47.6062, Longitude: -122.3321,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Provider != "NOAA" || report.StationID != "KBFI" ||
		report.Location.Name != "Seattle" || report.Location.Region != "WA" {
		t.Fatalf("unexpected NOAA metadata: %#v", report)
	}
	if report.Current.Temperature == nil || *report.Current.Temperature != 50 ||
		report.Current.Sky != "Light Rain" || report.Current.Wind != "5 mph NW" {
		t.Fatalf("unexpected current observation: %#v", report.Current)
	}
	if len(report.Hourly) != 2 || !report.Hourly[0].Observed ||
		report.Hourly[0].PrecipInches == nil || *report.Hourly[0].PrecipInches < 0.07 {
		t.Fatalf("station observation was not joined: %#v", report.Hourly)
	}
	if report.Hourly[1].PrecipInches == nil || *report.Hourly[1].PrecipInches != 0.5 {
		t.Fatalf("grid precipitation was not allocated: %#v", report.Hourly[1])
	}
	if len(report.Daily) != 2 || *report.Daily[0].High != 79 ||
		*report.Daily[0].Low != 58 || *report.Daily[1].PrecipChance != 18 {
		t.Fatalf("daily periods were not aggregated: %#v", report.Daily)
	}
}

func TestRecordProviderFixtures(t *testing.T) {
	fixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/acis":
			_, _ = response.Write([]byte(`{"meta":{"name":"BOEING FIELD"},"smry":[[["96","1965-07-27"],["101","1994-07-28"]],[["49","1948-07-27"],["50","1950-07-28"]]]}`))
		case "/archive":
			if request.URL.Query().Get("start_date") != "1950-01-01" ||
				request.URL.Query().Get("models") != "era5_land" {
				t.Errorf("unexpected archive query: %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{"daily":{"time":["1950-07-27","1951-07-27"],"temperature_2m_max":[80,85],"temperature_2m_min":[50,45]}}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer fixture.Close()

	records := NewRecordsClient(&HTTPClient{Client: fixture.Client(), Retries: 0})
	records.ACISURL = fixture.URL + "/acis"
	records.ArchiveURL = fixture.URL + "/archive"

	acis, err := records.ACIS(context.Background(), "KBFI")
	if err != nil {
		t.Fatal(err)
	}
	if acis.Records["07-27"].High.Temperature != 96 ||
		acis.Records["07-27"].Low.Temperature != 49 ||
		acis.Source != "ACIS / BOEING FIELD" {
		t.Fatalf("unexpected ACIS records: %#v", acis)
	}

	era, err := records.ERA5(context.Background(), Location{
		Latitude: 49.28, Longitude: -123.12, TimeZone: "America/Vancouver",
	}, 2025)
	if err != nil {
		t.Fatal(err)
	}
	record := era.Records["07-27"]
	if record.High.Temperature != 85 || record.Low.Temperature != 45 ||
		!record.High.Estimated || !strings.Contains(era.Source, "1950–2025") {
		t.Fatalf("unexpected ERA5 records: %#v", era)
	}
}

func TestNOAAPrecipitationHelpers(t *testing.T) {
	duration, err := parseISODuration("P1DT2H30M")
	if err != nil || duration != 26*time.Hour+30*time.Minute {
		t.Fatalf("unexpected duration: %v %v", duration, err)
	}
	inches, err := precipitationInches(25.4, "wmoUnit:mm")
	if err != nil || inches != 1 {
		t.Fatalf("unexpected conversion: %v %v", inches, err)
	}
}

func TestObservationHoursPreserveRepeatedDSTHour(t *testing.T) {
	observations := []noaaObservation{
		{Timestamp: "2026-11-01T05:55:00Z", RawMessage: "FIRST"},
		{Timestamp: "2026-11-01T06:55:00Z", RawMessage: "SECOND"},
	}
	byHour := preferredObservationsByHour(observations)
	if len(byHour) != 2 ||
		byHour["2026-11-01-05"].RawMessage != "FIRST" ||
		byHour["2026-11-01-06"].RawMessage != "SECOND" {
		t.Fatalf("repeated DST observations were collapsed: %#v", byHour)
	}
}

func TestLastCompletedYearUsesLocationTimezone(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 30, 0, 0, time.UTC)
	if actual := lastCompletedYear(now, "America/Los_Angeles"); actual != 2024 {
		t.Fatalf("got %d, want 2024", actual)
	}
}
