import assert from "node:assert/strict";
import test from "node:test";
import vm from "node:vm";
import {
  WEATHER_CACHE_KEY,
  WEATHER_CACHE_TTL_MS,
  weatherCacheBootstrapScript,
} from "../app/weather-cache-bootstrap.mjs";

function runBootstrap({
  age = 1_000,
  pathname = "/portland/",
  search = "",
} = {}) {
  const dataset = {};
  const now = Date.now();
  const entries = [
    {
      key: "45.520,-122.674",
      paths: ["/portland/"],
      updatedAt: now - age,
      weather: {
        city: "Portland",
        current: {},
        daily: [],
        hourly: [],
      },
    },
  ];

  vm.runInNewContext(weatherCacheBootstrapScript(), {
    Date: { now: () => now },
    JSON,
    Number,
    URLSearchParams,
    document: { documentElement: { dataset } },
    localStorage: {
      getItem: (key) =>
        key === WEATHER_CACHE_KEY ? JSON.stringify(entries) : null,
    },
    location: { pathname, search },
  });

  return dataset;
}

test("marks a fresh cache hit before the weather UI renders", () => {
  assert.deepEqual(runBootstrap(), { weatherCache: "hit" });
});

test("does not hide loading for expired or unrelated cache entries", () => {
  assert.deepEqual(runBootstrap({ age: WEATHER_CACHE_TTL_MS }), {});
  assert.deepEqual(runBootstrap({ pathname: "/seattle/" }), {});
});

test("does not override legacy coordinate links", () => {
  assert.deepEqual(runBootstrap({ search: "?location=45.5,-122.6" }), {});
});
