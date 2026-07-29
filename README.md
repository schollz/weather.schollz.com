# wthrtxt.com

A minimal worldwide weather page powered by
[NOAA](https://www.weather.gov/documentation/services-web-api) and
[Open-Meteo](https://open-meteo.com/en/docs).

[Open wthrtxt.com](https://wthrtxt.com)

## About

The site uses your browser location or a worldwide place search to show current
conditions, hourly details, and a seven-day forecast. Supported U.S. areas use
NOAA forecasts and station observations plus ACIS daily temperature records and
historical averages. Everywhere else uses Open-Meteo forecasts and estimated
ERA5-Land temperature records and averages. U.S. locations display Fahrenheit,
inches, and mph; other locations automatically display Celsius, millimeters,
and km/h. Live place search uses OpenStreetMap Photon, with OpenStreetMap
Nominatim used for noncanonical place links and reverse lookup for raw
coordinates. Locations use readable, shareable URL slugs such as `/seattle/`
and `/hillsboro-or/`; ambiguous U.S. city names include their state
abbreviation. Inputs such as `/portland%20oregon/` resolve to the canonical
`/portland-or/` location. Browser requests go directly to the listed providers;
terminal requests are fetched and rendered by the Go server.

Open-Meteo past-hour values and international climate history are model-based
estimates, not official station observations. The public endpoints used by this
personal site require noncommercial, moderate-volume use and source attribution.

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
npm install
npm run dev
```

`npm run dev` runs the browser application. `make` builds those assets into the
Go executable and downloads, integrity-checks, and decompresses
[GeoLite2-City.mmdb.gz](https://github.com/wp-statistics/GeoLite2-City/blob/master/GeoLite2-City.mmdb.gz)
when the local database is missing:

```bash
make
make run
```

Useful checks:

```bash
npm test
npm run lint
go test -race ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
docker build -t wthrtxt .
```

The production image is deployed to Disco and serves both the embedded browser
application and plaintext terminal forecasts on port `8080`. The Docker build
downloads and validates GeoLite2 itself; it does not require MaxMind credentials
or BuildKit secrets.

Outside Docker, the persistent BoltDB cache defaults to
`filepath.Join(os.TempDir(), "wthrtxt")`. Set `DATA_DIR` to choose a different
location. The production image sets it to the Disco volume at `/data`.

Runtime configuration:

| Variable | Default outside Docker | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP listening port |
| `DATA_DIR` | `filepath.Join(os.TempDir(), "wthrtxt")` | BoltDB cache directory |
| `GEOLITE2_DB` | `/opt/wthrtxt/GeoLite2-City.mmdb` | GeoLite2 City database |
| `TRUST_PROXY_HEADERS` | `false` | Trust Caddy's client-IP headers |

GeoLite2 data is created by [MaxMind](https://www.maxmind.com) and redistributed
by the [wp-statistics GeoLite2-City project](https://github.com/wp-statistics/GeoLite2-City)
under CC BY-SA 4.0. IP-derived locations are approximate.
