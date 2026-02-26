package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestTransport_RoundTrip_Success(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 2,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		"mock": {URL: ts.URL, ParsedURL: mustParseURL(ts.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	req, _ := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"http://original/path",
		bytes.NewReader([]byte(`{"test":1}`)),
	)

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&requestCount) != 1 {
		t.Errorf("expected 1 request, got %d", requestCount)
	}
}

func TestTransport_RoundTrip_Retry(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer ts.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 2,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		"mock": {URL: ts.URL, ParsedURL: mustParseURL(ts.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	req, _ := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"http://original/path",
		bytes.NewReader([]byte(`{"test":1}`)),
	)

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 Too Many Requests, got %d", resp.StatusCode)
	}

	// Since 429 is retryable and MaxCycles=1, Attempts=2, it should attempt 2 times
	if atomic.LoadInt32(&requestCount) != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}
}

func TestTransport_RoundTrip_MultiCycle(t *testing.T) {
	var requestCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 2,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		"mock": {URL: ts.URL, ParsedURL: mustParseURL(ts.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       2,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	req, _ := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"http://original/path",
		bytes.NewReader([]byte(`{}`)),
	)

	_, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// MaxCycles=2, Attempts=2, so total attempts = 2 * 2 = 4
	if atomic.LoadInt32(&requestCount) != 4 {
		t.Errorf("expected 4 requests, got %d", requestCount)
	}
}

func TestTransport_RoundTrip_Fallback(t *testing.T) {
	var requestCount1 int32
	var requestCount2 int32

	ts1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount1, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts1.Close()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount2, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts2.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock1",
			Model:    "test-model-1",
			Type:     "openai",
			Attempts: 1,
			Timeout:  time.Second,
		},
		{
			ID:       "m2",
			Provider: "mock2",
			Model:    "test-model-2",
			Type:     "openai",
			Attempts: 1,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		"mock1": {URL: ts1.URL, ParsedURL: mustParseURL(ts1.URL)},
		"mock2": {URL: ts2.URL, ParsedURL: mustParseURL(ts2.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://original/path", nil)

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&requestCount1) != 1 {
		t.Errorf("expected 1 request to ts1, got %d", requestCount1)
	}
	if atomic.LoadInt32(&requestCount2) != 1 {
		t.Errorf("expected 1 request to ts2, got %d", requestCount2)
	}
}

func TestTransport_RoundTrip_Cancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer ts.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 2,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		"mock": {URL: ts.URL, ParsedURL: mustParseURL(ts.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://original/path", nil)

	// Cancel after a short delay to simulate real cancellation scenario
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := transport.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected error due to context cancellation")
	}
}

func TestTransport_RoundTrip_Timeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 1,
			Timeout:  50 * time.Millisecond,
		},
	}
	providers := map[string]Provider{
		"mock": {URL: ts.URL, ParsedURL: mustParseURL(ts.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://original/path", nil)

	start := time.Now()
	_, err := transport.RoundTrip(req)
	duration := time.Since(start)

	if err == nil {
		t.Fatalf("expected error due to timeout")
	}
	if duration > 100*time.Millisecond {
		t.Errorf("request took too long: %v, expected timeout around 50ms", duration)
	}
}

func TestTransport_RoundTrip_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 1,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		"mock": {URL: ts.URL, ParsedURL: mustParseURL(ts.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	// Request with nil body
	req, _ := http.NewRequestWithContext(context.Background(), "POST", "http://original/path", nil)

	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestTransport_RoundTrip_AllAttemptsExhausted(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 2,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		"mock": {URL: ts.URL, ParsedURL: mustParseURL(ts.URL)},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	req, _ := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"http://original/path",
		bytes.NewReader([]byte(`{}`)),
	)

	// 500 is retryable, so all attempts will be exhausted and return the last 500 response
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", resp.StatusCode)
	}
}

func TestTransport_RoundTrip_ConnectionError(t *testing.T) {
	models := []Model{
		{
			ID:       "m1",
			Provider: "mock",
			Model:    "test-model",
			Type:     "openai",
			Attempts: 1,
			Timeout:  time.Second,
		},
	}
	providers := map[string]Provider{
		// Invalid port that won't connect
		"mock": {URL: "http://127.0.0.1:1", ParsedURL: mustParseURL("http://127.0.0.1:1")},
	}
	retry := RetryConfig{
		MaxCycles:       1,
		DefaultInterval: time.Millisecond,
		DefaultTimeout:  time.Second,
	}

	transport := newRetryTransport(models, providers, retry, LogConfig{}, log.New(io.Discard))

	req, _ := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"http://original/path",
		bytes.NewReader([]byte(`{}`)),
	)

	_, err := transport.RoundTrip(req)
	if err == nil {
		t.Fatalf("expected connection error")
	}
}
