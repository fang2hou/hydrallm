package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/charmbracelet/log"
)

func TestShouldWait(t *testing.T) {
	cfg := &Config{}
	logger := log.New(io.Discard)
	transport := newRetryTransport(cfg, logger)

	// shouldWait(cycle, modelIdx, attempt, numModels, modelAttempts, maxCycles int)

	tests := []struct {
		name          string
		cycle         int
		modelIdx      int
		attempt       int
		numModels     int
		modelAttempts int
		maxCycles     int
		want          bool
	}{
		{"first attempt", 0, 0, 0, 1, 3, 2, true},
		{"last attempt of final cycle", 1, 0, 2, 1, 3, 2, false},
		{"middle attempt", 0, 0, 1, 1, 3, 2, true},
		{"last model but not last attempt", 0, 1, 0, 2, 2, 1, true},
		{"last attempt but not last model", 0, 0, 1, 2, 2, 1, true},
		{"single attempt single model single cycle", 0, 0, 0, 1, 1, 1, false},
		{"first cycle of multi cycle", 0, 0, 0, 1, 1, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transport.shouldWait(
				tt.cycle,
				tt.modelIdx,
				tt.attempt,
				tt.numModels,
				tt.modelAttempts,
				tt.maxCycles,
			)
			if got != tt.want {
				t.Errorf("shouldWait() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		code int
		want bool
	}{
		{"200 OK is not retryable", 200, false},
		{"201 Created is not retryable", 201, false},
		{"204 No Content is not retryable", 204, false},
		{"301 Moved is not retryable", 301, false},
		{"400 Bad Request is not retryable", 400, false},
		{"401 Unauthorized is not retryable", 401, false},
		{"403 Forbidden is not retryable", 403, false},
		{"404 Not Found is not retryable", 404, false},
		{"408 Request Timeout is not retryable", 408, false},
		{"429 Too Many Requests is retryable", 429, true},
		{"500 Internal Server Error is retryable", 500, true},
		{"501 Not Implemented is retryable", 501, true},
		{"502 Bad Gateway is retryable", 502, true},
		{"503 Service Unavailable is retryable", 503, true},
		{"504 Gateway Timeout is retryable", 504, true},
		{"505 HTTP Version Not Supported is retryable", 505, true},
		{"599 is retryable", 599, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryable(tt.code); got != tt.want {
				t.Errorf("isRetryable(%d) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestBuildTargetURL(t *testing.T) {
	t.Run("strip version prefix", func(t *testing.T) {
		transport := &RetryTransport{}
		parsedURL, _ := url.Parse("https://api.openai.com")
		endpoint := Endpoint{
			ParsedURL:          parsedURL,
			StripVersionPrefix: true,
		}

		originalReq, _ := http.NewRequest("POST", "http://localhost/v1/chat/completions", nil)
		newReq := originalReq.Clone(context.Background())

		transport.buildTargetURL(newReq, originalReq, endpoint)

		if newReq.URL.String() != "https://api.openai.com/chat/completions" {
			t.Errorf("unexpected URL: %s", newReq.URL.String())
		}
		if newReq.Host != "api.openai.com" {
			t.Errorf("unexpected host: %s", newReq.Host)
		}
	})

	t.Run("keep version prefix", func(t *testing.T) {
		transport := &RetryTransport{}
		parsedURL, _ := url.Parse("https://api.openai.com")
		endpoint := Endpoint{
			ParsedURL:          parsedURL,
			StripVersionPrefix: false,
		}

		originalReq, _ := http.NewRequest("POST", "http://localhost/v1/chat/completions", nil)
		newReq := originalReq.Clone(context.Background())

		transport.buildTargetURL(newReq, originalReq, endpoint)

		if newReq.URL.String() != "https://api.openai.com/v1/chat/completions" {
			t.Errorf("unexpected URL: %s", newReq.URL.String())
		}
	})
}

func TestSetAuthHeaders(t *testing.T) {
	transport := &RetryTransport{
		logger: log.New(io.Discard),
	}

	t.Run("openai with key", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		endpoint := Endpoint{APIKey: "sk-123"}
		transport.setAuthHeaders(req, "openai", endpoint)
		if req.Header.Get("Authorization") != "Bearer sk-123" {
			t.Errorf("unexpected Authorization header for openai")
		}
	})

	t.Run("openai skip header", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		req.Header.Set("Authorization", "Bearer something")
		endpoint := Endpoint{APIKey: "-"}
		transport.setAuthHeaders(req, "openai", endpoint)
		if req.Header.Get("Authorization") != "" {
			t.Errorf("expected Authorization header to be deleted")
		}
	})

	t.Run("openai empty key does not modify header", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		endpoint := Endpoint{APIKey: ""}
		transport.setAuthHeaders(req, "openai", endpoint)
		if req.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header for empty key")
		}
	})

	t.Run("anthropic with key", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		endpoint := Endpoint{APIKey: "anthropic-key"}
		transport.setAuthHeaders(req, "anthropic", endpoint)
		if req.Header.Get("x-api-key") != "anthropic-key" {
			t.Errorf("unexpected x-api-key header for anthropic")
		}
		if req.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("unexpected anthropic-version header")
		}
	})

	t.Run("anthropic skip header", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		req.Header.Set("x-api-key", "something")
		endpoint := Endpoint{APIKey: "-"}
		transport.setAuthHeaders(req, "anthropic", endpoint)
		if req.Header.Get("x-api-key") != "" {
			t.Errorf("expected x-api-key header to be deleted")
		}
	})

	t.Run("anthropic empty key does not modify header", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		endpoint := Endpoint{APIKey: ""}
		transport.setAuthHeaders(req, "anthropic", endpoint)
		if req.Header.Get("x-api-key") != "" {
			t.Errorf("expected no x-api-key header for empty key")
		}
		// anthropic-version should still be set
		if req.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header")
		}
	})

	t.Run("bedrock without creds skips signing", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		endpoint := Endpoint{AWSAccessKeyID: ""}
		transport.setAuthHeaders(req, "bedrock", endpoint)
		if req.Header.Get("Authorization") != "" {
			t.Errorf("expected no Authorization header for bedrock without creds")
		}
	})

	t.Run("unknown type defaults to openai behavior", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/", nil)
		endpoint := Endpoint{APIKey: "test-key"}
		transport.setAuthHeaders(req, "unknown-type", endpoint)
		if req.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer token for unknown type")
		}
	})
}

func TestWaitExitsOnContext(t *testing.T) {
	transport := &RetryTransport{
		logger: log.New(io.Discard),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	transport.wait(ctx, 10*time.Second, 1, false)
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("wait took too long: %v, should have returned immediately via context", duration)
	}
}

func TestWaitExponential(t *testing.T) {
	transport := &RetryTransport{
		logger: log.New(io.Discard),
	}

	ctx := context.Background()
	start := time.Now()
	transport.wait(ctx, 10*time.Millisecond, 2, true)
	duration := time.Since(start)

	// With exponential backoff: interval * totalAttempts = 10ms * 2 = 20ms
	// Allow some tolerance for timing variations
	expectedMin := 18 * time.Millisecond
	if duration < expectedMin {
		t.Errorf("wait was too short: %v, expected at least %v", duration, expectedMin)
	}
}

func TestWaitNonExponential(t *testing.T) {
	transport := &RetryTransport{
		logger: log.New(io.Discard),
	}

	ctx := context.Background()
	interval := 10 * time.Millisecond
	start := time.Now()
	transport.wait(ctx, interval, 5, false)
	duration := time.Since(start)

	// Without exponential backoff, should wait exactly interval (totalAttempts is ignored)
	// Allow tolerance for timing variations
	expectedMin := 8 * time.Millisecond
	expectedMax := 50 * time.Millisecond
	if duration < expectedMin {
		t.Errorf("wait was too short: %v, expected at least %v", duration, expectedMin)
	}
	if duration > expectedMax {
		t.Errorf("wait was too long: %v, expected around %v", duration, interval)
	}
}

func TestHandleRetryableResponse(t *testing.T) {
	t.Run("include error body", func(t *testing.T) {
		logOutput := &bytes.Buffer{}
		logger := log.New(logOutput)
		cfg := &Config{Log: LogConfig{IncludeErrorBody: true}}
		transport := &RetryTransport{config: cfg, logger: logger}

		resp := &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(bytes.NewReader([]byte("rate limited error"))),
		}

		transport.handleRetryableResponse(resp, "test-endpoint")
		if !bytes.Contains(logOutput.Bytes(), []byte("rate limited error")) {
			t.Errorf("expected error body in log, got: %s", logOutput.String())
		}
	})

	t.Run("exclude error body", func(t *testing.T) {
		logOutput := &bytes.Buffer{}
		logger := log.New(logOutput)
		cfg := &Config{Log: LogConfig{IncludeErrorBody: false}}
		transport := &RetryTransport{config: cfg, logger: logger}

		resp := &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(bytes.NewReader([]byte("rate limited error 2"))),
		}

		transport.handleRetryableResponse(resp, "test-endpoint")
		if bytes.Contains(logOutput.Bytes(), []byte("rate limited error 2")) {
			t.Errorf("did not expect error body in log")
		}
	})
}

func TestHandleErrorResponse(t *testing.T) {
	t.Run("include error body", func(t *testing.T) {
		logOutput := &bytes.Buffer{}
		logger := log.New(logOutput)
		cfg := &Config{Log: LogConfig{IncludeErrorBody: true}}
		transport := &RetryTransport{config: cfg, logger: logger}

		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewReader([]byte("bad request error"))),
		}
		model := Model{Endpoint: "e1", Model: "gpt-4"}

		transport.handleErrorResponse(resp, model)
		if !bytes.Contains(logOutput.Bytes(), []byte("bad request error")) {
			t.Errorf("expected error body in log, got: %s", logOutput.String())
		}

		// Read body again to ensure it was properly replaced
		bodyBytes, _ := io.ReadAll(resp.Body)
		if string(bodyBytes) != "bad request error" {
			t.Errorf("expected body to be preserved, got: %s", string(bodyBytes))
		}
	})

	t.Run("exclude error body", func(t *testing.T) {
		logOutput := &bytes.Buffer{}
		logger := log.New(logOutput)
		cfg := &Config{Log: LogConfig{IncludeErrorBody: false}}
		transport := &RetryTransport{config: cfg, logger: logger}

		resp := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(bytes.NewReader([]byte("bad request error 2"))),
		}
		model := Model{Endpoint: "e1", Model: "gpt-4"}

		transport.handleErrorResponse(resp, model)
		if bytes.Contains(logOutput.Bytes(), []byte("bad request error 2")) {
			t.Errorf("did not expect error body in log")
		}
	})
}

func TestSignAWSRequest(t *testing.T) {
	t.Run("with valid credentials", func(t *testing.T) {
		transport := &RetryTransport{
			logger: log.New(io.Discard),
		}

		req, _ := http.NewRequestWithContext(
			context.Background(),
			"POST",
			"https://bedrock.us-east-1.amazonaws.com",
			bytes.NewReader([]byte(`{}`)),
		)

		endpoint := Endpoint{
			AWSRegion:          "us-east-1",
			AWSAccessKeyID:     "mock-key",
			AWSSecretAccessKey: "mock-secret",
		}

		transport.signAWSRequest(req, endpoint)

		if req.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header to be set by aws signer")
		}
		if !bytes.Contains([]byte(req.Header.Get("Authorization")), []byte("Credential=mock-key")) {
			t.Errorf(
				"expected mock-key credential in Authorization header, got: %s",
				req.Header.Get("Authorization"),
			)
		}
	})

	t.Run("with session token", func(t *testing.T) {
		transport := &RetryTransport{
			logger: log.New(io.Discard),
		}

		req, _ := http.NewRequestWithContext(
			context.Background(),
			"POST",
			"https://bedrock.us-east-1.amazonaws.com",
			bytes.NewReader([]byte(`{}`)),
		)

		endpoint := Endpoint{
			AWSRegion:          "us-east-1",
			AWSAccessKeyID:     "mock-key",
			AWSSecretAccessKey: "mock-secret",
			AWSSessionToken:    "mock-session-token",
		}

		transport.signAWSRequest(req, endpoint)

		if req.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header to be set by aws signer")
		}
		// Session token should be in SecurityToken header
		if req.Header.Get("X-Amz-Security-Token") != "mock-session-token" {
			t.Errorf(
				"expected X-Amz-Security-Token header, got: %s",
				req.Header.Get("X-Amz-Security-Token"),
			)
		}
	})

	t.Run("uses default region when not specified", func(t *testing.T) {
		transport := &RetryTransport{
			logger: log.New(io.Discard),
		}

		req, _ := http.NewRequestWithContext(
			context.Background(),
			"POST",
			"https://bedrock.amazonaws.com",
			bytes.NewReader([]byte(`{}`)),
		)

		endpoint := Endpoint{
			// AWSRegion is empty - should default to us-east-1
			AWSAccessKeyID:     "mock-key",
			AWSSecretAccessKey: "mock-secret",
		}

		transport.signAWSRequest(req, endpoint)

		if req.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header to be set by aws signer")
		}
		// Should contain us-east-1 in the credential scope
		if !bytes.Contains([]byte(req.Header.Get("Authorization")), []byte("us-east-1")) {
			t.Errorf(
				"expected us-east-1 in Authorization header, got: %s",
				req.Header.Get("Authorization"),
			)
		}
	})

	t.Run("sets Content-Type when not present", func(t *testing.T) {
		transport := &RetryTransport{
			logger: log.New(io.Discard),
		}

		req, _ := http.NewRequestWithContext(
			context.Background(),
			"POST",
			"https://bedrock.us-east-1.amazonaws.com",
			bytes.NewReader([]byte(`{}`)),
		)

		endpoint := Endpoint{
			AWSRegion:          "us-east-1",
			AWSAccessKeyID:     "mock-key",
			AWSSecretAccessKey: "mock-secret",
		}

		transport.signAWSRequest(req, endpoint)

		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf(
				"expected Content-Type application/json, got: %s",
				req.Header.Get("Content-Type"),
			)
		}
	})

	t.Run("preserves existing Content-Type", func(t *testing.T) {
		transport := &RetryTransport{
			logger: log.New(io.Discard),
		}

		req, _ := http.NewRequestWithContext(
			context.Background(),
			"POST",
			"https://bedrock.us-east-1.amazonaws.com",
			bytes.NewReader([]byte(`{}`)),
		)
		req.Header.Set("Content-Type", "application/octet-stream")

		endpoint := Endpoint{
			AWSRegion:          "us-east-1",
			AWSAccessKeyID:     "mock-key",
			AWSSecretAccessKey: "mock-secret",
		}

		transport.signAWSRequest(req, endpoint)

		if req.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf(
				"expected Content-Type to be preserved, got: %s",
				req.Header.Get("Content-Type"),
			)
		}
	})

	t.Run("with nil body", func(t *testing.T) {
		transport := &RetryTransport{
			logger: log.New(io.Discard),
		}

		req, _ := http.NewRequestWithContext(
			context.Background(),
			"POST",
			"https://bedrock.us-east-1.amazonaws.com",
			nil,
		)

		endpoint := Endpoint{
			AWSRegion:          "us-east-1",
			AWSAccessKeyID:     "mock-key",
			AWSSecretAccessKey: "mock-secret",
		}

		transport.signAWSRequest(req, endpoint)

		if req.Header.Get("Authorization") == "" {
			t.Error("expected Authorization header to be set by aws signer")
		}
	})
}

func TestSignAWSRequestNoCredentials(t *testing.T) {
	transport := &RetryTransport{
		logger: log.New(io.Discard),
	}

	req, _ := http.NewRequestWithContext(
		context.Background(),
		"POST",
		"https://bedrock.us-east-1.amazonaws.com",
		bytes.NewReader([]byte(`{}`)),
	)

	// No credentials - should skip signing
	endpoint := Endpoint{}
	transport.signAWSRequest(req, endpoint)

	if req.Header.Get("Authorization") != "" {
		t.Error("expected no Authorization header when no credentials provided")
	}
}

func TestBuildTargetURLWithBasePath(t *testing.T) {
	transport := &RetryTransport{}

	parsedURL, _ := url.Parse("https://api.example.com/base")
	endpoint := Endpoint{
		ParsedURL:          parsedURL,
		StripVersionPrefix: false,
	}

	originalReq, _ := http.NewRequest("POST", "http://localhost/v1/chat", nil)
	newReq := originalReq.Clone(context.Background())

	transport.buildTargetURL(newReq, originalReq, endpoint)

	expected := "https://api.example.com/base/v1/chat"
	if newReq.URL.String() != expected {
		t.Errorf("unexpected URL: %s, want %s", newReq.URL.String(), expected)
	}
}

func TestBuildTargetURLTrailingSlash(t *testing.T) {
	transport := &RetryTransport{}

	tests := []struct {
		name         string
		endpointURL  string
		requestPath  string
		stripVersion bool
		expected     string
	}{
		{
			name:         "URL with trailing slash and request with leading slash",
			endpointURL:  "https://api.example.com/v1/",
			requestPath:  "/chat/completions",
			stripVersion: false,
			expected:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:         "URL with trailing slash, strip version",
			endpointURL:  "https://api.example.com/v1/",
			requestPath:  "/v1/chat/completions",
			stripVersion: true,
			expected:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:         "URL without trailing slash",
			endpointURL:  "https://api.example.com/v1",
			requestPath:  "/chat/completions",
			stripVersion: false,
			expected:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:         "URL with double trailing slash",
			endpointURL:  "https://api.example.com/v1//",
			requestPath:  "/chat/completions",
			stripVersion: false,
			expected:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:         "Empty path with trailing slash",
			endpointURL:  "https://api.example.com/",
			requestPath:  "/v1/chat/completions",
			stripVersion: false,
			expected:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:         "Base path with trailing slash",
			endpointURL:  "https://api.example.com/base/path/",
			requestPath:  "/v1/chat",
			stripVersion: false,
			expected:     "https://api.example.com/base/path/v1/chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedURL, _ := url.Parse(tt.endpointURL)
			endpoint := Endpoint{
				ParsedURL:          parsedURL,
				StripVersionPrefix: tt.stripVersion,
			}

			originalReq, _ := http.NewRequest("POST", "http://localhost"+tt.requestPath, nil)
			newReq := originalReq.Clone(context.Background())

			transport.buildTargetURL(newReq, originalReq, endpoint)

			if newReq.URL.String() != tt.expected {
				t.Errorf("unexpected URL: %s, want %s", newReq.URL.String(), tt.expected)
			}
		})
	}
}

func TestTryModelEndpointNotFound(t *testing.T) {
	cfg := &Config{
		Endpoints: map[string]Endpoint{
			"existing": {URL: "http://localhost"},
		},
		Models: []Model{
			{Endpoint: "nonexistent", Model: "test", Type: "openai"},
		},
	}
	_ = cfg.validate()

	transport := newRetryTransport(cfg, log.New(io.Discard))

	originalReq, _ := http.NewRequest(
		"POST",
		"http://localhost/test",
		bytes.NewReader([]byte(`{}`)),
	)
	model := Model{Endpoint: "nonexistent", Model: "test", Type: "openai"}

	_, err := transport.tryModel(
		context.Background(),
		originalReq,
		[]byte(`{}`),
		model,
		false,
		false,
	)
	if err == nil {
		t.Error("expected error for nonexistent endpoint")
	}
}
