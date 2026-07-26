import assert from "node:assert/strict";
import test from "node:test";
import {
  locationSlugForPlace,
  searchTermsForLocationSlug,
  slugifyPlaceName,
} from "../app/location-slugs.mjs";

const seattle = {
  admin1: "Washington",
  country_code: "US",
  feature_code: "PPLA2",
  name: "Seattle",
};
const hillsboroOregon = {
  admin1: "Oregon",
  country_code: "US",
  feature_code: "PPLA2",
  name: "Hillsboro",
};
const hillsboroTexas = {
  admin1: "Texas",
  country_code: "US",
  feature_code: "PPLA2",
  name: "Hillsboro",
};

test("creates readable ASCII place slugs", () => {
  assert.equal(slugifyPlaceName("São José"), "sao-jose");
  assert.equal(slugifyPlaceName("Coeur d’Alene"), "coeur-dalene");
  assert.equal(slugifyPlaceName("Washington, D.C."), "washington-d-c");
});

test("keeps a unique city short even when another country has the name", () => {
  const mexicoNeighborhood = {
    admin1: "Jalisco",
    country_code: "MX",
    feature_code: "PPLX",
    name: "Seattle",
  };

  assert.equal(
    locationSlugForPlace(seattle, [seattle, mexicoNeighborhood]),
    "seattle",
  );
});

test("adds a state abbreviation when a US city name is ambiguous", () => {
  const candidates = [hillsboroOregon, hillsboroTexas];

  assert.equal(
    locationSlugForPlace(hillsboroOregon, candidates),
    "hillsboro-or",
  );
  assert.equal(
    locationSlugForPlace(hillsboroTexas, candidates),
    "hillsboro-tx",
  );
});

test("builds fallback search terms for suffixed and multi-word slugs", () => {
  assert.deepEqual(searchTermsForLocationSlug("hillsboro-or"), [
    "hillsboro or",
    "hillsboro",
  ]);
  assert.deepEqual(searchTermsForLocationSlug("san-jose-ca"), [
    "san jose ca",
    "san jose",
    "san",
  ]);
});
