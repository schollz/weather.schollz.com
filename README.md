# weather.schollz.com

A minimal worldwide weather page powered by
[NOAA](https://www.weather.gov/documentation/services-web-api) and
[Open-Meteo](https://open-meteo.com/en/docs).

[Open weather.schollz.com](https://weather.schollz.com)

## About

The site uses your browser location or a worldwide place search to show current
conditions, hourly details, and a seven-day forecast. Supported U.S. areas use
NOAA forecasts and station observations plus ACIS daily temperature records.
Everywhere else uses Open-Meteo forecasts and estimated ERA5-Land temperature
records. Place search uses Open-Meteo, and raw coordinates are named through a
cached OpenStreetMap Nominatim reverse lookup. Locations can be shared by URL,
and no API keys or backend are required.

Open-Meteo past-hour values and international records are model-based estimates,
not official station observations. The public endpoints used by this personal
site require noncommercial, moderate-volume use and source attribution.

## Development

Requires Node.js 22.13 or newer.

```bash
npm install
npm run dev
```

Useful checks:

```bash
npm test
npm run lint
npm run build:pages
```

Pushes to `main` are deployed to GitHub Pages.
