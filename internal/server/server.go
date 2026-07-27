package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"wthrtxt.com/internal/weather"
)

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
		static:            http.FileServer(http.FS(assets)),
	}
}

func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("X-Content-Type-Options", "nosniff")
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
	index, err := fs.ReadFile(s.Assets, "index.html")
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

func (s *Server) isStaticPath(requestPath string) bool {
	clean := strings.TrimPrefix(path.Clean(requestPath), "/")
	if clean == "." || clean == "" || clean == "index.html" {
		return false
	}
	if !strings.HasPrefix(clean, "assets/") && clean != "favicon.svg" {
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

	response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if fromIP {
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
