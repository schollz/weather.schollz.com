import assert from "node:assert/strict";
import test from "node:test";
import { photonPlaceResults } from "../app/photon-places.mjs";

test("maps Portland, Oregon from Photon GeoJSON", () => {
  const places = photonPlaceResults({
    features: [
      {
        geometry: {
          coordinates: [-122.674194, 45.5202471],
          type: "Point",
        },
        properties: {
          country: "United States",
          countrycode: "US",
          county: "Multnomah",
          name: "Portland",
          osm_id: 186579,
          osm_key: "place",
          osm_type: "R",
          osm_value: "city",
          state: "Oregon",
          type: "city",
        },
        type: "Feature",
      },
    ],
    type: "FeatureCollection",
  });

  assert.deepEqual(places, [
    {
      admin1: "Oregon",
      admin2: "Multnomah",
      country: "United States",
      country_code: "US",
      feature_code: "PPL",
      id: 186579,
      latitude: 45.5202471,
      longitude: -122.674194,
      name: "Portland",
      source: "osm",
    },
  ]);
});

test("marks Photon subdivisions and ignores malformed features", () => {
  assert.deepEqual(
    photonPlaceResults({
      features: [
        {
          geometry: { coordinates: [-122.676439, 45.48825] },
          properties: {
            country: "United States",
            countrycode: "us",
            name: "South Portland",
            osm_id: 7709227,
            osm_value: "suburb",
            state: "Oregon",
          },
        },
        { properties: { name: "Missing coordinates" } },
      ],
    }),
    [
      {
        admin1: "Oregon",
        admin2: undefined,
        country: "United States",
        country_code: "US",
        feature_code: "PPLX",
        id: 7709227,
        latitude: 45.48825,
        longitude: -122.676439,
        name: "South Portland",
        source: "osm",
      },
    ],
  );
});
