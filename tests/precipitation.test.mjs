import assert from "node:assert/strict";
import test from "node:test";
import {
  forecastPrecipitationForPeriod,
  formatForecastRainfall,
  formatObservedRainfall,
  formatRainfallInches,
  normalizeNoaaPrecipitation,
  remainingRainfallTotal,
} from "../app/precipitation.mjs";

test("distributes a multi-hour NOAA total evenly across its hours", () => {
  const intervals = normalizeNoaaPrecipitation({
    uom: "wmoUnit:in",
    values: [
      {
        validTime: "2026-07-26T14:00:00-04:00/PT6H",
        value: 6,
      },
    ],
  });

  for (let hour = 14; hour < 20; hour += 1) {
    const start = `2026-07-26T${hour}:00:00-04:00`;
    const end = `2026-07-26T${hour + 1}:00:00-04:00`;
    assert.equal(
      forecastPrecipitationForPeriod(intervals, start, end),
      1,
    );
  }
});

test("converts NOAA millimeters and allocates periods crossing intervals", () => {
  const intervals = normalizeNoaaPrecipitation({
    uom: "wmoUnit:mm",
    values: [
      {
        validTime: "2026-07-26T00:00:00Z/PT2H",
        value: 25.4,
      },
      {
        validTime: "2026-07-26T02:00:00Z/PT2H",
        value: 25.4,
      },
    ],
  });

  assert.equal(
    forecastPrecipitationForPeriod(
      intervals,
      "2026-07-26T01:30:00Z",
      "2026-07-26T02:30:00Z",
    ),
    0.5,
  );
});

test("totals only future forecast hours remaining in the local day", () => {
  const periods = [
    {
      endTime: "2026-07-26T15:00:00-04:00",
      forecastPrecipitationInches: 0.1,
      startTime: "2026-07-26T14:00:00-04:00",
    },
    {
      endTime: "2026-07-26T16:00:00-04:00",
      forecastPrecipitationInches: 0.2,
      startTime: "2026-07-26T15:00:00-04:00",
    },
    {
      endTime: "2026-07-27T00:00:00-04:00",
      forecastPrecipitationInches: 0.3,
      startTime: "2026-07-26T23:00:00-04:00",
    },
    {
      endTime: "2026-07-27T01:00:00-04:00",
      forecastPrecipitationInches: 0.4,
      startTime: "2026-07-27T00:00:00-04:00",
    },
  ];

  assert.equal(
    remainingRainfallTotal(
      periods,
      "America/New_York",
      new Date("2026-07-26T14:30:00-04:00"),
    ),
    0.5,
  );
});

test("formats positive hourly amounts without adding zero amounts", () => {
  assert.equal(formatForecastRainfall(57, 0.34), "57% (0.34 in.)");
  assert.equal(formatForecastRainfall(57, 0), "57%");
  assert.equal(formatForecastRainfall(null, 0.006), "<0.01 in.");
  assert.equal(formatForecastRainfall(null, null), "—");
  assert.equal(formatObservedRainfall(0), "—");
  assert.equal(formatObservedRainfall(null), "—");
  assert.equal(formatObservedRainfall(0.006), "<0.01 in.");
  assert.equal(formatObservedRainfall(0.34), "0.34 in.");
  assert.equal(formatRainfallInches(0.004), "<0.01 in.");
});
