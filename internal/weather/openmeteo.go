package weather

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultOpenMeteoForecastURL = "https://api.open-meteo.com/v1/forecast"

type OpenMeteo struct {
	HTTP        *HTTPClient
	ForecastURL string
}

func NewOpenMeteo(client *HTTPClient) *OpenMeteo {
	return &OpenMeteo{HTTP: client, ForecastURL: defaultOpenMeteoForecastURL}
}

type openMeteoForecast struct {
	Timezone string `json:"timezone"`
	Current  struct {
		Time          int64   `json:"time"`
		Temperature   float64 `json:"temperature_2m"`
		Humidity      float64 `json:"relative_humidity_2m"`
		WeatherCode   int     `json:"weather_code"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindDirection float64 `json:"wind_direction_10m"`
	} `json:"current"`
	Hourly struct {
		Time              []int64    `json:"time"`
		Temperature       []*float64 `json:"temperature_2m"`
		Humidity          []*float64 `json:"relative_humidity_2m"`
		PrecipProbability []*float64 `json:"precipitation_probability"`
		Precipitation     []*float64 `json:"precipitation"`
		WeatherCode       []int      `json:"weather_code"`
		WindSpeed         []*float64 `json:"wind_speed_10m"`
		WindDirection     []*float64 `json:"wind_direction_10m"`
	} `json:"hourly"`
	Daily struct {
		Time              []int64    `json:"time"`
		WeatherCode       []int      `json:"weather_code"`
		TemperatureMax    []*float64 `json:"temperature_2m_max"`
		TemperatureMin    []*float64 `json:"temperature_2m_min"`
		PrecipProbability []*float64 `json:"precipitation_probability_max"`
	} `json:"daily"`
}

func (o *OpenMeteo) Fetch(ctx context.Context, location Location) (WeatherReport, error) {
	endpoint, err := url.Parse(o.ForecastURL)
	if err != nil {
		return WeatherReport{}, err
	}
	query := endpoint.Query()
	query.Set("latitude", strconv.FormatFloat(location.Latitude, 'f', 4, 64))
	query.Set("longitude", strconv.FormatFloat(location.Longitude, 'f', 4, 64))
	query.Set("current", strings.Join([]string{
		"temperature_2m",
		"relative_humidity_2m",
		"is_day",
		"weather_code",
		"wind_speed_10m",
		"wind_direction_10m",
	}, ","))
	query.Set("hourly", strings.Join([]string{
		"temperature_2m",
		"relative_humidity_2m",
		"precipitation_probability",
		"precipitation",
		"weather_code",
		"wind_speed_10m",
		"wind_direction_10m",
		"is_day",
	}, ","))
	query.Set("daily", strings.Join([]string{
		"weather_code",
		"temperature_2m_max",
		"temperature_2m_min",
		"precipitation_probability_max",
	}, ","))
	query.Set("temperature_unit", "fahrenheit")
	query.Set("wind_speed_unit", "mph")
	query.Set("precipitation_unit", "inch")
	query.Set("timezone", "auto")
	query.Set("timeformat", "unixtime")
	query.Set("forecast_days", "7")
	endpoint.RawQuery = query.Encode()

	var response openMeteoForecast
	if err := o.HTTP.JSON(ctx, "Open-Meteo", "GET", endpoint.String(), "", nil, &response); err != nil {
		return WeatherReport{}, err
	}
	if response.Timezone == "" || response.Current.Time == 0 ||
		len(response.Hourly.Time) == 0 || len(response.Daily.Time) == 0 {
		return WeatherReport{}, errors.New("Open-Meteo returned an incomplete forecast")
	}
	location.TimeZone = response.Timezone

	report := WeatherReport{
		Location: location,
		Current: CurrentConditions{
			ObservedAt:  time.Unix(response.Current.Time, 0),
			Temperature: floatPointer(response.Current.Temperature),
			Humidity:    floatPointer(response.Current.Humidity),
			Sky:         WeatherCodeDescription(response.Current.WeatherCode),
			Wind:        formatWind(response.Current.WindSpeed, response.Current.WindDirection),
			Source:      "model time",
		},
		Provider: "Open-Meteo",
	}

	for index, timestamp := range response.Hourly.Time {
		if index >= len(response.Hourly.WeatherCode) {
			break
		}
		start := time.Unix(timestamp, 0)
		reading := HourlyReading{
			StartTime:       start,
			EndTime:         start.Add(time.Hour),
			Temperature:     valueAt(response.Hourly.Temperature, index),
			Humidity:        valueAt(response.Hourly.Humidity, index),
			PrecipChance:    valueAt(response.Hourly.PrecipProbability, index),
			PrecipInches:    valueAt(response.Hourly.Precipitation, index),
			Sky:             WeatherCodeDescription(response.Hourly.WeatherCode[index]),
			Observed:        timestamp <= response.Current.Time,
			ObservationKind: "estimated",
		}
		report.Hourly = append(report.Hourly, reading)
	}

	for index, timestamp := range response.Daily.Time {
		if index >= len(response.Daily.WeatherCode) {
			break
		}
		report.Daily = append(report.Daily, DailyForecast{
			Date:         time.Unix(timestamp, 0),
			High:         valueAt(response.Daily.TemperatureMax, index),
			Low:          valueAt(response.Daily.TemperatureMin, index),
			PrecipChance: valueAt(response.Daily.PrecipProbability, index),
			Sky:          WeatherCodeDescription(response.Daily.WeatherCode[index]),
		})
	}

	return report, nil
}

func valueAt(values []*float64, index int) *float64 {
	if index < 0 || index >= len(values) || values[index] == nil {
		return nil
	}
	value := *values[index]
	return &value
}

func formatWind(speed, degrees float64) string {
	return fmt.Sprintf("%.0f mph %s", speed, degreesToCompass(degrees))
}

func degreesToCompass(degrees float64) string {
	directions := []string{
		"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW",
	}
	index := int(degrees/22.5+0.5) % len(directions)
	if index < 0 {
		index += len(directions)
	}
	return directions[index]
}

func WeatherCodeDescription(code int) string {
	descriptions := map[int]string{
		0: "Clear sky", 1: "Mainly clear", 2: "Partly cloudy", 3: "Overcast",
		45: "Fog", 48: "Rime fog", 51: "Light drizzle", 53: "Moderate drizzle",
		55: "Dense drizzle", 56: "Light freezing drizzle", 57: "Dense freezing drizzle",
		61: "Light rain", 63: "Moderate rain", 65: "Heavy rain",
		66: "Light freezing rain", 67: "Heavy freezing rain",
		71: "Light snow", 73: "Moderate snow", 75: "Heavy snow", 77: "Snow grains",
		80: "Light rain showers", 81: "Moderate rain showers", 82: "Violent rain showers",
		85: "Light snow showers", 86: "Heavy snow showers", 95: "Thunderstorms",
		96: "Thunderstorms with light hail", 99: "Thunderstorms with heavy hail",
	}
	if description := descriptions[code]; description != "" {
		return description
	}
	return "Conditions unavailable"
}
