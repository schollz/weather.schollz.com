"use client";

import {
  Cloud,
  CloudFog,
  CloudLightning,
  CloudRain,
  CloudSun,
  Droplets,
  LocateFixed,
  MapPin,
  Moon,
  RefreshCw,
  Search,
  Snowflake,
  Sun,
  Wind,
  X,
} from "lucide-react";
import Link from "next/link";
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
} from "react";

const DISPLAY_HOURS = Array.from({ length: 18 }, (_, index) => index + 5);
const SEATTLE = { latitude: 47.6062, longitude: -122.3321 };
const ACIS_STATION_DATA_URL = "https://data.rcc-acis.org/StnData";
const LOCATION_QUERY_KEY = "location";
const SEARCH_DEBOUNCE_MS = 350;
const THEME_STORAGE_KEY = "wx-theme";
const THEME_CHANGE_EVENT = "wx-theme-change";

type LoadPhase = "locating" | "loading" | "ready" | "error";
type Theme = "light" | "dark";

type TooltipState = {
  horizontalOffset: number;
  horizontalSide: "left" | "right";
  placement: "above" | "below";
  text: string;
  verticalOffset: number;
};

type PlaceResult = {
  admin1?: string;
  admin2?: string;
  country_code: string;
  id: number;
  latitude: number;
  longitude: number;
  name: string;
};

type GeocodingResponse = {
  results?: PlaceResult[];
};

function getStoredTheme(): Theme {
  if (typeof window === "undefined") return "dark";
  return window.localStorage.getItem(THEME_STORAGE_KEY) === "light"
    ? "light"
    : "dark";
}

function subscribeToTheme(onStoreChange: () => void) {
  const handleStorage = (event: StorageEvent) => {
    if (event.key === THEME_STORAGE_KEY) onStoreChange();
  };

  window.addEventListener("storage", handleStorage);
  window.addEventListener(THEME_CHANGE_EVENT, onStoreChange);

  return () => {
    window.removeEventListener("storage", handleStorage);
    window.removeEventListener(THEME_CHANGE_EVENT, onStoreChange);
  };
}

function useTooltip() {
  const [tooltip, setTooltip] = useState<TooltipState | null>(null);

  useEffect(() => {
    const tooltipTarget = (event: Event) =>
      event.target instanceof Element
        ? event.target.closest<HTMLElement>(".hover-tip[data-tooltip]")
        : null;

    const showTooltip = (target: HTMLElement) => {
      const text = target.dataset.tooltip;
      if (!text) return;

      const rect = target.getBoundingClientRect();
      const horizontalSide =
        rect.left + rect.width / 2 > window.innerWidth / 2 ? "right" : "left";
      const placement =
        target.classList.contains("header-tip") ||
        rect.top < window.innerHeight / 2
          ? "below"
          : "above";

      setTooltip({
        horizontalOffset:
          horizontalSide === "right"
            ? Math.max(16, window.innerWidth - rect.right)
            : Math.max(16, rect.left),
        horizontalSide,
        placement,
        text,
        verticalOffset:
          placement === "below"
            ? rect.bottom + 7
            : window.innerHeight - rect.top + 7,
      });
    };

    const handleMouseOver = (event: MouseEvent) => {
      const target = tooltipTarget(event);
      if (!target) return;

      if (
        event.relatedTarget instanceof Node &&
        target.contains(event.relatedTarget)
      ) {
        return;
      }

      showTooltip(target);
    };

    const handleMouseOut = (event: MouseEvent) => {
      const target = tooltipTarget(event);
      if (!target) return;

      if (
        event.relatedTarget instanceof Node &&
        target.contains(event.relatedTarget)
      ) {
        return;
      }

      setTooltip(null);
    };

    const handleFocusIn = (event: FocusEvent) => {
      const target = tooltipTarget(event);
      if (target) showTooltip(target);
    };

    const handleFocusOut = (event: FocusEvent) => {
      if (tooltipTarget(event)) setTooltip(null);
    };

    const hideTooltip = () => setTooltip(null);

    document.addEventListener("mouseover", handleMouseOver);
    document.addEventListener("mouseout", handleMouseOut);
    document.addEventListener("focusin", handleFocusIn);
    document.addEventListener("focusout", handleFocusOut);
    document.addEventListener("scroll", hideTooltip, true);
    window.addEventListener("resize", hideTooltip);

    return () => {
      document.removeEventListener("mouseover", handleMouseOver);
      document.removeEventListener("mouseout", handleMouseOut);
      document.removeEventListener("focusin", handleFocusIn);
      document.removeEventListener("focusout", handleFocusOut);
      document.removeEventListener("scroll", hideTooltip, true);
      window.removeEventListener("resize", hideTooltip);
    };
  }, []);

  return tooltip;
}

function saveTheme(theme: Theme) {
  window.localStorage.setItem(THEME_STORAGE_KEY, theme);
  document.documentElement.dataset.theme = theme;
  window.dispatchEvent(new Event(THEME_CHANGE_EVENT));
}

function readSharedCoordinates() {
  if (typeof window === "undefined") return null;

  const value = new URLSearchParams(window.location.search).get(
    LOCATION_QUERY_KEY,
  );

  if (!value) return null;

  const parts = value.split(",");
  if (parts.length !== 2) return null;

  const latitude = Number(parts[0]);
  const longitude = Number(parts[1]);
  const isValid =
    Number.isFinite(latitude) &&
    Number.isFinite(longitude) &&
    latitude >= -90 &&
    latitude <= 90 &&
    longitude >= -180 &&
    longitude <= 180;

  return isValid ? { latitude, longitude } : null;
}

function updateSharedCoordinates(latitude: number, longitude: number) {
  const url = new URL(window.location.href);
  url.searchParams.set(
    LOCATION_QUERY_KEY,
    `${latitude.toFixed(4)},${longitude.toFixed(4)}`,
  );
  window.history.replaceState(window.history.state, "", url);
}

type NoaaValue = {
  unitCode: string;
  value: number | null;
};

type HourlyPeriod = {
  endTime: string;
  icon: string;
  isDaytime: boolean;
  name: string;
  number: number;
  probabilityOfPrecipitation?: NoaaValue;
  relativeHumidity?: NoaaValue;
  shortForecast: string;
  startTime: string;
  temperature: number;
  temperatureUnit: string;
  windDirection: string;
  windSpeed: string;
};

type CurrentReading = {
  description: string;
  humidity: number | null;
  temperatureF: number | null;
  timestamp: string | null;
};

type WeatherData = {
  city: string;
  coordinates: { latitude: number; longitude: number };
  current: CurrentReading;
  daily: HourlyPeriod[];
  hourly: HourlyPeriod[];
  observations: StationObservation[];
  state: string;
  stationId: string | null;
  timeZone: string;
};

type ClimateRecordValue = {
  date: string;
  temperatureF: number;
};

type ClimateRecord = {
  high: ClimateRecordValue | null;
  low: ClimateRecordValue | null;
};

type ClimateRecords = Record<string, ClimateRecord>;

type AcisResponse = {
  error?: string;
  meta?: {
    name?: string;
  };
  smry?: unknown[];
};

type PointResponse = {
  properties: {
    forecast: string;
    forecastHourly: string;
    observationStations: string;
    relativeLocation?: {
      properties?: {
        city?: string;
        state?: string;
      };
    };
    timeZone: string;
  };
};

type ObservationStationsResponse = {
  features: Array<{ id: string }>;
};

type StationObservation = {
  icon: string | null;
  precipitationLastHour: NoaaValue | null;
  rawMessage: string | null;
  relativeHumidity: NoaaValue;
  temperature: NoaaValue;
  textDescription: string;
  timestamp: string;
};

type StationObservationsResponse = {
  features: Array<{
    properties: StationObservation;
  }>;
};

type ForecastResponse = {
  properties: {
    periods: HourlyPeriod[];
  };
};

const noaaHeaders = {
  Accept: "application/geo+json",
};

async function fetchNoaaJson<T>(url: string): Promise<T> {
  const response = await fetch(url, { headers: noaaHeaders });

  if (!response.ok) {
    throw new Error(`NOAA request failed (${response.status})`);
  }

  return response.json() as Promise<T>;
}

function celsiusToFahrenheit(value: number | null) {
  return value === null ? null : (value * 9) / 5 + 32;
}

function metersToInches(value: number | null) {
  return value === null ? null : value * 39.3701;
}

function observationPrecipitationInches(observation: StationObservation) {
  const structuredValue = observation.precipitationLastHour?.value ?? null;
  const rawMessage = observation.rawMessage ?? "";

  if (structuredValue !== null) {
    return metersToInches(structuredValue);
  }

  const metarAmount = rawMessage.match(/\bP(\d{4})\b/);

  if (metarAmount) {
    return Number(metarAmount[1]) / 100;
  }

  // Routine hourly METARs omit the P-group when no measurable rain fell.
  return rawMessage ? 0 : null;
}

function observationPriority(observation: StationObservation) {
  const minute = new Date(observation.timestamp).getUTCMinutes();
  const isRoutineHourlyReport = minute >= 51 && minute <= 59;
  const rawMessage = observation.rawMessage ?? "";
  const hasMetar = Boolean(rawMessage);
  const hasRainValue =
    observation.precipitationLastHour?.value != null ||
    /\bP\d{4}\b/.test(rawMessage);

  return (
    (isRoutineHourlyReport ? 4 : 0) +
    (hasMetar ? 2 : 0) +
    (hasRainValue ? 1 : 0)
  );
}

function isPreferredObservation(
  candidate: StationObservation,
  existing: StationObservation | undefined,
) {
  if (!existing) return true;

  const candidatePriority = observationPriority(candidate);
  const existingPriority = observationPriority(existing);

  return (
    candidatePriority > existingPriority ||
    (candidatePriority === existingPriority &&
      new Date(candidate.timestamp).getTime() >
        new Date(existing.timestamp).getTime())
  );
}

function cumulativeObservedRainfall(
  observations: StationObservation[],
  timeZone: string,
  now = new Date(),
) {
  const today = localDateKey(now, timeZone);
  const observationsByHour = new Map<string, StationObservation>();

  observations.forEach((observation) => {
    const timestamp = new Date(observation.timestamp);
    if (localDateKey(timestamp, timeZone) !== today) return;

    // UTC hour keys preserve both occurrences of a repeated DST hour while
    // still collapsing multiple station reports within the same hour.
    const hourKey = timestamp.toISOString().slice(0, 13);
    const existing = observationsByHour.get(hourKey);

    if (isPreferredObservation(observation, existing)) {
      observationsByHour.set(hourKey, observation);
    }
  });

  let foundRainfallValue = false;
  let total = 0;

  observationsByHour.forEach((observation) => {
    const rainfall = observationPrecipitationInches(observation);

    if (rainfall !== null) {
      foundRainfallValue = true;
      total += rainfall;
    }
  });

  return foundRainfallValue ? total : null;
}

async function getStationObservations(
  stationsUrl: string,
): Promise<{
  observations: StationObservation[];
  stationId: string | null;
}> {
  try {
    const stations =
      await fetchNoaaJson<ObservationStationsResponse>(stationsUrl);
    const stationUrl = stations.features[0]?.id;

    if (!stationUrl) return { observations: [], stationId: null };

    const end = new Date();
    const start = new Date(end.getTime() - 30 * 60 * 60 * 1000);
    const url = new URL(`${stationUrl}/observations`);
    url.searchParams.set("start", start.toISOString());
    url.searchParams.set("end", end.toISOString());
    url.searchParams.set("limit", "500");

    const response = await fetchNoaaJson<StationObservationsResponse>(
      url.toString(),
    );
    return {
      observations: response.features.map(({ properties }) => properties),
      stationId: stationUrl.split("/").pop() ?? null,
    };
  } catch {
    return { observations: [], stationId: null };
  }
}

async function getNoaaWeather(
  latitude: number,
  longitude: number,
): Promise<WeatherData> {
  const point = await fetchNoaaJson<PointResponse>(
    `https://api.weather.gov/points/${latitude.toFixed(4)},${longitude.toFixed(4)}`,
  );

  const [hourlyForecast, dailyForecast, stationData] = await Promise.all([
    fetchNoaaJson<ForecastResponse>(point.properties.forecastHourly),
    fetchNoaaJson<ForecastResponse>(point.properties.forecast),
    getStationObservations(point.properties.observationStations),
  ]);

  const hourly = hourlyForecast.properties.periods;
  const observation = stationData.observations[0];
  const fallback = hourly[0];
  const temperatureF = observation?.temperature
    ? celsiusToFahrenheit(observation.temperature.value)
    : (fallback?.temperature ?? null);
  const humidity =
    observation?.relativeHumidity.value ??
    fallback?.relativeHumidity?.value ??
    null;

  return {
    city: point.properties.relativeLocation?.properties?.city ?? "Your location",
    coordinates: { latitude, longitude },
    current: {
      description:
        observation?.textDescription ||
        fallback?.shortForecast ||
        "Conditions unavailable",
      humidity,
      temperatureF,
      timestamp: observation?.timestamp ?? fallback?.startTime ?? null,
    },
    daily: dailyForecast.properties.periods,
    hourly,
    observations: stationData.observations,
    state: point.properties.relativeLocation?.properties?.state ?? "",
    stationId: stationData.stationId,
    timeZone: point.properties.timeZone,
  };
}

function addAcisSummaries(
  records: ClimateRecords,
  summaries: unknown,
  kind: keyof ClimateRecord,
) {
  if (!Array.isArray(summaries)) return;

  summaries.forEach((summary) => {
    if (!Array.isArray(summary) || summary.length < 2) return;

    const temperatureF = Number(summary[0]);
    const date = summary[1];
    const dateMatch =
      typeof date === "string"
        ? /^\d{4}-(\d{2})-(\d{2})$/.exec(date)
        : null;

    if (!Number.isFinite(temperatureF) || !dateMatch) return;

    const monthDay = `${dateMatch[1]}-${dateMatch[2]}`;
    const record = records[monthDay] ?? { high: null, low: null };
    record[kind] = { date, temperatureF };
    records[monthDay] = record;
  });
}

async function getAcisDailyRecords(
  stationId: string,
  signal: AbortSignal,
): Promise<{ records: ClimateRecords; stationName: string | null }> {
  const params = {
    sid: stationId,
    sdate: "por",
    edate: "por",
    meta: ["name", "state"],
    elems: [
      {
        name: "maxt",
        interval: "dly",
        duration: "dly",
        smry: { reduce: "max", add: "date" },
        smry_only: 1,
        groupby: "year",
      },
      {
        name: "mint",
        interval: "dly",
        duration: "dly",
        smry: { reduce: "min", add: "date" },
        smry_only: 1,
        groupby: "year",
      },
    ],
  };
  const response = await fetch(ACIS_STATION_DATA_URL, {
    body: new URLSearchParams({ params: JSON.stringify(params) }),
    method: "POST",
    signal,
  });

  if (!response.ok) {
    throw new Error(`ACIS request failed (${response.status})`);
  }

  const data = (await response.json()) as AcisResponse;

  if (data.error) throw new Error(data.error);

  const records: ClimateRecords = {};
  addAcisSummaries(records, data.smry?.[0], "high");
  addAcisSummaries(records, data.smry?.[1], "low");

  return {
    records,
    stationName: data.meta?.name ?? null,
  };
}

function zonedParts(date: Date, timeZone: string) {
  const parts = new Intl.DateTimeFormat("en-CA", {
    day: "2-digit",
    hour: "2-digit",
    hour12: false,
    hourCycle: "h23",
    minute: "2-digit",
    month: "2-digit",
    timeZone,
    year: "numeric",
  }).formatToParts(date);

  return Object.fromEntries(parts.map(({ type, value }) => [type, value]));
}

function localDateKey(date: Date, timeZone: string) {
  const parts = zonedParts(date, timeZone);
  return `${parts.year}-${parts.month}-${parts.day}`;
}

function closestDisplayHour(now: Date, timeZone: string) {
  const parts = zonedParts(now, timeZone);
  const fractionalHour = Number(parts.hour) + Number(parts.minute) / 60;
  return Math.min(22, Math.max(5, Math.round(fractionalHour)));
}

function formatHour(hour: number) {
  const period = hour >= 12 ? "PM" : "AM";
  const displayHour = hour % 12 || 12;
  return `${displayHour} ${period}`;
}

function formatDate(timeZone: string) {
  return new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "short",
    timeZone,
    weekday: "long",
  }).format(new Date());
}

function formatForecastDay(date: Date, timeZone: string) {
  const weekday = new Intl.DateTimeFormat("en-US", {
    timeZone,
    weekday: "short",
  }).format(date);
  const monthDay = new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "numeric",
    timeZone,
  }).format(date);

  return `${weekday} ${monthDay}`;
}

function formatRecordTitle(
  kind: "high" | "low",
  record: ClimateRecordValue | null,
) {
  if (!record) return undefined;

  const date = new Intl.DateTimeFormat("en-US", {
    day: "numeric",
    month: "long",
    timeZone: "UTC",
    year: "numeric",
  }).format(new Date(`${record.date}T00:00:00Z`));

  return `Record ${kind}: ${Math.round(record.temperatureF)}°F on ${date}`;
}

type PrecipitationPeak = {
  startTime: string;
  value: number;
};

function maximumPrecipitationChance(
  periods: HourlyPeriod[],
): PrecipitationPeak | null {
  return [...periods]
    .sort(
      (left, right) =>
        new Date(left.startTime).getTime() -
        new Date(right.startTime).getTime(),
    )
    .reduce<PrecipitationPeak | null>((highest, period) => {
      const value = period.probabilityOfPrecipitation?.value ?? null;

      // Keep the existing period on ties so the first chronological hour wins.
      if (value === null || (highest && value <= highest.value)) return highest;

      return { startTime: period.startTime, value };
    }, null);
}

function formatRainTitle(
  rain: PrecipitationPeak | null,
  timeZone: string,
) {
  if (!rain) return undefined;

  const time = new Intl.DateTimeFormat("en-US", {
    hour: "numeric",
    timeZone,
    timeZoneName: "short",
  }).format(new Date(rain.startTime));

  return `Highest rain chance: ${Math.round(rain.value)}% at ${time}`;
}

function formatUpdated(timestamp: string | null, timeZone: string) {
  if (!timestamp) return "update time unavailable";
  return `observed ${new Intl.DateTimeFormat("en-US", {
    hour: "numeric",
    minute: "2-digit",
    timeZone,
  }).format(new Date(timestamp))}`;
}

function WeatherIcon({
  description,
  isDaytime = true,
  size,
  strokeWidth,
}: {
  description: string;
  isDaytime?: boolean;
  size: number;
  strokeWidth?: number;
}) {
  const value = description.toLowerCase();
  const props = { "aria-hidden": true, size, strokeWidth };

  if (value.includes("thunder")) return <CloudLightning {...props} />;
  if (value.includes("snow") || value.includes("sleet")) {
    return <Snowflake {...props} />;
  }
  if (
    value.includes("rain") ||
    value.includes("shower") ||
    value.includes("drizzle")
  ) {
    return <CloudRain {...props} />;
  }
  if (value.includes("fog") || value.includes("haze") || value.includes("smoke")) {
    return <CloudFog {...props} />;
  }
  if (
    value.includes("partly") ||
    value.includes("mostly sunny") ||
    value.includes("mostly clear")
  ) {
    return <CloudSun {...props} />;
  }
  if (value.includes("cloud") || value.includes("overcast")) {
    return <Cloud {...props} />;
  }
  return isDaytime ? <Sun {...props} /> : <Moon {...props} />;
}

function readableError(error: GeolocationPositionError | Error) {
  if ("code" in error) {
    if (error.code === error.PERMISSION_DENIED) {
      return "Location access was declined. Allow location access and try again.";
    }
    if (error.code === error.POSITION_UNAVAILABLE) {
      return "Your location is currently unavailable.";
    }
    return "Finding your location took too long.";
  }

  return error.message.includes("404")
    ? "NOAA forecasts are available for U.S. locations only."
    : "NOAA weather data could not be reached. Please try again.";
}

export default function WeatherClient() {
  const [phase, setPhase] = useState<LoadPhase>("locating");
  const [weather, setWeather] = useState<WeatherData | null>(null);
  const [error, setError] = useState("");
  const [requestKey, setRequestKey] = useState(0);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const [placeResults, setPlaceResults] = useState<PlaceResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState("");
  const [climateRecords, setClimateRecords] = useState<ClimateRecords>({});
  const [climateRecordStationId, setClimateRecordStationId] = useState<
    string | null
  >(null);
  const [climateStationName, setClimateStationName] = useState<string | null>(
    null,
  );
  const theme = useSyncExternalStore(
    subscribeToTheme,
    getStoredTheme,
    () => "dark",
  );
  const tooltip = useTooltip();

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  const searchPlaces = useCallback(
    async (query: string, signal: AbortSignal) => {
      setSearching(true);
      setSearchError("");

      try {
        const url = new URL(
          "https://geocoding-api.open-meteo.com/v1/search",
        );
        url.searchParams.set("name", query);
        url.searchParams.set("count", "6");
        url.searchParams.set("language", "en");
        url.searchParams.set("format", "json");
        url.searchParams.set("countryCode", "US");

        const response = await fetch(url, { signal });

        if (!response.ok) {
          throw new Error(`Place search failed (${response.status})`);
        }

        const data = (await response.json()) as GeocodingResponse;
        const results = (data.results ?? []).filter(
          (place) => place.country_code === "US",
        );

        setPlaceResults(results);
        setSearchError(results.length ? "" : "No U.S. places found.");
      } catch {
        if (signal.aborted) return;
        setPlaceResults([]);
        setSearchError("Place search is unavailable. Try again.");
      } finally {
        if (!signal.aborted) setSearching(false);
      }
    },
    [],
  );

  useEffect(() => {
    const query = searchQuery.trim();
    if (!searchOpen || query.length < 2) return;

    const controller = new AbortController();
    const timeout = window.setTimeout(() => {
      void searchPlaces(query, controller.signal);
    }, SEARCH_DEBOUNCE_MS);

    return () => {
      window.clearTimeout(timeout);
      controller.abort();
    };
  }, [searchOpen, searchPlaces, searchQuery]);

  const loadCoordinates = useCallback(
    async (latitude: number, longitude: number) => {
      setPhase("loading");
      setError("");

      try {
        const result = await getNoaaWeather(latitude, longitude);
        updateSharedCoordinates(latitude, longitude);
        setWeather(result);
        setPhase("ready");
      } catch (weatherError) {
        setError(readableError(weatherError as Error));
        setPhase("error");
      }
    },
    [],
  );

  const stationId = weather?.stationId ?? null;

  useEffect(() => {
    if (!stationId) return;

    const controller = new AbortController();

    void getAcisDailyRecords(stationId, controller.signal)
      .then(({ records, stationName }) => {
        if (controller.signal.aborted) return;
        setClimateRecords(records);
        setClimateRecordStationId(stationId);
        setClimateStationName(stationName);
      })
      .catch(() => {
        if (controller.signal.aborted) return;
        setClimateRecords({});
        setClimateRecordStationId(null);
        setClimateStationName(null);
      });

    return () => controller.abort();
  }, [stationId, weather]);

  useEffect(() => {
    const sharedCoordinates =
      requestKey === 0 ? readSharedCoordinates() : null;

    if (sharedCoordinates) {
      queueMicrotask(() => {
        void loadCoordinates(
          sharedCoordinates.latitude,
          sharedCoordinates.longitude,
        );
      });
      return;
    }

    if (!navigator.geolocation) {
      queueMicrotask(() => {
        setError("This browser does not support location access.");
        setPhase("error");
      });
      return;
    }

    navigator.geolocation.getCurrentPosition(
      ({ coords }) => loadCoordinates(coords.latitude, coords.longitude),
      (locationError) => {
        setError(readableError(locationError));
        setPhase("error");
      },
      {
        enableHighAccuracy: false,
        maximumAge: requestKey === 0 ? 5 * 60 * 1000 : 0,
        timeout: 15_000,
      },
    );
  }, [loadCoordinates, requestKey]);

  const day = useMemo(() => {
    if (!weather) return [];

    const now = new Date();
    const today = localDateKey(now, weather.timeZone);
    const closestHour = closestDisplayHour(now, weather.timeZone);
    const currentHour = Number(zonedParts(now, weather.timeZone).hour);
    const hourlyByHour = new Map<number, HourlyPeriod>();
    const observationsByHour = new Map<number, StationObservation>();

    weather.hourly.forEach((period) => {
      const start = new Date(period.startTime);
      const parts = zonedParts(start, weather.timeZone);
      const hour = Number(parts.hour);

      if (
        localDateKey(start, weather.timeZone) === today &&
        hour >= 5 &&
        hour <= 22
      ) {
        hourlyByHour.set(hour, period);
      }
    });

    weather.observations.forEach((observation) => {
      const timestamp = new Date(observation.timestamp);
      const parts = zonedParts(timestamp, weather.timeZone);
      const hour = Number(parts.hour);
      const existing = observationsByHour.get(hour);

      if (
        localDateKey(timestamp, weather.timeZone) === today &&
        hour >= 5 &&
        hour <= 22 &&
        isPreferredObservation(observation, existing)
      ) {
        observationsByHour.set(hour, observation);
      }
    });

    return DISPLAY_HOURS.map((hour) => ({
      hour,
      isClosest: hour === closestHour,
      reading:
        hour <= currentHour && observationsByHour.has(hour)
          ? (() => {
              const observation = observationsByHour.get(hour)!;
              const precipitation =
                observationPrecipitationInches(observation);
              return {
                description:
                  observation.textDescription || "Observed conditions",
                humidity: observation.relativeHumidity.value,
                isDaytime: hour >= 6 && hour < 20,
                rain:
                  precipitation === null
                    ? "—"
                    : `${precipitation.toFixed(2)} in.`,
                source: "observed" as const,
                temperatureF: celsiusToFahrenheit(
                  observation.temperature.value,
                ),
              };
            })()
          : hourlyByHour.has(hour)
            ? (() => {
                const period = hourlyByHour.get(hour)!;
                const precipitation =
                  period.probabilityOfPrecipitation?.value ?? null;
                return {
                  description: period.shortForecast,
                  humidity: period.relativeHumidity?.value ?? null,
                  isDaytime: period.isDaytime,
                  rain:
                    precipitation === null
                      ? "—"
                      : `${Math.round(precipitation)}%`,
                  source: "forecast" as const,
                  temperatureF: period.temperature,
                };
              })()
            : null,
    }));
  }, [weather]);

  const todayRainfall = useMemo(
    () =>
      weather
        ? cumulativeObservedRainfall(
            weather.observations,
            weather.timeZone,
          )
        : null,
    [weather],
  );

  const sevenDay = useMemo(() => {
    if (!weather) return [];

    const today = localDateKey(new Date(), weather.timeZone);
    const periodsByDate = new Map<string, HourlyPeriod[]>();
    const hourlyByDate = new Map<string, HourlyPeriod[]>();

    weather.daily.forEach((period) => {
      const start = new Date(period.startTime);
      const dateKey = localDateKey(start, weather.timeZone);
      const periods = periodsByDate.get(dateKey) ?? [];
      periods.push(period);
      periodsByDate.set(dateKey, periods);
    });

    weather.hourly.forEach((period) => {
      const dateKey = localDateKey(
        new Date(period.startTime),
        weather.timeZone,
      );
      const periods = hourlyByDate.get(dateKey) ?? [];
      periods.push(period);
      hourlyByDate.set(dateKey, periods);
    });

    return Array.from(periodsByDate.entries())
      .slice(0, 7)
      .map(([dateKey, periods]) => {
        const daytime = periods.find((period) => period.isDaytime);
        const nighttime = periods.find((period) => !period.isDaytime);
        const representative = daytime ?? nighttime ?? periods[0];
        const hourlyPeriods = hourlyByDate.get(dateKey) ?? [];
        const record =
          climateRecordStationId === stationId
            ? climateRecords[dateKey.slice(5)]
            : undefined;

        return {
          date: formatForecastDay(
            new Date(representative.startTime),
            weather.timeZone,
          ),
          description: representative.shortForecast,
          high: daytime?.temperature ?? null,
          isDaytime: representative.isDaytime,
          isToday: dateKey === today,
          low: nighttime?.temperature ?? null,
          rain: maximumPrecipitationChance(
            hourlyPeriods.length ? hourlyPeriods : periods,
          ),
          recordHigh: record?.high ?? null,
          recordLow: record?.low ?? null,
        };
      });
  }, [climateRecordStationId, climateRecords, stationId, weather]);

  const retryLocation = () => {
    setSearchOpen(false);
    setSearching(false);
    setPhase("locating");
    setError("");
    setRequestKey((value) => value + 1);
  };

  const closePlaceSearch = () => {
    setSearchOpen(false);
    setSearching(false);
  };

  const selectPlace = (place: PlaceResult) => {
    closePlaceSearch();
    setSearchQuery("");
    setPlaceResults([]);
    setSearchError("");
    void loadCoordinates(place.latitude, place.longitude);
  };

  return (
    <main className="weather-shell">
      <header className="site-header">
        <div className="header-line">
          <Link
            className="site-title"
            href="/"
            onClick={(event) => {
              event.preventDefault();
              retryLocation();
            }}
          >
            weather.schollz.com
          </Link>
          <div className="header-actions">
            <button
              aria-controls="place-search"
              aria-expanded={searchOpen}
              aria-label={
                searchOpen ? "Close place search" : "Search U.S. places"
              }
              className="icon-button hover-tip header-tip"
              data-tooltip={
                searchOpen ? "Close place search" : "Search U.S. places"
              }
              onClick={() => {
                if (searchOpen) closePlaceSearch();
                else setSearchOpen(true);
              }}
              type="button"
            >
              <Search aria-hidden="true" size={14} />
            </button>
            <button
              aria-label="Use current location"
              className="icon-button hover-tip header-tip"
              data-tooltip="Use current location"
              onClick={retryLocation}
              type="button"
            >
              <LocateFixed aria-hidden="true" size={14} />
            </button>
            <button
              className="icon-button hover-tip header-tip"
              type="button"
              aria-label={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
              data-tooltip={`Switch to ${theme === "dark" ? "light" : "dark"} mode`}
              onClick={() => saveTheme(theme === "dark" ? "light" : "dark")}
            >
              {theme === "dark" ? (
                <Sun aria-hidden="true" size={14} />
              ) : (
                <Moon aria-hidden="true" size={14} />
              )}
            </button>
          </div>
        </div>

        {searchOpen ? (
          <div
            aria-label="Search U.S. places"
            className="place-search"
            id="place-search"
            onKeyDown={(event) => {
              if (event.key === "Escape") closePlaceSearch();
            }}
            role="dialog"
          >
            <div className="search-heading">
              <span>search U.S. weather</span>
              <button
                aria-label="Close place search"
                className="search-close"
                onClick={closePlaceSearch}
                type="button"
              >
                <X aria-hidden="true" size={14} />
              </button>
            </div>

            <form onSubmit={(event) => event.preventDefault()}>
              <label className="sr-only" htmlFor="place-query">
                City or ZIP code
              </label>
              <div className="search-input-row">
                <Search aria-hidden="true" size={14} />
                <input
                  autoComplete="off"
                  autoFocus
                  id="place-query"
                  onChange={(event) => {
                    setSearchQuery(event.target.value);
                    setPlaceResults([]);
                    setSearchError("");
                    setSearching(false);
                  }}
                  placeholder="City or ZIP code"
                  type="search"
                  value={searchQuery}
                />
              </div>
            </form>

            <p aria-live="polite" className="search-hint">
              {searchError ||
                (searching
                  ? "Searching..."
                  : searchQuery.trim().length === 1
                    ? "Enter at least two characters."
                    : "U.S. locations only")}
            </p>

            {placeResults.length ? (
              <ul className="search-results">
                {placeResults.map((place) => (
                  <li key={place.id}>
                    <button
                      className="search-result"
                      onClick={() => selectPlace(place)}
                      type="button"
                    >
                      <strong>{place.name}</strong>
                      <span>
                        {[place.admin2, place.admin1]
                          .filter(Boolean)
                          .join(", ")}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
        ) : null}
      </header>

      {phase === "ready" && weather ? (
        <>
          <section className="current-panel" aria-labelledby="current-heading">
            <p className="section-label">Weather for</p>
            <h1 id="current-heading">
              <MapPin aria-hidden="true" size={15} />
              {weather.city}
              {weather.state ? `, ${weather.state}` : ""}
            </h1>
            <p className="meta-line">
              {formatDate(weather.timeZone)} /{" "}
              {formatUpdated(weather.current.timestamp, weather.timeZone)}
            </p>

            <div className="text-rule" aria-hidden="true" />

            <p className="section-label">Current conditions</p>
            <div className="current-reading">
              <WeatherIcon
                description={weather.current.description}
                size={27}
                strokeWidth={1.5}
              />
              <div>
                <strong className="current-temperature">
                  {weather.current.temperatureF === null
                    ? "—"
                    : `${Math.round(weather.current.temperatureF)}°F`}
                </strong>
                <span className="condition-text">
                  {weather.current.description}
                </span>
              </div>
            </div>

            <div className="current-facts">
              <div className="current-fact-stack">
                <span className="current-fact">
                  <Droplets aria-hidden="true" size={14} />
                  humidity:{" "}
                  <strong>
                    {weather.current.humidity === null
                      ? "—"
                      : `${Math.round(weather.current.humidity)}%`}
                  </strong>
                </span>
                {todayRainfall !== null && todayRainfall > 0.1 ? (
                  <span className="current-fact">
                    <CloudRain aria-hidden="true" size={14} />
                    rainfall: <strong>{todayRainfall.toFixed(2)} in.</strong>
                  </span>
                ) : null}
              </div>
              <span className="current-fact">
                <Wind aria-hidden="true" size={14} />
                wind:{" "}
                <strong>{weather.hourly[0]?.windSpeed ?? "—"}</strong>
              </span>
            </div>
          </section>

          <div className="ascii-rule" aria-hidden="true">
            ============================================================
          </div>

          <section className="hourly-panel" aria-label="Hourly weather">
            <div className="hourly-heading">
              <div>
                <p>{formatDate(weather.timeZone)}</p>
              </div>
            </div>

            <div className="forecast-labels" aria-hidden="true">
              <span />
              <span>time</span>
              <span>sky</span>
              <span>temp</span>
              <span>humid</span>
              <span>rain*</span>
            </div>

            <ol className="hourly-list">
              {day.map(({ hour, isClosest, reading }) => (
                <li className={isClosest ? "closest" : ""} key={hour}>
                  <span className="now-marker" aria-hidden="true">
                    {isClosest ? ">" : ""}
                  </span>
                  <time>{formatHour(hour)}</time>
                  <span
                    className="weather-cell-icon hover-tip"
                    aria-label={reading?.description ?? "Unavailable"}
                    data-tooltip={
                      reading
                        ? `${reading.description} — ${reading.source}`
                        : "Unavailable"
                    }
                    tabIndex={0}
                  >
                    {reading ? (
                      <WeatherIcon
                        description={reading.description}
                        isDaytime={reading.isDaytime}
                        size={15}
                      />
                    ) : (
                      "—"
                    )}
                  </span>
                  <span>
                    {reading?.temperatureF === null ||
                    reading?.temperatureF === undefined
                      ? "—"
                      : `${Math.round(reading.temperatureF)}°F`}
                  </span>
                  <span>
                    {reading?.humidity === null ||
                    reading?.humidity === undefined
                      ? "—"
                      : `${Math.round(reading.humidity)}%`}
                  </span>
                  <span>{reading?.rain ?? "—"}</span>
                </li>
              ))}
            </ol>

            <p className="table-note">
              * past = observed 1h rainfall (in.) / future = precipitation
              chance. Earlier hours use station {weather.stationId ?? "data"}.
            </p>
          </section>

          <div className="ascii-rule" aria-hidden="true">
            ============================================================
          </div>

          <section
            aria-labelledby="daily-heading"
            className="daily-panel"
          >
            <h2 id="daily-heading">7-day forecast</h2>

            <div className="daily-labels" aria-hidden="true">
              <span>day</span>
              <span>sky</span>
              <span>high</span>
              <span>low</span>
              <span>rec hi</span>
              <span>rec lo</span>
              <span>rain</span>
            </div>

            <ol className="daily-list">
              {sevenDay.map((forecast) => (
                <li
                  className={forecast.isToday ? "today" : undefined}
                  key={forecast.date}
                >
                  <time>{forecast.date}</time>
                  <span
                    aria-label={forecast.description}
                    className="weather-cell-icon hover-tip"
                    data-tooltip={forecast.description}
                    tabIndex={0}
                  >
                    <WeatherIcon
                      description={forecast.description}
                      isDaytime={forecast.isDaytime}
                      size={15}
                    />
                  </span>
                  <span>
                    {forecast.high === null
                      ? "—"
                      : `${Math.round(forecast.high)}°F`}
                  </span>
                  <span>
                    {forecast.low === null
                      ? "—"
                      : `${Math.round(forecast.low)}°F`}
                  </span>
                  <span
                    aria-label={formatRecordTitle("high", forecast.recordHigh)}
                    className={forecast.recordHigh ? "hover-tip" : undefined}
                    data-tooltip={formatRecordTitle(
                      "high",
                      forecast.recordHigh,
                    )}
                    tabIndex={forecast.recordHigh ? 0 : undefined}
                  >
                    {forecast.recordHigh === null
                      ? "—"
                      : `${Math.round(forecast.recordHigh.temperatureF)}°F`}
                  </span>
                  <span
                    aria-label={formatRecordTitle("low", forecast.recordLow)}
                    className={forecast.recordLow ? "hover-tip" : undefined}
                    data-tooltip={formatRecordTitle("low", forecast.recordLow)}
                    tabIndex={forecast.recordLow ? 0 : undefined}
                  >
                    {forecast.recordLow === null
                      ? "—"
                      : `${Math.round(forecast.recordLow.temperatureF)}°F`}
                  </span>
                  <span
                    aria-label={formatRainTitle(
                      forecast.rain,
                      weather.timeZone,
                    )}
                    className={forecast.rain ? "hover-tip" : undefined}
                    data-tooltip={formatRainTitle(
                      forecast.rain,
                      weather.timeZone,
                    )}
                    tabIndex={forecast.rain ? 0 : undefined}
                  >
                    {forecast.rain === null
                      ? "—"
                      : `${Math.round(forecast.rain.value)}%`}
                  </span>
                </li>
              ))}
            </ol>
          </section>

          <div className="ascii-rule" aria-hidden="true">
            ============================================================
          </div>

          <footer className="site-footer">
            <p>
              location: {weather.coordinates.latitude.toFixed(3)},{" "}
              {weather.coordinates.longitude.toFixed(3)}
            </p>
            <p>
              data:{" "}
              <a
                href="https://www.weather.gov/documentation/services-web-api"
                target="_blank"
                rel="noreferrer"
              >
                weather.gov
              </a>
            </p>
            {weather.stationId ? (
              <p>
                history:{" "}
                <a
                  href={`https://www.weather.gov/wrh/timeseries?site=${weather.stationId}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  station {weather.stationId}
                </a>
              </p>
            ) : null}
            {climateRecordStationId === stationId &&
            Object.keys(climateRecords).length ? (
              <p>
                records:{" "}
                <a
                  href="https://builder.rcc-acis.org/"
                  target="_blank"
                  rel="noreferrer"
                >
                  ACIS / {climateStationName ?? stationId}
                </a>
              </p>
            ) : null}
            <button
              className="text-button"
              onClick={() =>
                loadCoordinates(
                  weather.coordinates.latitude,
                  weather.coordinates.longitude,
                )
              }
              type="button"
            >
              <RefreshCw aria-hidden="true" size={13} />
              refresh forecast
            </button>
          </footer>
        </>
      ) : (
        <section className="status-panel" aria-live="polite">
          <LocateFixed
            className="status-glyph"
            aria-hidden="true"
            size={24}
            strokeWidth={1.5}
          />
          {phase === "error" ? (
            <>
              <p className="status-code">&gt; location / data error</p>
              <h1>{error}</h1>
              <p>
                Your coordinates go directly from this browser to NOAA and are
                not stored by this site.
              </p>
              <div className="status-actions">
                <button
                  className="primary-button"
                  onClick={retryLocation}
                  type="button"
                >
                  &gt; try location again
                </button>
                <button
                  className="secondary-button"
                  onClick={() =>
                    loadCoordinates(SEATTLE.latitude, SEATTLE.longitude)
                  }
                  type="button"
                >
                  &gt; preview seattle
                </button>
              </div>
            </>
          ) : (
            <>
              <p className="status-code">
                &gt; {phase === "locating" ? "locating" : "contacting noaa"}
                <span className="loading-dots" aria-hidden="true">
                  ...
                </span>
              </p>
              <h1>
                {phase === "locating"
                  ? "Finding your local weather."
                  : "Reading the latest forecast."}
              </h1>
              <p>
                Allow location access when prompted. NOAA covers U.S. locations.
              </p>
            </>
          )}
        </section>
      )}
      {tooltip ? (
        <div
          className="tooltip-layer"
          role="tooltip"
          style={{
            ...(tooltip.horizontalSide === "right"
              ? { right: tooltip.horizontalOffset }
              : { left: tooltip.horizontalOffset }),
            ...(tooltip.placement === "below"
              ? { top: tooltip.verticalOffset }
              : { bottom: tooltip.verticalOffset }),
          }}
        >
          {tooltip.text}
        </div>
      ) : null}
    </main>
  );
}
