BINARY ?= wthrtxt
CURL ?= curl
DATA_DIR ?=
GEOLITE2_DB ?= GeoLite2-City.mmdb
GEOLITE2_URL ?= https://raw.githubusercontent.com/wp-statistics/GeoLite2-City/master/GeoLite2-City.mmdb.gz
GO ?= go
NPM ?= npm

.DEFAULT_GOAL := build

.PHONY: build assets run

build: assets $(GEOLITE2_DB)
	$(GO) build -o $(BINARY) ./cmd/wthrtxt

run: build
	@if [ -n "$(DATA_DIR)" ]; then mkdir -p "$(DATA_DIR)"; fi
	DATA_DIR="$(abspath $(DATA_DIR))" \
		GEOLITE2_DB="$(abspath $(GEOLITE2_DB))" \
		$(abspath $(BINARY))

assets:
	@if [ ! -d node_modules ] || \
		[ ! -f node_modules/.package-lock.json ] || \
		[ package.json -nt node_modules/.package-lock.json ] || \
		[ package-lock.json -nt node_modules/.package-lock.json ]; then \
		$(NPM) install; \
	fi
	$(NPM) run build

$(GEOLITE2_DB):
	@set -eu; \
		mkdir -p "$(dir $@)"; \
		archive="$@.gz.tmp"; \
		database="$@.tmp"; \
		trap 'rm -f "$$archive" "$$database"' 0 1 2 15; \
		$(CURL) --fail --location --show-error --silent --retry 3 \
			--output "$$archive" "$(GEOLITE2_URL)"; \
		gzip -t "$$archive"; \
		gzip -dc "$$archive" > "$$database"; \
		test -s "$$database"; \
		mv "$$database" "$@"
