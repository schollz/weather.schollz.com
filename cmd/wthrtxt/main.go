package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"wthrtxt.com/internal/server"
	"wthrtxt.com/internal/store"
	"wthrtxt.com/internal/weather"
	"wthrtxt.com/internal/web"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("wthrtxt.com stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	assets, err := web.Files()
	if err != nil {
		return fmt.Errorf("load embedded browser application: %w", err)
	}

	dataDirectory := cacheDirectory()
	cachePath := filepath.Join(dataDirectory, "cache.db")
	cache, err := store.Open(cachePath, 256)
	if err != nil {
		return fmt.Errorf("open persistent cache: %w", err)
	}
	defer cache.Close()
	cachePending := !cache.Persistent()
	if cachePending {
		logger.Warn(
			"persistent cache is locked; serving from memory during deployment handoff",
			"path",
			cachePath,
		)
	}

	geoPath := environment(
		"GEOLITE2_DB",
		"/opt/wthrtxt/GeoLite2-City.mmdb",
	)
	geoIP, err := weather.OpenGeoIP(geoPath)
	if err != nil {
		return fmt.Errorf("validate GeoLite2 database: %w", err)
	}
	defer geoIP.Close()

	httpClient := weather.NewHTTPClient()
	geocoder := weather.NewGeocoder(httpClient, cache)
	weatherService := weather.NewService(
		cache,
		weather.NewNOAA(httpClient),
		weather.NewOpenMeteo(httpClient),
		weather.NewRecordsClient(httpClient),
		logger,
	)
	handler := server.New(
		assets,
		weatherService,
		geocoder,
		geoIP,
		environmentBool("TRUST_PROXY_HEADERS", false),
		logger,
	)
	handler.UmamiURL = environment("UMAMI_URL", "")
	handler.UmamiWebsiteID = environment("UMAMI_WEBSITE_ID", "")

	httpServer := &http.Server{
		Addr:              ":" + environment("PORT", "8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	shutdownContext, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()
	if cachePending {
		go func() {
			if err := cache.WaitForPersistence(shutdownContext); err != nil {
				if !errors.Is(err, context.Canceled) {
					logger.Error("attach persistent cache", "error", err)
				}
				return
			}
			logger.Info("persistent cache attached", "path", cachePath)
		}()
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("wthrtxt.com listening", "address", httpServer.Addr)
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server: %w", err)
		}
		return nil
	case <-shutdownContext.Done():
		logger.Info("shutting down")
		serverContext, cancelServer := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelServer()
		if err := httpServer.Shutdown(serverContext); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			_ = httpServer.Close()
		}
		recordsContext, cancelRecords := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelRecords()
		if err := weatherService.Close(recordsContext); err != nil {
			logger.Warn("background work did not stop cleanly", "error", err)
		}
		return nil
	}
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func environmentBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func cacheDirectory() string {
	if configured := os.Getenv("DATA_DIR"); configured != "" {
		return configured
	}
	return filepath.Join(os.TempDir(), "wthrtxt")
}
