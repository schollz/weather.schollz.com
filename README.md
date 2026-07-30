<p align="center">
  <a href="https://wthrtxt.com"><img
    src="public/wthrtxt-graphic.png"
    width="454"
    alt="wthrtxt logo: weather with lines"
  ></a>
</p>

<p align="center">local weather without the clutter</p>


<p align="center">
  <a href="https://github.com/schollz/wthrtxt/actions/workflows/ci.yml">
    <img
      src="https://github.com/schollz/wthrtxt/actions/workflows/ci.yml/badge.svg?branch=main"
      alt="CI status"
    >
  </a>
  <a href="https://github.com/schollz/wthrtxt/releases/latest">
    <img
      src="https://img.shields.io/github/v/release/schollz/wthrtxt"
      alt="Latest release"
    >
  </a>
  <a href="https://github.com/sponsors/schollz"><img alt="GitHub Sponsors" src="https://img.shields.io/github/sponsors/schollz"></a>
</p>

A minimal worldwide weather page powered by
[NOAA](https://www.weather.gov/documentation/services-web-api) and
[Open-Meteo](https://open-meteo.com/en/docs).

Try it at [wthrtxt.com](https://wthrtxt.com)

## About

The site uses your browser location or a worldwide place search to show current conditions, hourly details, and a seven-day forecast. 

Supported U.S. areas use NOAA forecasts and station observations plus ACIS daily temperature records and historical averages. Everywhere else uses Open-Meteo forecasts and estimated
ERA5-Land temperature records and averages. 

## Terminal

The server selects plaintext for curl, wget, HTTPie, and PowerShell:

```bash
curl https://wthrtxt.com
curl https://wthrtxt.com/seattle
curl https://wthrtxt.com/hillsboro-or
curl https://wthrtxt.com/47.6062,-122.3321
```

The root route estimates the caller's city from the bundled GeoLite2 database.
Explicit locations can use a readable place slug or latitude and longitude.
Use `?format=text` or `?format=html` to override automatic content negotiation.

## Development

Requires Node.js 22.13 or newer and Go 1.24 or newer.

```bash
make serve
```

Useful checks:

```bash
npm test
npm run lint
go test -race ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
docker build -t wthrtxt .
```

## Production 

The production image is deployed uising [Disco](https://disco.cloud/).

Runtime configuration:

| Variable | Default outside Docker | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP listening port |
| `DATA_DIR` | `filepath.Join(os.TempDir(), "wthrtxt")` | BoltDB cache directory |
| `GEOLITE2_DB` | `/opt/wthrtxt/GeoLite2-City.mmdb` | GeoLite2 City database |
| `TRUST_PROXY_HEADERS` | `false` | Trust Caddy's client-IP headers |
| `UMAMI_URL` | unset | Umami base URL, such as `https://umami.schollz.com` |
| `UMAMI_WEBSITE_ID` | unset | Umami website ID, such as `website-uuid` |

## Acknowledgements 

GeoLite2 data is created by [MaxMind](https://www.maxmind.com) and redistributed
by the [wp-statistics GeoLite2-City project](https://github.com/wp-statistics/GeoLite2-City)
under CC BY-SA 4.0. IP-derived locations are approximate.
