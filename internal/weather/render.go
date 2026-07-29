package weather

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const separator = "============================================================"

var windSpeedNumber = regexp.MustCompile(`\d+(?:\.\d+)?`)

func RenderText(report WeatherReport, now time.Time) string {
	timezone := loadTimezone(report.Location.TimeZone)
	localNow := now.In(timezone)
	metric := usesMetricUnits(report)
	var output strings.Builder

	fmt.Fprintf(&output, "Weather for %s\n", truncate(displayLocation(report.Location), 68))
	fmt.Fprintf(
		&output,
		"%s / %s\n\n",
		localNow.Format("Monday, Jan 2"),
		formatObservationTime(report.Current, timezone),
	)

	output.WriteString("Current conditions\n")
	fmt.Fprintf(&output, "temp       %s\n", formatTemperature(report.Current.Temperature, metric))
	fmt.Fprintf(&output, "sky        %s\n", truncate(report.Current.Sky, 65))
	fmt.Fprintf(&output, "humidity   %s\n", formatPercent(report.Current.Humidity))
	fmt.Fprintf(
		&output,
		"wind       %s\n\n",
		formatDisplayWind(firstNonEmpty(report.Current.Wind, "—"), metric),
	)

	output.WriteString(separator + "\n")
	fmt.Fprintf(&output, "%s\n\n", localNow.Format("Monday, Jan 2"))
	output.WriteString("   time     sky                       temp   humid  rain*\n")

	hours := displayHours(report.Hourly, localNow, timezone)
	markerHour := closestHour(localNow)
	for _, hour := range hours {
		marker := " "
		if hour.StartTime.In(timezone).Hour() == markerHour {
			marker = ">"
		}
		fmt.Fprintf(
			&output,
			"%s  %-7s  %-24s %5s  %6s  %6s\n",
			marker,
			hour.StartTime.In(timezone).Format("3 PM"),
			truncate(hour.Sky, 24),
			formatTemperature(hour.Temperature, metric),
			formatPercent(hour.Humidity),
			formatHourlyRain(hour, metric),
		)
	}
	if len(hours) == 0 {
		output.WriteString("   hourly data unavailable\n")
	}

	fmt.Fprintf(
		&output,
		"\n* past = observed 1h rainfall (%s)\n",
		map[bool]string{false: "in.", true: "mm"}[metric],
	)
	output.WriteString("  future = precipitation chance\n")
	switch {
	case report.Provider == "NOAA" && report.StationID != "":
		fmt.Fprintf(&output, "  earlier observations use station %s\n", report.StationID)
	case report.Provider == "Open-Meteo":
		output.WriteString("  past hours use Open-Meteo model estimates\n")
	}

	output.WriteString(separator + "\n")
	output.WriteString("7-day forecast\n\n")
	output.WriteString("day       sky                   high   low  rec hi  rec lo rec avg  rain\n")
	for _, daily := range report.Daily {
		highRecord, lowRecord, average := "—", "—", "—"
		if daily.Record != nil {
			if daily.Record.High != nil {
				highRecord = formatTemperature(&daily.Record.High.Temperature, metric)
			}
			if daily.Record.Low != nil {
				lowRecord = formatTemperature(&daily.Record.Low.Temperature, metric)
			}
			if daily.Record.Average != nil {
				average = formatTemperature(&daily.Record.Average.Temperature, metric)
			}
		}
		fmt.Fprintf(
			&output,
			"%-9s %-20s %5s %5s %7s %7s %7s %5s\n",
			daily.Date.In(timezone).Format("Mon 1/2"),
			truncate(daily.Sky, 20),
			formatTemperature(daily.High, metric),
			formatTemperature(daily.Low, metric),
			highRecord,
			lowRecord,
			average,
			formatPercent(daily.PrecipChance),
		)
	}

	output.WriteString("\n")
	fmt.Fprintf(
		&output,
		"location: %.3f, %.3f / %s",
		report.Location.Latitude,
		report.Location.Longitude,
		firstNonEmpty(report.Location.TimeZone, "UTC"),
	)
	output.WriteString("\n")
	if report.LocationFromIP {
		output.WriteString("geolocation: MaxMind GeoLite2 City (IP estimate)\n")
	}
	if report.Location.Source == "OpenStreetMap Nominatim" {
		output.WriteString("geocoding: © OpenStreetMap contributors / openstreetmap.org/copyright\n")
	}
	fmt.Fprintf(&output, "data: %s\n", report.Provider)
	if report.StationID != "" {
		fmt.Fprintf(&output, "history: station %s\n", report.StationID)
	}
	switch report.RecordsState {
	case "ready":
		fmt.Fprintf(&output, "records: %s\n", truncate(report.RecordsSource, 71))
	case "warming":
		output.WriteString("records: warming; retry shortly\n")
	default:
		output.WriteString("records: unavailable\n")
	}
	output.WriteString("service: wthrtxt.com\n")

	return output.String()
}

func displayHours(readings []HourlyReading, localNow time.Time, timezone *time.Location) []HourlyReading {
	today := localNow.Format("2006-01-02")
	result := make([]HourlyReading, 0, 18)
	for _, reading := range readings {
		local := reading.StartTime.In(timezone)
		if local.Format("2006-01-02") == today && local.Hour() >= 5 && local.Hour() <= 22 {
			result = append(result, reading)
		}
	}
	return result
}

func closestHour(localNow time.Time) int {
	fractional := float64(localNow.Hour()) + float64(localNow.Minute())/60
	hour := int(math.Floor(fractional + 0.5))
	if hour < 5 {
		return 5
	}
	if hour > 22 {
		return 22
	}
	return hour
}

func formatObservationTime(current CurrentConditions, timezone *time.Location) string {
	if current.ObservedAt.IsZero() {
		return "update time unavailable"
	}
	label := firstNonEmpty(current.Source, "observed")
	return fmt.Sprintf("%s %s", label, current.ObservedAt.In(timezone).Format("3:04 PM"))
}

func usesMetricUnits(report WeatherReport) bool {
	countryCode := strings.TrimSpace(report.Location.CountryCode)
	if countryCode != "" {
		return !isNOAACountry(countryCode)
	}
	return report.Provider == "Open-Meteo"
}

func formatTemperature(value *float64, metric bool) string {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return "—"
	}
	if metric {
		return fmt.Sprintf("%.0f°C", (*value-32)*5/9)
	}
	return fmt.Sprintf("%.0f°F", *value)
}

func formatPercent(value *float64) string {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return "—"
	}
	return fmt.Sprintf("%.0f%%", *value)
}

func formatHourlyRain(reading HourlyReading, metric bool) string {
	if !reading.Observed {
		return formatPercent(reading.PrecipChance)
	}
	if reading.PrecipInches == nil || *reading.PrecipInches <= 0 {
		return "—"
	}
	if metric {
		millimeters := *reading.PrecipInches * 25.4
		if millimeters < 0.1 {
			return "<0.1mm"
		}
		return fmt.Sprintf("%.1fmm", millimeters)
	}
	if *reading.PrecipInches < 0.01 {
		return "<.01in"
	}
	return fmt.Sprintf("%.2fin", *reading.PrecipInches)
}

func formatDisplayWind(value string, metric bool) string {
	if !metric || !strings.Contains(strings.ToLower(value), "mph") {
		return value
	}
	converted := windSpeedNumber.ReplaceAllStringFunc(value, func(speed string) string {
		milesPerHour, err := strconv.ParseFloat(speed, 64)
		if err != nil {
			return speed
		}
		return fmt.Sprintf("%.0f", milesPerHour*1.609344)
	})
	return strings.ReplaceAll(converted, "mph", "km/h")
}

func displayLocation(location Location) string {
	if location.Name == "" {
		return fmt.Sprintf("%.3f, %.3f", location.Latitude, location.Longitude)
	}
	if location.Region != "" {
		return location.Name + ", " + location.Region
	}
	if location.Country != "" {
		return location.Name + ", " + location.Country
	}
	return location.Name
}

func truncate(value string, width int) string {
	value = strings.Join(strings.Fields(firstNonEmpty(value, "Conditions unavailable")), " ")
	if utf8.RuneCountInString(value) <= width {
		return value
	}
	if width <= 1 {
		return "…"
	}
	runes := []rune(value)
	return string(runes[:width-1]) + "…"
}
