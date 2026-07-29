import assert from "node:assert/strict";
import test from "node:test";
import {
  fahrenheitToCelsius,
  formatTemperature,
  formatWind,
  usesMetricUnits,
} from "../app/units.mjs";

test("selects metric units outside U.S. locations", () => {
  assert.equal(usesMetricUnits("US", "noaa"), false);
  assert.equal(usesMetricUnits("us", "open-meteo"), false);
  assert.equal(usesMetricUnits("CA", "open-meteo"), true);
  assert.equal(usesMetricUnits("FR", "open-meteo"), true);
  assert.equal(usesMetricUnits(null, "noaa"), false);
  assert.equal(usesMetricUnits(null, "open-meteo"), true);
});

test("formats normalized weather values in the selected units", () => {
  assert.equal(fahrenheitToCelsius(68), 20);
  assert.equal(formatTemperature(68, false), "68°F");
  assert.equal(formatTemperature(68, true), "20°C");
  assert.equal(formatTemperature(null, true), "—");
  assert.equal(formatWind("5 to 10 mph", false), "5 to 10 mph");
  assert.equal(formatWind("5 to 10 mph", true), "8 to 16 km/h");
});
