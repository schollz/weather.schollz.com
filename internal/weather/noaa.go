package weather

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

const defaultNOAABaseURL = "https://api.weather.gov"

type NOAA struct {
	HTTP    *HTTPClient
	BaseURL string
	Now     func() time.Time
}

func NewNOAA(client *HTTPClient) *NOAA {
	return &NOAA{HTTP: client, BaseURL: defaultNOAABaseURL, Now: time.Now}
}

type noaaValue struct {
	UnitCode string   `json:"unitCode"`
	Value    *float64 `json:"value"`
}

type noaaPointResponse struct {
	Properties struct {
		Forecast            string `json:"forecast"`
		ForecastGridData    string `json:"forecastGridData"`
		ForecastHourly      string `json:"forecastHourly"`
		ObservationStations string `json:"observationStations"`
		TimeZone            string `json:"timeZone"`
		RelativeLocation    struct {
			Properties struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"properties"`
		} `json:"relativeLocation"`
	} `json:"properties"`
}

type noaaPeriod struct {
	EndTime                    string    `json:"endTime"`
	IsDaytime                  bool      `json:"isDaytime"`
	ProbabilityOfPrecipitation noaaValue `json:"probabilityOfPrecipitation"`
	RelativeHumidity           noaaValue `json:"relativeHumidity"`
	ShortForecast              string    `json:"shortForecast"`
	StartTime                  string    `json:"startTime"`
	Temperature                *float64  `json:"temperature"`
	TemperatureUnit            string    `json:"temperatureUnit"`
	WindDirection              string    `json:"windDirection"`
	WindSpeed                  string    `json:"windSpeed"`
}

type noaaForecastResponse struct {
	Properties struct {
		Periods []noaaPeriod `json:"periods"`
	} `json:"properties"`
}

type noaaObservation struct {
	PrecipitationLastHour noaaValue `json:"precipitationLastHour"`
	RawMessage            string    `json:"rawMessage"`
	RelativeHumidity      noaaValue `json:"relativeHumidity"`
	Temperature           noaaValue `json:"temperature"`
	TextDescription       string    `json:"textDescription"`
	Timestamp             string    `json:"timestamp"`
}

type noaaStationsResponse struct {
	Features []struct {
		ID string `json:"id"`
	} `json:"features"`
}

type noaaObservationsResponse struct {
	Features []struct {
		Properties noaaObservation `json:"properties"`
	} `json:"features"`
}

type noaaGridResponse struct {
	Properties struct {
		QuantitativePrecipitation struct {
			UOM    string `json:"uom"`
			Values []struct {
				ValidTime string   `json:"validTime"`
				Value     *float64 `json:"value"`
			} `json:"values"`
		} `json:"quantitativePrecipitation"`
	} `json:"properties"`
}

type precipitationInterval struct {
	Start time.Time
	End   time.Time
	Total float64
}

func (n *NOAA) Fetch(ctx context.Context, location Location) (WeatherReport, error) {
	var point noaaPointResponse
	pointURL := fmt.Sprintf(
		"%s/points/%.4f,%.4f",
		strings.TrimRight(n.BaseURL, "/"),
		location.Latitude,
		location.Longitude,
	)
	if err := n.HTTP.JSON(ctx, "NOAA", "GET", pointURL, "", nil, &point); err != nil {
		return WeatherReport{}, err
	}

	var hourlyResponse, dailyResponse noaaForecastResponse
	var observations []noaaObservation
	var stationID string
	var grid noaaGridResponse

	group, groupContext := errgroup.WithContext(ctx)
	group.Go(func() error {
		return n.HTTP.JSON(groupContext, "NOAA", "GET", point.Properties.ForecastHourly, "", nil, &hourlyResponse)
	})
	group.Go(func() error {
		return n.HTTP.JSON(groupContext, "NOAA", "GET", point.Properties.Forecast, "", nil, &dailyResponse)
	})
	group.Go(func() error {
		var err error
		observations, stationID, err = n.fetchObservations(groupContext, point.Properties.ObservationStations)
		if err != nil {
			observations = nil
			stationID = ""
		}
		return nil
	})
	group.Go(func() error {
		if point.Properties.ForecastGridData == "" {
			return nil
		}
		if err := n.HTTP.JSON(groupContext, "NOAA", "GET", point.Properties.ForecastGridData, "", nil, &grid); err != nil {
			grid = noaaGridResponse{}
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return WeatherReport{}, err
	}
	if len(hourlyResponse.Properties.Periods) == 0 || len(dailyResponse.Properties.Periods) == 0 {
		return WeatherReport{}, fmt.Errorf("NOAA returned an incomplete forecast")
	}

	location.TimeZone = point.Properties.TimeZone
	location.Name = firstNonEmpty(point.Properties.RelativeLocation.Properties.City, location.Name)
	location.Region = firstNonEmpty(point.Properties.RelativeLocation.Properties.State, location.Region)
	if location.CountryCode == "" {
		location.CountryCode = "US"
	}
	if location.Country == "" {
		location.Country = "United States"
	}

	precipitation := normalizePrecipitation(grid.Properties.QuantitativePrecipitation)
	report := WeatherReport{
		Location:  location,
		Provider:  "NOAA",
		StationID: stationID,
	}

	forecastHours := make([]HourlyReading, 0, len(hourlyResponse.Properties.Periods))
	for _, period := range hourlyResponse.Properties.Periods {
		start, startErr := time.Parse(time.RFC3339, period.StartTime)
		end, endErr := time.Parse(time.RFC3339, period.EndTime)
		if startErr != nil || endErr != nil {
			continue
		}
		forecastHours = append(forecastHours, HourlyReading{
			StartTime:    start,
			EndTime:      end,
			Temperature:  temperatureF(period.Temperature, period.TemperatureUnit),
			Humidity:     copyFloat(period.RelativeHumidity.Value),
			PrecipChance: copyFloat(period.ProbabilityOfPrecipitation.Value),
			PrecipInches: forecastPrecipitation(precipitation, start, end),
			Sky:          firstNonEmpty(period.ShortForecast, "Conditions unavailable"),
		})
	}

	timezone, err := time.LoadLocation(location.TimeZone)
	if err != nil {
		timezone = time.UTC
	}
	observationHours := preferredObservationsByHour(observations)
	now := n.Now()
	for _, reading := range forecastHours {
		key := reading.StartTime.UTC().Format("2006-01-02-15")
		if observation, found := observationHours[key]; found && !reading.StartTime.After(now) {
			reading.Observed = true
			reading.ObservationKind = "station"
			reading.Temperature = observationTemperature(observation)
			if observation.RelativeHumidity.Value != nil {
				reading.Humidity = copyFloat(observation.RelativeHumidity.Value)
			}
			reading.PrecipInches = observationPrecipitation(observation)
			reading.Sky = firstNonEmpty(observation.TextDescription, reading.Sky)
		}
		report.Hourly = append(report.Hourly, reading)
	}

	report.Daily = aggregateNOAADaily(dailyResponse.Properties.Periods, timezone)
	report.Current = currentNOAAConditions(
		observations,
		forecastHours,
		hourlyResponse.Properties.Periods,
	)
	return report, nil
}

func (n *NOAA) fetchObservations(ctx context.Context, stationsURL string) ([]noaaObservation, string, error) {
	if stationsURL == "" {
		return nil, "", nil
	}
	var stations noaaStationsResponse
	if err := n.HTTP.JSON(ctx, "NOAA", "GET", stationsURL, "", nil, &stations); err != nil {
		return nil, "", err
	}
	if len(stations.Features) == 0 || stations.Features[0].ID == "" {
		return nil, "", nil
	}

	stationURL := strings.TrimRight(stations.Features[0].ID, "/")
	end := n.Now().UTC()
	start := end.Add(-30 * time.Hour)
	observationsURL := fmt.Sprintf(
		"%s/observations?start=%s&end=%s&limit=500",
		stationURL,
		urlQueryEscape(start.Format(time.RFC3339)),
		urlQueryEscape(end.Format(time.RFC3339)),
	)
	var response noaaObservationsResponse
	if err := n.HTTP.JSON(ctx, "NOAA", "GET", observationsURL, "", nil, &response); err != nil {
		return nil, "", err
	}

	result := make([]noaaObservation, 0, len(response.Features))
	for _, feature := range response.Features {
		result = append(result, feature.Properties)
	}
	parts := strings.Split(stationURL, "/")
	return result, parts[len(parts)-1], nil
}

func aggregateNOAADaily(periods []noaaPeriod, timezone *time.Location) []DailyForecast {
	byDate := make(map[string]*DailyForecast)
	order := make([]string, 0, 7)

	for _, period := range periods {
		start, err := time.Parse(time.RFC3339, period.StartTime)
		if err != nil {
			continue
		}
		key := start.In(timezone).Format("2006-01-02")
		daily := byDate[key]
		if daily == nil {
			localDate, parseErr := time.ParseInLocation("2006-01-02", key, timezone)
			if parseErr != nil {
				continue
			}
			daily = &DailyForecast{Date: localDate}
			byDate[key] = daily
			order = append(order, key)
		}
		if period.IsDaytime {
			daily.High = temperatureF(period.Temperature, period.TemperatureUnit)
			daily.Sky = firstNonEmpty(period.ShortForecast, daily.Sky)
		} else {
			daily.Low = temperatureF(period.Temperature, period.TemperatureUnit)
			if daily.Sky == "" {
				daily.Sky = period.ShortForecast
			}
		}
		if value := period.ProbabilityOfPrecipitation.Value; value != nil &&
			(daily.PrecipChance == nil || *value > *daily.PrecipChance) {
			daily.PrecipChance = copyFloat(value)
		}
	}

	result := make([]DailyForecast, 0, min(7, len(order)))
	for _, key := range order {
		if len(result) == 7 {
			break
		}
		result = append(result, *byDate[key])
	}
	return result
}

func currentNOAAConditions(
	observations []noaaObservation,
	forecasts []HourlyReading,
	periods []noaaPeriod,
) CurrentConditions {
	sort.SliceStable(observations, func(left, right int) bool {
		leftTime, _ := time.Parse(time.RFC3339, observations[left].Timestamp)
		rightTime, _ := time.Parse(time.RFC3339, observations[right].Timestamp)
		return leftTime.After(rightTime)
	})

	var fallback HourlyReading
	if len(forecasts) > 0 {
		fallback = forecasts[0]
	}
	current := CurrentConditions{
		ObservedAt:  fallback.StartTime,
		Temperature: fallback.Temperature,
		Humidity:    fallback.Humidity,
		Sky:         firstNonEmpty(fallback.Sky, "Conditions unavailable"),
		Wind:        "—",
		Source:      "forecast",
	}
	if len(observations) > 0 {
		observation := observations[0]
		if timestamp, err := time.Parse(time.RFC3339, observation.Timestamp); err == nil {
			current.ObservedAt = timestamp
		}
		if value := observationTemperature(observation); value != nil {
			current.Temperature = value
		}
		if observation.RelativeHumidity.Value != nil {
			current.Humidity = copyFloat(observation.RelativeHumidity.Value)
		}
		current.Sky = firstNonEmpty(observation.TextDescription, current.Sky)
		current.Source = "observed"
	}
	if len(periods) > 0 {
		current.Wind = strings.TrimSpace(strings.Join(
			[]string{periods[0].WindSpeed, periods[0].WindDirection},
			" ",
		))
		if current.Wind == "" {
			current.Wind = "—"
		}
	}
	return current
}

func preferredObservationsByHour(observations []noaaObservation) map[string]noaaObservation {
	result := make(map[string]noaaObservation)
	for _, observation := range observations {
		timestamp, err := time.Parse(time.RFC3339, observation.Timestamp)
		if err != nil {
			continue
		}
		// UTC keys preserve both occurrences of a repeated local DST hour.
		key := timestamp.UTC().Format("2006-01-02-15")
		existing, found := result[key]
		if !found || observationPriority(observation) > observationPriority(existing) ||
			(observationPriority(observation) == observationPriority(existing) &&
				observation.Timestamp > existing.Timestamp) {
			result[key] = observation
		}
	}
	return result
}

func observationPriority(observation noaaObservation) int {
	timestamp, _ := time.Parse(time.RFC3339, observation.Timestamp)
	minute := timestamp.UTC().Minute()
	priority := 0
	if minute >= 51 && minute <= 59 {
		priority += 4
	}
	if observation.RawMessage != "" {
		priority += 2
	}
	if observation.PrecipitationLastHour.Value != nil || metarPrecipitation.MatchString(observation.RawMessage) {
		priority++
	}
	return priority
}

func temperatureF(value *float64, unit string) *float64 {
	if value == nil {
		return nil
	}
	result := *value
	if !strings.Contains(strings.ToLower(unit), "f") {
		result = result*9/5 + 32
	}
	return &result
}

func observationTemperature(observation noaaObservation) *float64 {
	return temperatureF(observation.Temperature.Value, observation.Temperature.UnitCode)
}

var metarPrecipitation = regexp.MustCompile(`\bP(\d{4})\b`)

func observationPrecipitation(observation noaaObservation) *float64 {
	if observation.PrecipitationLastHour.Value != nil {
		value := *observation.PrecipitationLastHour.Value
		unit := strings.ToLower(observation.PrecipitationLastHour.UnitCode)
		switch {
		case strings.HasSuffix(unit, ":in") || unit == "in":
		case strings.HasSuffix(unit, ":mm") || unit == "mm":
			value /= 25.4
		case strings.HasSuffix(unit, ":cm") || unit == "cm":
			value /= 2.54
		default:
			value *= 39.3700787402
		}
		return &value
	}
	match := metarPrecipitation.FindStringSubmatch(observation.RawMessage)
	if len(match) == 2 {
		value, _ := strconv.ParseFloat(match[1], 64)
		value /= 100
		return &value
	}
	if observation.RawMessage != "" {
		return floatPointer(0)
	}
	return nil
}

func normalizePrecipitation(value struct {
	UOM    string `json:"uom"`
	Values []struct {
		ValidTime string   `json:"validTime"`
		Value     *float64 `json:"value"`
	} `json:"values"`
}) []precipitationInterval {
	result := make([]precipitationInterval, 0, len(value.Values))
	for _, item := range value.Values {
		if item.Value == nil || *item.Value < 0 {
			continue
		}
		parts := strings.SplitN(item.ValidTime, "/", 2)
		if len(parts) != 2 {
			continue
		}
		start, err := time.Parse(time.RFC3339, parts[0])
		duration, durationErr := parseISODuration(parts[1])
		total, unitErr := precipitationInches(*item.Value, value.UOM)
		if err != nil || durationErr != nil || unitErr != nil {
			continue
		}
		result = append(result, precipitationInterval{
			Start: start,
			End:   start.Add(duration),
			Total: total,
		})
	}
	return result
}

func forecastPrecipitation(intervals []precipitationInterval, start, end time.Time) *float64 {
	found := false
	total := 0.0
	for _, interval := range intervals {
		overlapStart := start
		if interval.Start.After(overlapStart) {
			overlapStart = interval.Start
		}
		overlapEnd := end
		if interval.End.Before(overlapEnd) {
			overlapEnd = interval.End
		}
		if !overlapEnd.After(overlapStart) {
			continue
		}
		found = true
		total += interval.Total * overlapEnd.Sub(overlapStart).Seconds() / interval.End.Sub(interval.Start).Seconds()
	}
	if !found {
		return nil
	}
	return &total
}

var isoDuration = regexp.MustCompile(`^P(?:(\d+(?:\.\d+)?)D)?(?:T(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)

func parseISODuration(value string) (time.Duration, error) {
	match := isoDuration.FindStringSubmatch(value)
	if len(match) == 0 {
		return 0, fmt.Errorf("invalid ISO duration %q", value)
	}
	totalSeconds := 0.0
	multipliers := []float64{86400, 3600, 60, 1}
	for index, multiplier := range multipliers {
		if match[index+1] == "" {
			continue
		}
		amount, err := strconv.ParseFloat(match[index+1], 64)
		if err != nil {
			return 0, err
		}
		totalSeconds += amount * multiplier
	}
	if totalSeconds <= 0 {
		return 0, fmt.Errorf("empty ISO duration")
	}
	return time.Duration(totalSeconds * float64(time.Second)), nil
}

func precipitationInches(value float64, unit string) (float64, error) {
	unit = strings.ToLower(unit)
	switch {
	case strings.HasSuffix(unit, ":mm") || unit == "mm":
		return value / 25.4, nil
	case strings.HasSuffix(unit, ":cm") || unit == "cm":
		return value / 2.54, nil
	case strings.HasSuffix(unit, ":m") || unit == "m":
		return value * 39.3700787402, nil
	case strings.HasSuffix(unit, ":in") || unit == "in":
		return value, nil
	default:
		return math.NaN(), fmt.Errorf("unsupported precipitation unit %q", unit)
	}
}

func copyFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func urlQueryEscape(value string) string {
	replacer := strings.NewReplacer(":", "%3A", "+", "%2B")
	return replacer.Replace(value)
}
