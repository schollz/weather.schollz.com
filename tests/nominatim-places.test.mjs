import assert from "node:assert/strict";
import test from "node:test";
import { nominatimPlaceResults } from "../app/nominatim-places.mjs";

test("maps Portland, Oregon from Nominatim GeocodeJSON", () => {
  const places = nominatimPlaceResults({
    features: [
      {
        geometry: {
          coordinates: [-122.674194, 45.5202471],
          type: "Point",
        },
        properties: {
          geocoding: {
            country: "United States",
            country_code: "us",
            county: "Multnomah County",
            name: "Portland",
            place_id: 330096776,
            state: "Oregon",
            type: "city",
          },
        },
        type: "Feature",
      },
    ],
    type: "FeatureCollection",
  });

  assert.deepEqual(places, [
    {
      admin1: "Oregon",
      admin2: "Multnomah County",
      country: "United States",
      country_code: "US",
      feature_code: "PPL",
      id: 330096776,
      latitude: 45.5202471,
      longitude: -122.674194,
      name: "Portland",
      source: "osm",
    },
  ]);
});

test("ignores malformed Nominatim features", () => {
  assert.deepEqual(
    nominatimPlaceResults({
      features: [
        { properties: { geocoding: { name: "Missing coordinates" } } },
      ],
    }),
    [],
  );
});
