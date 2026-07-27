package weather

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"wthrtxt.com/internal/store"
)

func TestServiceCoalescesConcurrentForecastsAndWarmsRecords(t *testing.T) {
	var forecastRequests atomic.Int32
	var archiveRequests atomic.Int32
	fixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/forecast":
			forecastRequests.Add(1)
			time.Sleep(40 * time.Millisecond)
			_, _ = response.Write([]byte(`{
				"timezone":"UTC",
				"current":{"time":1785144000,"temperature_2m":61,"relative_humidity_2m":82,"weather_code":3,"wind_speed_10m":5,"wind_direction_10m":315},
				"hourly":{"time":[1785144000],"temperature_2m":[61],"relative_humidity_2m":[82],"precipitation_probability":[2],"precipitation":[0],"weather_code":[3],"wind_speed_10m":[5],"wind_direction_10m":[315]},
				"daily":{"time":[1785110400],"weather_code":[3],"temperature_2m_max":[79],"temperature_2m_min":[58],"precipitation_probability_max":[4]}
			}`))
		case "/archive":
			archiveRequests.Add(1)
			_, _ = response.Write([]byte(`{"daily":{"time":["1950-07-27"],"temperature_2m_max":[80],"temperature_2m_min":[50]}}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer fixture.Close()

	httpClient := &HTTPClient{Client: fixture.Client(), Retries: 0}
	openMeteo := NewOpenMeteo(httpClient)
	openMeteo.ForecastURL = fixture.URL + "/forecast"
	records := NewRecordsClient(httpClient)
	records.ArchiveURL = fixture.URL + "/archive"
	service := NewService(
		store.NewMemory(100),
		NewNOAA(httpClient),
		openMeteo,
		records,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	service.Now = func() time.Time {
		return time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	}
	location := Location{
		Name: "Vancouver", CountryCode: "CA", TimeZone: "UTC",
		Latitude: 49.28, Longitude: -123.12,
	}

	const callers = 20
	start := make(chan struct{})
	results := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := service.Report(context.Background(), location)
			results <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if forecastRequests.Load() != 1 {
		t.Fatalf("expected one coalesced forecast request, got %d", forecastRequests.Load())
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		report, err := service.Report(context.Background(), location)
		if err != nil {
			t.Fatal(err)
		}
		if report.RecordsState == "ready" {
			if report.Daily[0].Record == nil || report.Daily[0].Record.High == nil {
				t.Fatalf("ready report is missing records: %#v", report)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("record cache did not finish warming")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if archiveRequests.Load() != 1 {
		t.Fatalf("expected one record warm, got %d", archiveRequests.Load())
	}
	closeContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Close(closeContext); err != nil {
		t.Fatal(err)
	}
}

func TestServiceFallsBackToOpenMeteoOnNOAA404(t *testing.T) {
	fixture := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/forecast":
			_, _ = response.Write([]byte(`{
				"timezone":"UTC",
				"current":{"time":1785144000,"temperature_2m":61,"relative_humidity_2m":82,"weather_code":3,"wind_speed_10m":5,"wind_direction_10m":315},
				"hourly":{"time":[1785144000],"temperature_2m":[61],"relative_humidity_2m":[82],"precipitation_probability":[2],"precipitation":[0],"weather_code":[3],"wind_speed_10m":[5],"wind_direction_10m":[315]},
				"daily":{"time":[1785110400],"weather_code":[3],"temperature_2m_max":[79],"temperature_2m_min":[58],"precipitation_probability_max":[4]}
			}`))
		case request.URL.Path == "/archive":
			_, _ = response.Write([]byte(`{"daily":{"time":["1950-07-27"],"temperature_2m_max":[80],"temperature_2m_min":[50]}}`))
		default:
			http.Error(response, `{"error":"not found"}`, http.StatusNotFound)
		}
	}))
	defer fixture.Close()

	httpClient := &HTTPClient{Client: fixture.Client(), Retries: 0}
	noaa := NewNOAA(httpClient)
	noaa.BaseURL = fixture.URL
	openMeteo := NewOpenMeteo(httpClient)
	openMeteo.ForecastURL = fixture.URL + "/forecast"
	records := NewRecordsClient(httpClient)
	records.ArchiveURL = fixture.URL + "/archive"
	service := NewService(store.NewMemory(10), noaa, openMeteo, records, nil)

	report, err := service.Report(context.Background(), Location{
		Name: "Outside coverage", CountryCode: "US", TimeZone: "UTC",
		Latitude: 0, Longitude: -160,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Provider != "Open-Meteo" {
		t.Fatalf("got provider %q, want Open-Meteo", report.Provider)
	}
	closeContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Close(closeContext); err != nil {
		t.Fatal(err)
	}
}
