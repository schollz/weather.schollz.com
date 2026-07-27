package weather

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
	_ "time/tzdata"

	"golang.org/x/sync/singleflight"

	"wthrtxt.com/internal/store"
)

type Service struct {
	Cache   *store.Store
	NOAA    *NOAA
	Open    *OpenMeteo
	Records *RecordsClient
	Logger  *slog.Logger
	Now     func() time.Time

	weatherGroup    singleflight.Group
	recordGroup     singleflight.Group
	upstreams       chan struct{}
	recordUpstreams chan struct{}
	failuresMu      sync.Mutex
	failures        map[string]time.Time
	background      sync.WaitGroup
	backgroundCtx   context.Context
	cancel          context.CancelFunc
}

func NewService(
	cache *store.Store,
	noaa *NOAA,
	openMeteo *OpenMeteo,
	records *RecordsClient,
	logger *slog.Logger,
) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	backgroundContext, cancel := context.WithCancel(context.Background())
	return &Service{
		Cache:           cache,
		NOAA:            noaa,
		Open:            openMeteo,
		Records:         records,
		Logger:          logger,
		Now:             time.Now,
		upstreams:       make(chan struct{}, 8),
		recordUpstreams: make(chan struct{}, 2),
		failures:        make(map[string]time.Time),
		backgroundCtx:   backgroundContext,
		cancel:          cancel,
	}
}

func (s *Service) Report(ctx context.Context, location Location) (WeatherReport, error) {
	cacheKey := fmt.Sprintf(
		"weather:%s:%.3f,%.3f",
		location.CountryCode,
		location.Latitude,
		location.Longitude,
	)
	var cached WeatherReport
	if s.Cache != nil && s.Cache.Get(cacheKey, &cached) {
		s.attachRecords(&cached)
		return cached, nil
	}

	value, err, _ := s.weatherGroup.Do(cacheKey, func() (any, error) {
		if s.Cache != nil && s.Cache.Get(cacheKey, &cached) {
			return cached, nil
		}

		select {
		case s.upstreams <- struct{}{}:
			defer func() { <-s.upstreams }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		report, fetchErr := s.fetch(ctx, location)
		if fetchErr != nil {
			return nil, fetchErr
		}
		if s.Cache != nil {
			_ = s.Cache.Put(cacheKey, report, s.Now().Add(5*time.Minute))
		}
		return report, nil
	})
	if err != nil {
		return WeatherReport{}, err
	}

	report := value.(WeatherReport)
	s.attachRecords(&report)
	return report, nil
}

func (s *Service) fetch(ctx context.Context, location Location) (WeatherReport, error) {
	if location.CountryCode == "" || isNOAACountry(location.CountryCode) {
		report, err := s.NOAA.Fetch(ctx, location)
		if err == nil {
			return report, nil
		}
		if !IsStatus(err, 404) {
			return WeatherReport{}, err
		}
	}
	return s.Open.Fetch(ctx, location)
}

func (s *Service) attachRecords(report *WeatherReport) {
	year := lastCompletedYear(s.Now(), report.Location.TimeZone)
	var key string
	if report.Provider == "NOAA" && report.StationID != "" {
		key = fmt.Sprintf("records:acis:%s:%d", report.StationID, year)
	} else if report.Provider == "Open-Meteo" {
		key = fmt.Sprintf(
			"records:era5:%.3f,%.3f:%s:%d",
			report.Location.Latitude,
			report.Location.Longitude,
			report.Location.TimeZone,
			year,
		)
	} else {
		report.RecordsState = "unavailable"
		return
	}

	var cached recordsPayload
	if s.Cache != nil && s.Cache.Get(key, &cached) {
		applyRecords(report, cached)
		return
	}
	report.RecordsState = "warming"
	if s.recordRecentlyFailed(key) {
		return
	}

	s.background.Add(1)
	result := s.recordGroup.DoChan(key, func() (any, error) {
		timeout := 2 * time.Minute
		if report.Provider == "Open-Meteo" {
			timeout = 4 * time.Minute
		}
		ctx, cancel := context.WithTimeout(s.backgroundCtx, timeout)
		defer cancel()

		select {
		case s.recordUpstreams <- struct{}{}:
			defer func() { <-s.recordUpstreams }()
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		var payload recordsPayload
		var err error
		if report.Provider == "NOAA" {
			payload, err = s.Records.ACIS(ctx, report.StationID)
		} else {
			payload, err = s.Records.ERA5(ctx, report.Location, year)
		}
		if err != nil {
			s.Logger.Warn("record warm failed", "provider", report.Provider, "error", err)
			s.markRecordFailure(key)
			return nil, err
		}
		if s.Cache != nil {
			err = s.Cache.Put(key, payload, s.Now().Add(400*24*time.Hour))
		}
		return payload, err
	})
	go func() {
		defer s.background.Done()
		<-result
	}()
}

func applyRecords(report *WeatherReport, payload recordsPayload) {
	for index := range report.Daily {
		key := report.Daily[index].Date.In(loadTimezone(report.Location.TimeZone)).Format("01-02")
		if record, found := payload.Records[key]; found {
			recordCopy := record
			report.Daily[index].Record = &recordCopy
		}
	}
	report.RecordsSource = payload.Source
	report.RecordsState = "ready"
}

func lastCompletedYear(now time.Time, timezone string) int {
	return now.In(loadTimezone(timezone)).Year() - 1
}

func loadTimezone(name string) *time.Location {
	if name != "" {
		if location, err := time.LoadLocation(name); err == nil {
			return location
		}
	}
	return time.UTC
}

func (s *Service) recordRecentlyFailed(key string) bool {
	s.failuresMu.Lock()
	defer s.failuresMu.Unlock()
	failedAt, found := s.failures[key]
	return found && s.Now().Sub(failedAt) < 15*time.Minute
}

func (s *Service) markRecordFailure(key string) {
	s.failuresMu.Lock()
	s.failures[key] = s.Now()
	s.failuresMu.Unlock()
}

func (s *Service) Close(ctx context.Context) error {
	s.cancel()
	done := make(chan struct{})
	go func() {
		s.background.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
