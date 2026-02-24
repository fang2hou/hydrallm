package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

func TestNewProxy(t *testing.T) {
	cfg := &Config{
		Retry: RetryConfig{
			MaxCycles: 1,
		},
		Endpoints: map[string]Endpoint{
			"mock": {URL: "http://example.com"},
		},
		Models: []Model{
			{Endpoint: "mock", Model: "test-model", Type: "openai", Attempts: 1},
		},
	}
	logger := log.New(io.Discard)
	proxy := newProxy(cfg, logger)

	if proxy == nil {
		t.Fatal("expected proxy, got nil")
	}
}

func TestNewProxy_Transport(t *testing.T) {
	cfg := &Config{
		Retry: RetryConfig{MaxCycles: 1},
		Endpoints: map[string]Endpoint{
			"mock": {URL: "http://example.com"},
		},
		Models: []Model{
			{Endpoint: "mock", Model: "test-model", Type: "openai", Attempts: 1},
		},
	}
	proxy := newProxy(cfg, log.New(io.Discard))

	if proxy.Transport == nil {
		t.Fatal("expected transport, got nil")
	}
}

func TestNewProxy_FlushInterval(t *testing.T) {
	cfg := &Config{
		Retry: RetryConfig{MaxCycles: 1},
		Endpoints: map[string]Endpoint{
			"mock": {URL: "http://example.com"},
		},
		Models: []Model{
			{Endpoint: "mock", Model: "test-model", Type: "openai", Attempts: 1},
		},
	}
	proxy := newProxy(cfg, log.New(io.Discard))

	if proxy.FlushInterval != -1 {
		t.Errorf("expected FlushInterval -1, got %d", proxy.FlushInterval)
	}
}

func TestNewProxy_ErrorHandler(t *testing.T) {
	cfg := &Config{
		Retry: RetryConfig{MaxCycles: 1},
		Endpoints: map[string]Endpoint{
			"mock": {URL: "http://example.com"},
		},
		Models: []Model{
			{Endpoint: "mock", Model: "test-model", Type: "openai", Attempts: 1},
		},
	}
	proxy := newProxy(cfg, log.New(io.Discard))

	if proxy.ErrorHandler == nil {
		t.Fatal("expected error handler, got nil")
	}

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	proxy.ErrorHandler(recorder, req, http.ErrServerClosed)

	if recorder.Code != http.StatusBadGateway {
		t.Errorf("expected StatusBadGateway, got %d", recorder.Code)
	}
}

// Ensure the transport allows HTTP/2 for server endpoints
func TestRetryTransport_HTTP2(t *testing.T) {
	trans := newRetryTransport(&Config{}, log.New(io.Discard))
	baseTrans, ok := trans.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected underlying transport to be *http.Transport")
	}
	if !baseTrans.ForceAttemptHTTP2 {
		t.Errorf("expected ForceAttemptHTTP2 to be true")
	}
	if baseTrans.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("expected TLSHandshakeTimeout to be 10s")
	}
}
