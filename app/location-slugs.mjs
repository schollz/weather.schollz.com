const US_STATE_CODES = {
  alabama: "al",
  alaska: "ak",
  arizona: "az",
  arkansas: "ar",
  california: "ca",
  colorado: "co",
  connecticut: "ct",
  delaware: "de",
  "district-of-columbia": "dc",
  florida: "fl",
  georgia: "ga",
  hawaii: "hi",
  idaho: "id",
  illinois: "il",
  indiana: "in",
  iowa: "ia",
  kansas: "ks",
  kentucky: "ky",
  louisiana: "la",
  maine: "me",
  maryland: "md",
  massachusetts: "ma",
  michigan: "mi",
  minnesota: "mn",
  mississippi: "ms",
  missouri: "mo",
  montana: "mt",
  nebraska: "ne",
  nevada: "nv",
  "new-hampshire": "nh",
  "new-jersey": "nj",
  "new-mexico": "nm",
  "new-york": "ny",
  "north-carolina": "nc",
  "north-dakota": "nd",
  ohio: "oh",
  oklahoma: "ok",
  oregon: "or",
  pennsylvania: "pa",
  "rhode-island": "ri",
  "south-carolina": "sc",
  "south-dakota": "sd",
  tennessee: "tn",
  texas: "tx",
  utah: "ut",
  vermont: "vt",
  virginia: "va",
  washington: "wa",
  "west-virginia": "wv",
  wisconsin: "wi",
  wyoming: "wy",
};

/**
 * @typedef {{
 *   admin1?: string;
 *   admin2?: string;
 *   country_code: string;
 *   feature_code?: string;
 *   name: string;
 * }} SlugPlace
 */

/**
 * Convert a human-readable place name into its URL-safe representation.
 *
 * @param {string} value
 */
export function slugifyPlaceName(value) {
  return value
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[’']/g, "")
    .toLowerCase()
    .replace(/&/g, " and ")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

/**
 * @param {SlugPlace} place
 */
function regionSlug(place) {
  const region = slugifyPlaceName(place.admin1 ?? place.admin2 ?? "");

  if (place.country_code.toUpperCase() === "US") {
    if (/^[a-z]{2}$/.test(region)) return region;
    return US_STATE_CODES[region] ?? region;
  }

  return region || place.country_code.toLowerCase();
}

/**
 * Ignore populated-place subdivisions when deciding whether a city name needs
 * disambiguation. For example, Seattle and a neighborhood named Seattle in
 * another country should not force Seattle, Washington to use a suffix.
 *
 * @param {SlugPlace} place
 */
function isNamedPlace(place) {
  return place.feature_code !== "PPLX";
}

/**
 * Use the short city slug when it identifies one named place in the country.
 * If the country contains multiple cities with the same name, append a region
 * identifier such as `or` in `hillsboro-or`.
 *
 * @param {SlugPlace} place
 * @param {SlugPlace[]} [candidates]
 */
export function locationSlugForPlace(place, candidates = []) {
  const citySlug = slugifyPlaceName(place.name);
  if (!citySlug) return "";

  const countryCode = place.country_code.toUpperCase();
  const duplicates = candidates.filter(
    (candidate) =>
      isNamedPlace(candidate) &&
      candidate.country_code.toUpperCase() === countryCode &&
      slugifyPlaceName(candidate.name) === citySlug,
  );

  if (duplicates.length <= 1) return citySlug;

  const suffix = regionSlug(place);
  return suffix ? `${citySlug}-${suffix}` : citySlug;
}

/**
 * Generate progressively shorter search terms so a canonical slug with a
 * region suffix can be resolved through a place API that only accepts names.
 *
 * @param {string} slug
 */
export function searchTermsForLocationSlug(slug) {
  const normalized = slugifyPlaceName(slug);
  const parts = normalized.split("-").filter(Boolean);
  const terms = [];

  for (let length = parts.length; length > 0; length -= 1) {
    const term = parts.slice(0, length).join(" ");
    if (term.length >= 2 && !terms.includes(term)) terms.push(term);
  }

  return terms;
}
