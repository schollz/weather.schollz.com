const SUBDIVISION_TYPES = new Set([
  "borough",
  "district",
  "locality",
  "neighbourhood",
  "quarter",
  "suburb",
]);

/**
 * Convert Photon GeoJSON features into the place shape used by weather search
 * and readable location URLs.
 *
 * @param {unknown} value
 */
export function photonPlaceResults(value) {
  if (!value || typeof value !== "object") return [];

  const features = Array.isArray(value.features) ? value.features : [];

  return features.flatMap((feature, index) => {
    const properties = feature?.properties;
    const coordinates = feature?.geometry?.coordinates;

    if (!properties || !Array.isArray(coordinates)) return [];

    const longitude = Number(coordinates[0]);
    const latitude = Number(coordinates[1]);
    const countryCode =
      typeof properties.countrycode === "string"
        ? properties.countrycode.toUpperCase()
        : "";
    const name = [
      properties.name,
      properties.city,
      properties.town,
      properties.village,
      properties.locality,
    ].find((candidate) => typeof candidate === "string" && candidate.trim());

    if (
      !countryCode ||
      !name ||
      !Number.isFinite(latitude) ||
      !Number.isFinite(longitude)
    ) {
      return [];
    }

    const placeId = Number(properties.osm_id);
    const type = [properties.osm_value, properties.type].find(
      (candidate) => typeof candidate === "string" && candidate.trim(),
    );

    return [
      {
        admin1:
          typeof properties.state === "string" ? properties.state : undefined,
        admin2:
          typeof properties.county === "string"
            ? properties.county
            : typeof properties.district === "string"
              ? properties.district
              : undefined,
        country:
          typeof properties.country === "string"
            ? properties.country
            : countryCode,
        country_code: countryCode,
        feature_code:
          typeof type === "string" && SUBDIVISION_TYPES.has(type.toLowerCase())
            ? "PPLX"
            : "PPL",
        id: Number.isFinite(placeId) ? placeId : index + 1,
        latitude,
        longitude,
        name,
        source: "osm",
      },
    ];
  });
}
