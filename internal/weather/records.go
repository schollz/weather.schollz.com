package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultACISURL             = "https://data.rcc-acis.org/StnData"
	defaultOpenMeteoArchiveURL = "https://archive-api.open-meteo.com/v1/archive"
	recordStartYear            = 1950
)

type RecordsClient struct {
	HTTP       *HTTPClient
	ACISURL    string
	ArchiveURL string
}

func NewRecordsClient(client *HTTPClient) *RecordsClient {
	return &RecordsClient{
		HTTP:       client,
		ACISURL:    defaultACISURL,
		ArchiveURL: defaultOpenMeteoArchiveURL,
	}
}

type recordsPayload struct {
	Records map[string]ClimateRecord `json:"records"`
	Source  string                   `json:"source"`
}

func (r *RecordsClient) ACIS(ctx context.Context, stationID string) (recordsPayload, error) {
	parameters := map[string]any{
		"sid":   stationID,
		"sdate": "por",
		"edate": "por",
		"meta":  []string{"name", "state"},
		"elems": []map[string]any{
			{
				"name": "maxt", "interval": "dly", "duration": "dly",
				"smry":      map[string]string{"reduce": "max", "add": "date"},
				"smry_only": 1, "groupby": "year",
			},
			{
				"name": "mint", "interval": "dly", "duration": "dly",
				"smry":      map[string]string{"reduce": "min", "add": "date"},
				"smry_only": 1, "groupby": "year",
			},
			{
				"name": "avgt", "interval": "dly", "duration": "dly",
				"smry": "mean", "smry_only": 1, "groupby": "year",
			},
		},
	}
	encodedParameters, err := json.Marshal(parameters)
	if err != nil {
		return recordsPayload{}, err
	}
	body := url.Values{"params": {string(encodedParameters)}}.Encode()

	var response struct {
		Error string            `json:"error"`
		Meta  map[string]any    `json:"meta"`
		Smry  []json.RawMessage `json:"smry"`
	}
	if err := r.HTTP.JSON(
		ctx,
		"ACIS",
		"POST",
		r.ACISURL,
		"application/x-www-form-urlencoded",
		[]byte(body),
		&response,
	); err != nil {
		return recordsPayload{}, err
	}
	if response.Error != "" {
		return recordsPayload{}, errors.New(response.Error)
	}

	records := make(map[string]ClimateRecord)
	if len(response.Smry) > 0 {
		addACISSummaries(records, response.Smry[0], true)
	}
	if len(response.Smry) > 1 {
		addACISSummaries(records, response.Smry[1], false)
	}
	if len(response.Smry) > 2 {
		addACISAverages(records, response.Smry[2])
	}
	if len(records) == 0 {
		return recordsPayload{}, errors.New("ACIS returned no usable records")
	}

	stationName, _ := response.Meta["name"].(string)
	source := "ACIS / " + firstNonEmpty(stationName, stationID)
	return recordsPayload{Records: records, Source: source}, nil
}

func addACISSummaries(records map[string]ClimateRecord, raw json.RawMessage, high bool) {
	var summaries []json.RawMessage
	if json.Unmarshal(raw, &summaries) != nil {
		return
	}
	for _, rawSummary := range summaries {
		var summary []json.RawMessage
		if json.Unmarshal(rawSummary, &summary) != nil || len(summary) < 2 {
			continue
		}

		var temperatureValue any
		if json.Unmarshal(summary[0], &temperatureValue) != nil {
			continue
		}
		temperature, ok := numberFromJSON(temperatureValue)
		if !ok {
			continue
		}
		var date string
		if json.Unmarshal(summary[1], &date) != nil || len(date) != 10 {
			continue
		}
		if _, err := time.Parse("2006-01-02", date); err != nil {
			continue
		}

		key := date[5:]
		record := records[key]
		value := &ClimateValue{Date: date, Temperature: temperature}
		if high {
			if record.High == nil || value.Temperature > record.High.Temperature {
				record.High = value
			}
		} else if record.Low == nil || value.Temperature < record.Low.Temperature {
			record.Low = value
		}
		records[key] = record
	}
}

func addACISAverages(records map[string]ClimateRecord, raw json.RawMessage) {
	var summaries []json.RawMessage
	if json.Unmarshal(raw, &summaries) != nil {
		return
	}
	for index, rawSummary := range summaries {
		if index >= 366 {
			break
		}
		var temperatureValue any
		if json.Unmarshal(rawSummary, &temperatureValue) != nil {
			continue
		}
		temperature, ok := numberFromJSON(temperatureValue)
		if !ok {
			continue
		}

		date := time.Date(2000, time.January, 1+index, 0, 0, 0, 0, time.UTC)
		key := date.Format("01-02")
		record := records[key]
		record.Average = &ClimateValue{Temperature: temperature}
		records[key] = record
	}
}

func numberFromJSON(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case string:
		number, err := strconv.ParseFloat(typed, 64)
		return number, err == nil
	default:
		return 0, false
	}
}

func (r *RecordsClient) ERA5(
	ctx context.Context,
	location Location,
	throughYear int,
) (recordsPayload, error) {
	endpoint, err := url.Parse(r.ArchiveURL)
	if err != nil {
		return recordsPayload{}, err
	}
	query := endpoint.Query()
	query.Set("latitude", fmt.Sprintf("%.4f", location.Latitude))
	query.Set("longitude", fmt.Sprintf("%.4f", location.Longitude))
	query.Set("start_date", fmt.Sprintf("%d-01-01", recordStartYear))
	query.Set("end_date", fmt.Sprintf("%d-12-31", throughYear))
	query.Set("daily", "temperature_2m_max,temperature_2m_min")
	query.Set("temperature_unit", "fahrenheit")
	query.Set("timezone", location.TimeZone)
	query.Set("models", "era5_land")
	endpoint.RawQuery = query.Encode()

	var response struct {
		Error  bool   `json:"error"`
		Reason string `json:"reason"`
		Daily  struct {
			Time           []string   `json:"time"`
			TemperatureMax []*float64 `json:"temperature_2m_max"`
			TemperatureMin []*float64 `json:"temperature_2m_min"`
		} `json:"daily"`
	}
	if err := r.HTTP.JSON(ctx, "Open-Meteo archive", "GET", endpoint.String(), "", nil, &response); err != nil {
		return recordsPayload{}, err
	}
	if response.Error || len(response.Daily.Time) == 0 {
		return recordsPayload{}, errors.New(firstNonEmpty(response.Reason, "Open-Meteo archive returned no data"))
	}

	coverage := fmt.Sprintf("%d–%d", recordStartYear, throughYear)
	records := make(map[string]ClimateRecord)
	type averageAccumulator struct {
		count int
		sum   float64
	}
	averages := make(map[string]averageAccumulator)
	for index, date := range response.Daily.Time {
		if len(date) != 10 {
			continue
		}
		key := date[5:]
		record := records[key]
		high := valueAt(response.Daily.TemperatureMax, index)
		low := valueAt(response.Daily.TemperatureMin, index)
		if high != nil && (record.High == nil || *high > record.High.Temperature) {
			record.High = &ClimateValue{
				Date: date, Temperature: *high, Estimated: true, Coverage: coverage,
			}
		}
		if low != nil && (record.Low == nil || *low < record.Low.Temperature) {
			record.Low = &ClimateValue{
				Date: date, Temperature: *low, Estimated: true, Coverage: coverage,
			}
		}
		if high != nil && low != nil {
			average := averages[key]
			average.sum += (*high + *low) / 2
			average.count++
			averages[key] = average
		}
		records[key] = record
	}
	for key, average := range averages {
		if average.count == 0 {
			continue
		}
		record := records[key]
		record.Average = &ClimateValue{
			Temperature: average.sum / float64(average.count),
			Estimated:   true,
			Coverage:    coverage,
		}
		records[key] = record
	}
	if len(records) == 0 {
		return recordsPayload{}, errors.New("Open-Meteo archive returned no usable records")
	}
	return recordsPayload{
		Records: records,
		Source:  fmt.Sprintf("Open-Meteo ERA5-Land (estimated, %s)", coverage),
	}, nil
}

func isNOAACountry(countryCode string) bool {
	switch strings.ToUpper(countryCode) {
	case "AS", "GU", "MP", "PR", "US", "VI":
		return true
	default:
		return false
	}
}
