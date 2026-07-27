export const SITE_NAME = "wthrtxt.com";
export const SITE_URL = "https://wthrtxt.com";
export const ROOT_TITLE =
  "Local Weather Forecast: Current, Hourly & 7-Day | wthrtxt.com";
export const ROOT_SOCIAL_TITLE =
  "Local Weather Forecast — Current, Hourly & 7-Day";
export const SITE_DESCRIPTION =
  "Get your local weather forecast with current conditions, hour-by-hour temperature, humidity and rain, a 7-day outlook, and record highs and lows worldwide.";
export const SOCIAL_IMAGE = {
  alt: "wthrtxt.com local weather, plain text — current, hourly and 7-day forecasts",
  height: 909,
  url: "/og.png",
  width: 1731,
};

export const siteJsonLd = {
  "@context": "https://schema.org",
  "@graph": [
    {
      "@id": `${SITE_URL}/#website`,
      "@type": "WebSite",
      alternateName: "wthrtxt",
      description: SITE_DESCRIPTION,
      inLanguage: "en-US",
      isAccessibleForFree: true,
      name: SITE_NAME,
      potentialAction: {
        "@type": "SearchAction",
        "query-input": "required name=location",
        target: {
          "@type": "EntryPoint",
          urlTemplate: `${SITE_URL}/{location}/`,
        },
      },
      url: `${SITE_URL}/`,
    },
    {
      "@id": `${SITE_URL}/#weather-app`,
      "@type": "WebApplication",
      alternateName: [
        "wthrtxt",
        "wthrtxt weather",
        "weather by curl",
      ],
      applicationCategory: "WeatherApplication",
      applicationSubCategory: "Local weather forecast",
      browserRequirements:
        "A modern web browser for the visual forecast; curl or another HTTP client for plaintext.",
      description: SITE_DESCRIPTION,
      featureList: [
        "Current local weather conditions",
        "Hour-by-hour temperature, humidity, wind and rain",
        "Seven-day local and worldwide weather forecast",
        "Observed and estimated daily temperature records",
        "Readable, shareable location URLs",
        "Plaintext weather forecasts for curl and terminals",
      ],
      isAccessibleForFree: true,
      isBasedOn: [
        "https://www.weather.gov/documentation/services-web-api",
        "https://open-meteo.com/en/docs",
        "https://docs.rcc-acis.org/acisws/",
      ],
      isPartOf: {
        "@id": `${SITE_URL}/#website`,
      },
      name: "wthrtxt.com Weather",
      offers: {
        "@type": "Offer",
        price: 0,
        priceCurrency: "USD",
      },
      operatingSystem: "Any",
      softwareHelp: {
        "@id": `${SITE_URL}/about/#about`,
        "@type": "AboutPage",
        url: `${SITE_URL}/about/`,
      },
      url: `${SITE_URL}/`,
    },
  ],
};
