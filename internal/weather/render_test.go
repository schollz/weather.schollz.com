package weather

import (
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestRenderTextGolden(t *testing.T) {
	timezone, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 27, 2, 35, 0, 0, timezone)
	temperature, nextTemperature := 60.0, 61.0
	humidity, nextHumidity := 85.0, 84.0
	rain, nextRain := 2.0, 3.0
	high, low := 79.0, 58.0

	report := WeatherReport{
		Location: Location{
			Name: "Seattle", Region: "WA", Country: "United States", CountryCode: "US",
			TimeZone: "America/Los_Angeles", Latitude: 47.6062, Longitude: -122.3321,
		},
		Current: CurrentConditions{
			ObservedAt: now, Temperature: &nextTemperature, Humidity: &nextHumidity,
			Sky: "Cloudy", Wind: "5 mph", Source: "observed",
		},
		Hourly: []HourlyReading{
			{
				StartTime: now.Add(2*time.Hour + 25*time.Minute), EndTime: now.Add(3*time.Hour + 25*time.Minute),
				Temperature: &temperature, Humidity: &humidity, PrecipChance: &rain, Sky: "Cloudy",
			},
			{
				StartTime: now.Add(3*time.Hour + 25*time.Minute), EndTime: now.Add(4*time.Hour + 25*time.Minute),
				Temperature: &nextTemperature, Humidity: &nextHumidity, PrecipChance: &nextRain,
				Sky: "Partly cloudy with an intentionally very long description",
			},
		},
		Daily: []DailyForecast{
			{
				Date: now, High: &high, Low: &low, PrecipChance: &rain, Sky: "Cloudy",
				Record: &ClimateRecord{
					High: &ClimateValue{Date: "1965-07-27", Temperature: 96},
					Low:  &ClimateValue{Date: "1948-07-27", Temperature: 49},
				},
			},
			{
				Date: now.Add(24 * time.Hour), High: &high, Low: &low,
				PrecipChance: &nextRain, Sky: "Partly cloudy",
			},
		},
		Provider:      "NOAA",
		StationID:     "KBFI",
		RecordsSource: "ACIS / BOEING FIELD",
		RecordsState:  "ready",
	}

	actual := RenderText(report, now)
	expected, err := os.ReadFile("testdata/seattle.golden")
	if err != nil {
		t.Fatal(err)
	}
	if actual != string(expected) {
		t.Fatalf("rendered forecast differs from golden file\n--- got ---\n%s\n--- want ---\n%s", actual, expected)
	}
	if strings.Contains(actual, "\x1b") {
		t.Fatal("plaintext contains an ANSI escape")
	}
	for _, line := range strings.Split(strings.TrimSuffix(actual, "\n"), "\n") {
		if utf8.RuneCountInString(line) > 80 {
			t.Fatalf("line exceeds 80 columns (%d): %q", utf8.RuneCountInString(line), line)
		}
	}
}

func TestRenderTextShowsWarmingAndObservedRain(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	rain := 0.006
	report := WeatherReport{
		Location: Location{
			Name:   "A Very Long Municipality Name That Would Otherwise Overflow A Terminal",
			Region: "An Exceptionally Long Administrative Region", TimeZone: "UTC",
		},
		Current: CurrentConditions{ObservedAt: now, Source: "observed"},
		Hourly: []HourlyReading{{
			StartTime: now, EndTime: now.Add(time.Hour), Observed: true, PrecipInches: &rain, Sky: "Rain",
		}},
		Provider:       "Open-Meteo",
		RecordsState:   "warming",
		LocationFromIP: true,
	}
	rendered := RenderText(report, now)
	if !strings.Contains(rendered, "<.01in") ||
		!strings.Contains(rendered, "records: warming; retry shortly") {
		t.Fatalf("missing rainfall or warming state:\n%s", rendered)
	}
	for _, line := range strings.Split(strings.TrimSuffix(rendered, "\n"), "\n") {
		if utf8.RuneCountInString(line) > 80 {
			t.Fatalf("line exceeds 80 columns (%d): %q", utf8.RuneCountInString(line), line)
		}
	}
}

func TestRenderTextAttributesOpenStreetMapGeocoding(t *testing.T) {
	now := time.Date(2026, 7, 27, 12, 0, 0, 0, time.UTC)
	report := WeatherReport{
		Location: Location{
			Name: "Portland", Region: "Oregon", TimeZone: "UTC",
			Latitude: 45.5202471, Longitude: -122.674194,
			Source: "OpenStreetMap Nominatim",
		},
		Current:      CurrentConditions{ObservedAt: now},
		Provider:     "Open-Meteo",
		RecordsState: "unavailable",
	}

	rendered := RenderText(report, now)
	if !strings.Contains(
		rendered,
		"geocoding: © OpenStreetMap contributors / openstreetmap.org/copyright",
	) {
		t.Fatalf("OpenStreetMap attribution missing:\n%s", rendered)
	}
}
