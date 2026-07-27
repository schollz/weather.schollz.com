package weather

import "time"

type Location struct {
	Name        string  `json:"name"`
	Region      string  `json:"region"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	TimeZone    string  `json:"time_zone"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Source      string  `json:"source"`
}

type CurrentConditions struct {
	ObservedAt  time.Time `json:"observed_at"`
	Temperature *float64  `json:"temperature_f"`
	Humidity    *float64  `json:"humidity"`
	Sky         string    `json:"sky"`
	Wind        string    `json:"wind"`
	Source      string    `json:"source"`
}

type HourlyReading struct {
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Temperature     *float64  `json:"temperature_f"`
	Humidity        *float64  `json:"humidity"`
	PrecipChance    *float64  `json:"precip_chance"`
	PrecipInches    *float64  `json:"precip_inches"`
	Sky             string    `json:"sky"`
	Observed        bool      `json:"observed"`
	ObservationKind string    `json:"observation_kind"`
}

type ClimateValue struct {
	Date        string  `json:"date"`
	Temperature float64 `json:"temperature_f"`
	Estimated   bool    `json:"estimated"`
	Coverage    string  `json:"coverage,omitempty"`
}

type ClimateRecord struct {
	High *ClimateValue `json:"high,omitempty"`
	Low  *ClimateValue `json:"low,omitempty"`
}

type DailyForecast struct {
	Date         time.Time      `json:"date"`
	High         *float64       `json:"high_f"`
	Low          *float64       `json:"low_f"`
	PrecipChance *float64       `json:"precip_chance"`
	Sky          string         `json:"sky"`
	Record       *ClimateRecord `json:"record,omitempty"`
}

type WeatherReport struct {
	Location       Location          `json:"location"`
	Current        CurrentConditions `json:"current"`
	Hourly         []HourlyReading   `json:"hourly"`
	Daily          []DailyForecast   `json:"daily"`
	Provider       string            `json:"provider"`
	StationID      string            `json:"station_id,omitempty"`
	RecordsSource  string            `json:"records_source,omitempty"`
	RecordsState   string            `json:"records_state,omitempty"`
	LocationFromIP bool              `json:"location_from_ip,omitempty"`
}

func floatPointer(value float64) *float64 {
	return &value
}
