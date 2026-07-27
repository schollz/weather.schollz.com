# syntax=docker/dockerfile:1

FROM node:22-bookworm-slim AS frontend
WORKDIR /src
COPY package.json package-lock.json ./
RUN npm ci
COPY app ./app
COPY public ./public
COPY scripts ./scripts
COPY eslint.config.mjs next.config.ts tsconfig.json vite.config.ts ./
RUN npm run build

FROM alpine:3.23 AS geolite
ADD https://raw.githubusercontent.com/wp-statistics/GeoLite2-City/master/GeoLite2-City.mmdb.gz /tmp/GeoLite2-City.mmdb.gz
RUN gzip -t /tmp/GeoLite2-City.mmdb.gz \
    && mkdir -p /opt/wthrtxt \
    && gzip -dc /tmp/GeoLite2-City.mmdb.gz > /opt/wthrtxt/GeoLite2-City.mmdb \
    && test -s /opt/wthrtxt/GeoLite2-City.mmdb

FROM golang:1.26-bookworm AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=frontend /src/dist/client/ ./internal/web/public/
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/wthrtxt \
    ./cmd/wthrtxt

FROM alpine:3.23 AS runtime-files
RUN mkdir -p /data /opt/wthrtxt \
    && chown -R 65532:65532 /data /opt/wthrtxt

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=runtime-files --chown=65532:65532 /data /data
COPY --from=go-builder --chown=65532:65532 /out/wthrtxt /wthrtxt
COPY --from=geolite --chown=65532:65532 /opt/wthrtxt/GeoLite2-City.mmdb /opt/wthrtxt/GeoLite2-City.mmdb

ENV PORT=8080 \
    DATA_DIR=/data \
    GEOLITE2_DB=/opt/wthrtxt/GeoLite2-City.mmdb \
    TRUST_PROXY_HEADERS=true

EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/wthrtxt"]
