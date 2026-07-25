# weather.schollz.com

A minimal local weather page powered by
[NOAA](https://www.weather.gov/documentation/services-web-api).

[Open weather.schollz.com](https://weather.schollz.com)

## About

The site uses your browser location or a U.S. place search to show current
conditions, hourly details, and a seven-day forecast. Forecast data comes
directly from NOAA, daily temperature records come from ACIS, and place search
uses Open-Meteo. Locations can be shared by URL, and no API keys or backend are
required.

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
