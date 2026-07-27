package weather

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const upstreamUserAgent = "wthrtxt.com/1.0 (+https://wthrtxt.com)"

type UpstreamError struct {
	Provider string
	Status   int
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("%s request failed (%d)", e.Provider, e.Status)
}

func IsStatus(err error, status int) bool {
	var upstream *UpstreamError
	return errors.As(err, &upstream) && upstream.Status == status
}

type HTTPClient struct {
	Client  *http.Client
	Retries int
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{
			Timeout: 12 * time.Second,
			Transport: &http.Transport{
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          64,
				MaxIdleConnsPerHost:   8,
				IdleConnTimeout:       90 * time.Second,
				ResponseHeaderTimeout: 8 * time.Second,
			},
		},
		Retries: 2,
	}
}

func (c *HTTPClient) JSON(
	ctx context.Context,
	provider string,
	method string,
	url string,
	contentType string,
	body []byte,
	destination any,
) error {
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}

	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		var requestBody io.Reader
		if body != nil {
			requestBody = bytes.NewReader(body)
		}

		request, err := http.NewRequestWithContext(ctx, method, url, requestBody)
		if err != nil {
			return err
		}
		request.Header.Set("Accept", "application/json, application/geo+json")
		request.Header.Set("User-Agent", upstreamUserAgent)
		if contentType != "" {
			request.Header.Set("Content-Type", contentType)
		}

		response, err := client.Do(request)
		if err != nil {
			lastErr = err
			if attempt < c.Retries && waitForRetry(ctx, attempt, 0) {
				continue
			}
			return err
		}

		if response.StatusCode >= 200 && response.StatusCode < 300 {
			decodeErr := json.NewDecoder(io.LimitReader(response.Body, 64<<20)).Decode(destination)
			closeErr := response.Body.Close()
			if decodeErr != nil {
				return fmt.Errorf("%s returned invalid JSON: %w", provider, decodeErr)
			}
			return closeErr
		}

		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 32<<10))
		_ = response.Body.Close()
		lastErr = &UpstreamError{Provider: provider, Status: response.StatusCode}
		if attempt < c.Retries &&
			(response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500) {
			retryAfter := parseRetryAfter(response.Header.Get("Retry-After"))
			if waitForRetry(ctx, attempt, retryAfter) {
				continue
			}
		}
		return lastErr
	}

	return lastErr
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 || seconds > 5 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func waitForRetry(ctx context.Context, attempt int, requested time.Duration) bool {
	delay := requested
	if delay == 0 {
		delay = time.Duration(100*(attempt+1)) * time.Millisecond
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
