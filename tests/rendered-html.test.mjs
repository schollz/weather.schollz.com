import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

async function render() {
  const serverUrl = new URL("../dist/server/index.js", import.meta.url);
  serverUrl.searchParams.set("test", `${process.pid}-${Date.now()}`);
  const { default: handleRequest } = await import(serverUrl.href);

  return handleRequest(
    new Request("http://localhost/", {
      headers: { accept: "text/html" },
    }),
  );
}

test("server-renders the local NOAA weather app", async () => {
  const response = await render();
  assert.equal(response.status, 200);
  assert.match(response.headers.get("content-type") ?? "", /^text\/html\b/i);

  const html = await response.text();
  assert.match(
    html,
    /<title>weather\.schollz\.com — local NOAA weather<\/title>/i,
  );
  assert.match(
    html,
    /<link rel="canonical" href="https:\/\/weather\.schollz\.com\/?"/i,
  );
  assert.match(
    html,
    /<link href="favicon\.svg" rel="icon" type="image\/svg\+xml"/i,
  );
  assert.match(html, /property="og:title" content="Local NOAA weather"/i);
  assert.match(html, /name="twitter:card" content="summary"/i);
  assert.match(html, /type="application\/ld\+json"/i);
  assert.match(html, /"@type":"WebApplication"/i);
  assert.match(
    html,
    /<a[^>]*href="\/"[^>]*>weather\.schollz\.com<\/a>/i,
  );
  assert.match(html, /locating/);
  assert.match(html, /Finding your local weather/);
  assert.doesNotMatch(html, /codex-preview|react-loading-skeleton/i);
});

test("keeps NOAA requests and geolocation in the client app", async () => {
  const page = await readFile(
    new URL("../app/weather-client.tsx", import.meta.url),
    "utf8",
  );

  assert.match(page, /navigator\.geolocation\.getCurrentPosition/);
  assert.match(
    page,
    /maximumAge: requestKey === 0 \? 5 \* 60 \* 1000 : 0/,
  );
  assert.match(
    page,
    /new URLSearchParams\(window\.location\.search\)\.get\(/,
  );
  assert.match(page, /LOCATION_QUERY_KEY = "location"/);
  assert.match(page, /window\.history\.replaceState/);
  assert.match(page, /requestKey === 0 \? readSharedCoordinates\(\) : null/);
  assert.match(
    page,
    /https:\/\/geocoding-api\.open-meteo\.com\/v1\/search/,
  );
  assert.match(page, /url\.searchParams\.set\("countryCode", "US"\)/);
  assert.match(page, /Search U\.S\. places/);
  assert.match(page, /aria-label="Use current location"/);
  assert.match(page, /event\.preventDefault\(\);\s+retryLocation\(\);/);
  assert.match(page, /SEARCH_DEBOUNCE_MS = 350/);
  assert.match(page, /window\.setTimeout/);
  assert.match(page, /controller\.abort\(\)/);
  assert.match(page, /localStorage\.setItem\(THEME_STORAGE_KEY, theme\)/);
  assert.match(page, /useSyncExternalStore/);
  assert.match(page, /https:\/\/api\.weather\.gov\/points\//);
  assert.match(
    page,
    /fetchNoaaJson<ForecastResponse>\(point\.properties\.forecast\)/,
  );
  assert.match(page, /forecastHourly/);
  assert.match(page, /7-day forecast/);
  assert.match(page, /https:\/\/data\.rcc-acis\.org\/StnData/);
  assert.match(page, /groupby: "year"/);
  assert.match(page, /smry: \{ reduce: "max", add: "date" \}/);
  assert.match(page, /smry: \{ reduce: "min", add: "date" \}/);
  assert.match(page, /Record \$\{kind\}:/);
  assert.match(page, /rec hi/);
  assert.match(page, /rec lo/);
  assert.match(page, /isToday: dateKey === today/);
  assert.match(page, /className=\{forecast\.isToday \? "today" : undefined\}/);
  assert.match(page, /maximumPrecipitationChance/);
  assert.match(page, /\[\.\.\.periods\]\s*\.sort/);
  assert.match(
    page,
    /hourlyPeriods\.length \? hourlyPeriods : periods/,
  );
  assert.match(page, /cumulativeObservedRainfall/);
  assert.match(page, /todayRainfall !== null && todayRainfall > 0\.1/);
  assert.match(page, /rainfall: <strong>/);
  assert.match(page, /Highest rain chance:/);
  assert.match(
    page,
    /data-tooltip=\{formatRainTitle\(\s*forecast\.rain,\s*weather\.timeZone,/,
  );
  assert.doesNotMatch(page, /\btitle=/);
  assert.match(page, /period\.relativeHumidity\?\.value/);
  assert.match(page, /stationUrl}\/observations/);
  assert.match(page, /probabilityOfPrecipitation/);
  assert.match(page, /precipitationLastHour/);
  assert.match(page, /rawMessage\.match\(\/\\bP\(\\d\{4\}\)\\b\//);
  assert.match(page, /relativeHumidity/);
  assert.match(page, /DISPLAY_HOURS/);
});
