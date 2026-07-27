const SUBDIVISION_TYPES = new Set([
  "borough",
  "district",
  "neighbourhood",
  "quarter",
  "suburb",
]);

/**
 * Convert Nominatim's stable GeocodeJSON address categories into the place
 * shape used by weather search and readable location URLs.
 *
 * @param {unknown} value
 */
export function nominatimPlaceResults(value) {
  if (!value || typeof value !== "object") return [];

  const features = Array.isArray(value.features) ? value.features : [];

  return features.flatMap((feature, index) => {
    const geocoding = feature?.properties?.geocoding;
    const coordinates = feature?.geometry?.coordinates;

    if (!geocoding || !Array.isArray(coordinates)) return [];

    const longitude = Number(coordinates[0]);
    const latitude = Number(coordinates[1]);
    const countryCode =
      typeof geocoding.country_code === "string"
        ? geocoding.country_code.toUpperCase()
        : "";
    const name = [
      geocoding.name,
      geocoding.city,
      geocoding.town,
      geocoding.village,
      geocoding.municipality,
      geocoding.locality,
    ].find((candidate) => typeof candidate === "string" && candidate.trim());

    if (
      !countryCode ||
      !name ||
      !Number.isFinite(latitude) ||
      !Number.isFinite(longitude)
    ) {
      return [];
    }

    const placeId = Number(geocoding.place_id);
    const type =
      typeof geocoding.type === "string" ? geocoding.type.toLowerCase() : "";

    return [
      {
        admin1:
          typeof geocoding.state === "string" ? geocoding.state : undefined,
        admin2:
          typeof geocoding.county === "string"
            ? geocoding.county
            : typeof geocoding.district === "string"
              ? geocoding.district
              : undefined,
        country:
          typeof geocoding.country === "string"
            ? geocoding.country
            : countryCode,
        country_code: countryCode,
        feature_code: SUBDIVISION_TYPES.has(type) ? "PPLX" : "PPL",
        id: Number.isFinite(placeId) ? placeId : index + 1,
        latitude,
        longitude,
        name,
        source: "osm",
      },
    ];
  });
}
