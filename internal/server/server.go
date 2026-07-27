package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"wthrtxt.com/internal/store"
	"wthrtxt.com/internal/weather"
)

const browserWeatherCacheCookie = "wthrtxt_weather_cache"

type Reporter interface {
	Report(context.Context, weather.Location) (weather.WeatherReport, error)
}

type Geocoder interface {
	ResolveSlug(context.Context, string) (weather.Location, error)
	Reverse(context.Context, float64, float64) (weather.Location, error)
}

type IPResolver interface {
	Resolve(netip.Addr) (weather.Location, error)
}

type Server struct {
	Assets            fs.FS
	Reporter          Reporter
	Geocoder          Geocoder
	IPResolver        IPResolver
	TrustProxyHeaders bool
	Logger            *slog.Logger
	Now               func() time.Time
	BrowserCache      *store.Store
	static            http.Handler
}

func New(
	assets fs.FS,
	reporter Reporter,
	geocoder Geocoder,
	ipResolver IPResolver,
	trustProxyHeaders bool,
	logger *slog.Logger,
) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		Assets:            assets,
		Reporter:          reporter,
		Geocoder:          geocoder,
		IPResolver:        ipResolver,
		TrustProxyHeaders: trustProxyHeaders,
		Logger:            logger,
		Now:               time.Now,
		BrowserCache:      store.NewMemory(512),
		static:            http.FileServer(http.FS(assets)),
	}
}

func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if request.URL.Path == "/api/weather-cache" {
		s.serveBrowserWeatherCache(response, request)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", "GET, HEAD")
		s.writeTextError(response, request, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if request.URL.Path == "/healthz" {
		response.Header().Set("Content-Type", "text/plain; charset=utf-8")
		response.Header().Set("Cache-Control", "no-store")
		response.WriteHeader(http.StatusOK)
		if request.Method != http.MethodHead {
			_, _ = response.Write([]byte("ok\n"))
		}
		return
	}

	if s.isStaticPath(request.URL.Path) {
		s.serveStatic(response, request)
		return
	}

	response.Header().Set("Vary", "User-Agent, Accept")
	format, err := negotiateFormat(request)
	if err != nil {
		s.writeTextError(response, request, http.StatusBadRequest, err.Error())
		return
	}
	if format == "html" {
		s.serveBrowser(response, request)
		return
	}
	s.serveTerminal(response, request)
}

func (s *Server) serveBrowser(response http.ResponseWriter, request *http.Request) {
	indexPath := "index.html"
	if strings.Trim(request.URL.Path, "/") == "about" {
		indexPath = "about/index.html"
	}
	index, err := fs.ReadFile(s.Assets, indexPath)
	if err != nil {
		s.writeTextError(response, request, http.StatusServiceUnavailable, "browser application has not been built")
		return
	}
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Cache-Control", "no-cache")
	response.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = response.Write(index)
	}
}

type browserWeatherCacheRequest struct {
	MaxAgeSeconds int                   `json:"max_age_seconds"`
	Path          string                `json:"path"`
	Report        weather.WeatherReport `json:"report"`
}

func (s *Server) serveBrowserWeatherCache(
	response http.ResponseWriter,
	request *http.Request,
) {
	response.Header().Set("Cache-Control", "no-store")
	if request.Method != http.MethodPost {
		response.Header().Set("Allow", "POST")
		s.writeTextError(response, request, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if s.BrowserCache == nil {
		s.writeTextError(response, request, http.StatusServiceUnavailable, "browser cache is unavailable")
		return
	}

	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		s.writeTextError(response, request, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}

	request.Body = http.MaxBytesReader(response, request.Body, 1<<20)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	var payload browserWeatherCacheRequest
	if err := decoder.Decode(&payload); err != nil {
		s.writeTextError(response, request, http.StatusBadRequest, "invalid browser weather cache payload")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		s.writeTextError(response, request, http.StatusBadRequest, "invalid browser weather cache payload")
		return
	}

	cachePath, ok := normalizeBrowserWeatherPath(payload.Path)
	if !ok || payload.MaxAgeSeconds < 1 || payload.MaxAgeSeconds > 60*60 {
		s.writeTextError(response, request, http.StatusBadRequest, "invalid browser weather cache metadata")
		return
	}
	if err := validateBrowserWeatherReport(payload.Report, s.Now()); err != nil {
		s.writeTextError(response, request, http.StatusBadRequest, "invalid browser weather report")
		return
	}

	session, ok := browserWeatherSession(request)
	if !ok {
		session, err = newBrowserWeatherSession()
		if err != nil {
			s.writeTextError(response, request, http.StatusServiceUnavailable, "browser cache is unavailable")
			return
		}
	}

	payload.Report.Location.CanonicalSlug = ""
	payload.Report.LocationFromIP = false
	payload.Report.RecordsSource = ""
	payload.Report.RecordsState = ""
	for index := range payload.Report.Daily {
		payload.Report.Daily[index].Record = nil
	}

	expiresAt := s.Now().Add(time.Duration(payload.MaxAgeSeconds) * time.Second)
	if err := s.BrowserCache.Put(
		browserWeatherCacheKey(session, cachePath),
		payload.Report,
		expiresAt,
	); err != nil {
		s.writeTextError(response, request, http.StatusServiceUnavailable, "browser cache is unavailable")
		return
	}

	http.SetCookie(response, &http.Cookie{
		Name:     browserWeatherCacheCookie,
		Value:    session,
		Path:     "/",
		MaxAge:   60 * 60,
		Expires:  s.Now().Add(time.Hour),
		HttpOnly: true,
		Secure: request.TLS != nil ||
			(s.TrustProxyHeaders &&
				strings.EqualFold(request.Header.Get("X-Forwarded-Proto"), "https")),
		SameSite: http.SameSiteLaxMode,
	})
	response.WriteHeader(http.StatusNoContent)
}

func (s *Server) browserWeatherReport(
	request *http.Request,
) (weather.WeatherReport, bool) {
	if s.BrowserCache == nil {
		return weather.WeatherReport{}, false
	}
	session, ok := browserWeatherSession(request)
	if !ok {
		return weather.WeatherReport{}, false
	}
	cachePath, ok := normalizeBrowserWeatherPath(request.URL.EscapedPath())
	if !ok {
		return weather.WeatherReport{}, false
	}

	var report weather.WeatherReport
	if !s.BrowserCache.Get(browserWeatherCacheKey(session, cachePath), &report) {
		return weather.WeatherReport{}, false
	}
	return report, true
}

func browserWeatherSession(request *http.Request) (string, bool) {
	cookie, err := request.Cookie(browserWeatherCacheCookie)
	if err != nil || len(cookie.Value) != 32 {
		return "", false
	}
	if _, err := hex.DecodeString(cookie.Value); err != nil {
		return "", false
	}
	return cookie.Value, true
}

func newBrowserWeatherSession() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func browserWeatherCacheKey(session, cachePath string) string {
	return "browser-weather:" + session + ":" + cachePath
}

func normalizeBrowserWeatherPath(value string) (string, bool) {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return "", false
	}
	trimmed := strings.Trim(decoded, "/")
	if len(trimmed) > 100 || strings.Contains(trimmed, "/") {
		return "", false
	}
	if trimmed == "" {
		return "/", true
	}
	return "/" + trimmed + "/", true
}

func validateBrowserWeatherReport(
	report weather.WeatherReport,
	now time.Time,
) error {
	if report.Location.Latitude < -90 || report.Location.Latitude > 90 ||
		report.Location.Longitude < -180 || report.Location.Longitude > 180 ||
		!finite(report.Location.Latitude) || !finite(report.Location.Longitude) {
		return errors.New("invalid coordinates")
	}
	if report.Location.TimeZone == "" {
		return errors.New("missing timezone")
	}
	if _, err := time.LoadLocation(report.Location.TimeZone); err != nil {
		return errors.New("invalid timezone")
	}
	if report.Provider != "NOAA" && report.Provider != "Open-Meteo" {
		return errors.New("invalid provider")
	}
	if len(report.Hourly) == 0 || len(report.Hourly) > 200 ||
		len(report.Daily) == 0 || len(report.Daily) > 14 {
		return errors.New("invalid forecast length")
	}
	if !shortWeatherText(report.Location.Name, 200) ||
		!shortWeatherText(report.Location.Region, 200) ||
		!shortWeatherText(report.Location.Country, 200) ||
		!shortWeatherText(report.Location.CountryCode, 8) ||
		!shortWeatherText(report.Location.Source, 100) ||
		!shortWeatherText(report.Current.Sky, 200) ||
		!shortWeatherText(report.Current.Wind, 100) ||
		!shortWeatherText(report.Current.Source, 40) ||
		!shortWeatherText(report.StationID, 40) {
		return errors.New("weather text is too long")
	}
	if !boundedFloat(report.Current.Temperature, -200, 200) ||
		!boundedFloat(report.Current.Humidity, 0, 100) ||
		(!report.Current.ObservedAt.IsZero() &&
			!plausibleWeatherTime(report.Current.ObservedAt, now)) {
		return errors.New("invalid current conditions")
	}

	for _, reading := range report.Hourly {
		if !plausibleWeatherTime(reading.StartTime, now) ||
			!plausibleWeatherTime(reading.EndTime, now) ||
			!reading.EndTime.After(reading.StartTime) ||
			!boundedFloat(reading.Temperature, -200, 200) ||
			!boundedFloat(reading.Humidity, 0, 100) ||
			!boundedFloat(reading.PrecipChance, 0, 100) ||
			!boundedFloat(reading.PrecipInches, 0, 100) ||
			!shortWeatherText(reading.Sky, 200) ||
			!shortWeatherText(reading.ObservationKind, 40) {
			return errors.New("invalid hourly forecast")
		}
	}
	for _, daily := range report.Daily {
		if !plausibleWeatherTime(daily.Date, now) ||
			!boundedFloat(daily.High, -200, 200) ||
			!boundedFloat(daily.Low, -200, 200) ||
			!boundedFloat(daily.PrecipChance, 0, 100) ||
			!shortWeatherText(daily.Sky, 200) {
			return errors.New("invalid daily forecast")
		}
	}
	return nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func boundedFloat(value *float64, minimum, maximum float64) bool {
	return value == nil ||
		(finite(*value) && *value >= minimum && *value <= maximum)
}

func plausibleWeatherTime(value, now time.Time) bool {
	return value.After(now.Add(-72*time.Hour)) &&
		value.Before(now.Add(10*24*time.Hour))
}

func shortWeatherText(value string, maximum int) bool {
	return len(value) <= maximum
}

func (s *Server) isStaticPath(requestPath string) bool {
	clean := strings.TrimPrefix(path.Clean(requestPath), "/")
	if clean == "." || clean == "" || clean == "index.html" {
		return false
	}
	if clean == "404.html" || strings.HasSuffix(clean, ".rsc") {
		return false
	}
	info, err := fs.Stat(s.Assets, clean)
	return err == nil && !info.IsDir()
}

func (s *Server) serveStatic(response http.ResponseWriter, request *http.Request) {
	extension := path.Ext(request.URL.Path)
	if contentType := mime.TypeByExtension(extension); contentType != "" {
		response.Header().Set("Content-Type", contentType)
	}
	if strings.HasPrefix(request.URL.Path, "/assets/") {
		response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		response.Header().Set("Cache-Control", "public, max-age=3600")
	}
	s.static.ServeHTTP(response, request)
}

func (s *Server) serveTerminal(response http.ResponseWriter, request *http.Request) {
	if report, found := s.browserWeatherReport(request); found {
		s.writeTerminalReport(response, request, report, false, true)
		return
	}

	location, fromIP, status, err := s.resolveLocation(request)
	if err != nil {
		s.writeTextError(response, request, status, err.Error())
		return
	}
	if location.CanonicalSlug != "" {
		requestSlug := strings.Trim(request.URL.Path, "/")
		if requestSlug != location.CanonicalSlug {
			target := url.URL{
				Path:     "/" + location.CanonicalSlug + "/",
				RawQuery: request.URL.RawQuery,
			}
			http.Redirect(response, request, target.String(), http.StatusPermanentRedirect)
			return
		}
	}

	report, err := s.Reporter.Report(request.Context(), location)
	if err != nil {
		s.Logger.Warn("weather request failed", "error", err)
		status = http.StatusBadGateway
		if errors.Is(err, context.DeadlineExceeded) {
			status = http.StatusGatewayTimeout
		}
		s.writeTextError(response, request, status, "weather data is temporarily unavailable")
		return
	}
	report.LocationFromIP = fromIP

	s.writeTerminalReport(response, request, report, fromIP, false)
}

func (s *Server) writeTerminalReport(
	response http.ResponseWriter,
	request *http.Request,
	report weather.WeatherReport,
	fromIP bool,
	fromBrowserCache bool,
) {
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if fromBrowserCache {
		response.Header().Set("Cache-Control", "private, no-store")
		response.Header().Set("Vary", "User-Agent, Accept, Cookie")
	} else if fromIP {
		response.Header().Set("Cache-Control", "private, no-store")
	} else {
		response.Header().Set("Cache-Control", "public, max-age=300")
	}
	response.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = response.Write([]byte(weather.RenderText(report, s.Now())))
	}
}

func (s *Server) resolveLocation(request *http.Request) (weather.Location, bool, int, error) {
	if value := strings.TrimSpace(request.URL.Query().Get("location")); value != "" {
		latitude, longitude, ok := parseCoordinates(value)
		if !ok {
			return weather.Location{}, false, http.StatusBadRequest,
				errors.New("invalid location; expected latitude,longitude")
		}
		location := s.reverseOrCoordinates(request.Context(), latitude, longitude)
		return location, false, 0, nil
	}

	requestPath, err := url.PathUnescape(strings.Trim(request.URL.Path, "/"))
	if err != nil {
		return weather.Location{}, false, http.StatusBadRequest, errors.New("invalid location path")
	}
	if requestPath != "" {
		if strings.Contains(requestPath, "/") {
			return weather.Location{}, false, http.StatusNotFound, errors.New("location not found")
		}
		if latitude, longitude, ok := parseCoordinates(requestPath); ok {
			return s.reverseOrCoordinates(request.Context(), latitude, longitude), false, 0, nil
		}
		if strings.Contains(requestPath, ",") {
			return weather.Location{}, false, http.StatusBadRequest,
				errors.New("invalid coordinates")
		}
		location, resolveErr := s.Geocoder.ResolveSlug(request.Context(), requestPath)
		if resolveErr != nil {
			return weather.Location{}, false, http.StatusNotFound, errors.New("location not found")
		}
		return location, false, 0, nil
	}

	address, err := clientAddress(request, s.TrustProxyHeaders)
	if err != nil {
		return weather.Location{}, true, http.StatusBadRequest,
			errors.New("could not determine your public IP; try /seattle or /47.6062,-122.3321")
	}
	location, err := s.IPResolver.Resolve(address)
	if err != nil {
		return weather.Location{}, true, http.StatusBadRequest,
			errors.New("could not locate your IP; try /seattle or /47.6062,-122.3321")
	}
	return location, true, 0, nil
}

func (s *Server) reverseOrCoordinates(ctx context.Context, latitude, longitude float64) weather.Location {
	location, err := s.Geocoder.Reverse(ctx, latitude, longitude)
	if err == nil {
		return location
	}
	return weather.Location{
		Name:      fmt.Sprintf("%.3f, %.3f", latitude, longitude),
		Latitude:  latitude,
		Longitude: longitude,
		Source:    "coordinates",
	}
}

func (s *Server) writeTextError(
	response http.ResponseWriter,
	request *http.Request,
	status int,
	message string,
) {
	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	if request.Method != http.MethodHead {
		_, _ = fmt.Fprintf(response, "wthrtxt.com: %s\n", message)
	}
}

func negotiateFormat(request *http.Request) (string, error) {
	if value, present := request.URL.Query()["format"]; present {
		format := ""
		if len(value) > 0 {
			format = strings.ToLower(strings.TrimSpace(value[0]))
		}
		switch format {
		case "html", "text":
			return format, nil
		default:
			return "", errors.New("format must be html or text")
		}
	}

	userAgent := strings.ToLower(request.UserAgent())
	terminalAgents := []string{
		"curl/", "wget/", "httpie/", "powershell", "libwww-perl", "xh/",
	}
	for _, terminalAgent := range terminalAgents {
		if strings.Contains(userAgent, terminalAgent) {
			return "text", nil
		}
	}
	if strings.Contains(strings.ToLower(request.Header.Get("Accept")), "text/html") {
		return "html", nil
	}
	return "text", nil
}

func parseCoordinates(value string) (float64, float64, bool) {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	latitude, latitudeErr := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	longitude, longitudeErr := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if latitudeErr != nil || longitudeErr != nil ||
		latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return 0, 0, false
	}
	return latitude, longitude, true
}

func clientAddress(request *http.Request, trustProxy bool) (netip.Addr, error) {
	candidates := make([]string, 0, 3)
	if trustProxy {
		for _, forwarded := range strings.Split(request.Header.Get("X-Forwarded-For"), ",") {
			if value := strings.TrimSpace(forwarded); value != "" {
				candidates = append(candidates, value)
			}
		}
		if realIP := strings.TrimSpace(request.Header.Get("X-Real-IP")); realIP != "" {
			candidates = append(candidates, realIP)
		}
	}

	remote := request.RemoteAddr
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	candidates = append(candidates, remote)

	for _, candidate := range candidates {
		address, err := netip.ParseAddr(strings.Trim(candidate, "[]"))
		if err == nil && isPublicAddress(address.Unmap()) {
			return address.Unmap(), nil
		}
	}
	return netip.Addr{}, errors.New("no public client address")
}

func isPublicAddress(address netip.Addr) bool {
	if !address.IsValid() || !address.IsGlobalUnicast() ||
		address.IsPrivate() || address.IsLoopback() ||
		address.IsLinkLocalUnicast() || address.IsUnspecified() {
		return false
	}
	for _, prefix := range nonPublicPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

var nonPublicPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("2001:db8::/32"),
}
