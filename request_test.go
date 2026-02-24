package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSetModel(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		model    string
		expected string
		hasErr   bool
	}{
		{
			name:     "replace existing model",
			body:     []byte(`{"model": "gpt-3.5-turbo", "temperature": 0.7}`),
			model:    "gpt-4",
			expected: `{"model":"gpt-4","temperature":0.7}`,
			hasErr:   false,
		},
		{
			name:     "add model if not exists",
			body:     []byte(`{"temperature": 0.7}`),
			model:    "gpt-4",
			expected: `{"temperature":0.7,"model":"gpt-4"}`,
			hasErr:   false,
		},
		{
			name:     "empty body adds model",
			body:     []byte(`{}`),
			model:    "gpt-4",
			expected: `{"model":"gpt-4"}`,
			hasErr:   false,
		},
		{
			name:     "nil body creates object with model",
			body:     nil,
			model:    "gpt-4",
			expected: `{"model":"gpt-4"}`,
			hasErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setModel(tt.body, tt.model)
			if (err != nil) != tt.hasErr {
				t.Fatalf("setModel() error = %v, wantErr %v", err, tt.hasErr)
			}
			sGot := strings.ReplaceAll(string(got), " ", "")
			sExp := strings.ReplaceAll(tt.expected, " ", "")
			if sGot != sExp {
				t.Errorf("setModel() = %s, want %s", sGot, sExp)
			}
		})
	}
}

func TestIsStreamingRequest(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		accept string
		body   string
		want   bool
	}{
		{
			name: "streaming path segment /stream",
			path: "/v1/chat/completions/stream",
			want: true,
		},
		{
			name: "-stream in path",
			path: "/v1/chat-stream",
			want: true,
		},
		{
			name:   "SSE Accept header",
			path:   "/v1/chat",
			accept: "text/event-stream",
			want:   true,
		},
		{
			name:   "SSE Accept header with charset",
			path:   "/v1/chat",
			accept: "text/event-stream; charset=utf-8",
			want:   true,
		},
		{
			name: "stream true in body",
			path: "/v1/chat",
			body: `{"stream": true}`,
			want: true,
		},
		{
			name: "stream false in body",
			path: "/v1/chat",
			body: `{"stream": false}`,
			want: false,
		},
		{
			name: "not streaming",
			path: "/v1/chat",
			body: `{"model": "abc"}`,
			want: false,
		},
		{
			name: "empty body",
			path: "/v1/chat",
			body: "",
			want: false,
		},
		{
			name: "invalid JSON body",
			path: "/v1/chat",
			body: "not json",
			want: false,
		},
		{
			name: "nil body",
			path: "/v1/chat",
			body: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				URL:    &url.URL{Path: tt.path},
				Header: http.Header{},
			}
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			var body []byte
			if tt.body != "" {
				body = []byte(tt.body)
			}
			got := isStreamingRequest(req, body)
			if got != tt.want {
				t.Errorf("isStreamingRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadErrorBody(t *testing.T) {
	t.Run("uncompressed", func(t *testing.T) {
		resp := &http.Response{
			Body:   io.NopCloser(strings.NewReader("plain text error")),
			Header: http.Header{},
		}
		got, err := readErrorBody(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "plain text error" {
			t.Errorf("got %q, want %q", string(got), "plain text error")
		}
	})

	t.Run("gzipped", func(t *testing.T) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, _ = gw.Write([]byte("gzipped error"))
		_ = gw.Close()

		resp := &http.Response{
			Body: io.NopCloser(&buf),
			Header: http.Header{
				"Content-Encoding": []string{"gzip"},
			},
		}

		got, err := readErrorBody(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "gzipped error" {
			t.Errorf("got %q, want %q", string(got), "gzipped error")
		}
	})

	t.Run("invalid gzip data", func(t *testing.T) {
		resp := &http.Response{
			Body: io.NopCloser(strings.NewReader("not valid gzip data")),
			Header: http.Header{
				"Content-Encoding": []string{"gzip"},
			},
		}

		_, err := readErrorBody(resp)
		if err == nil {
			t.Error("expected error for invalid gzip data")
		}
	})

	t.Run("large body truncated to 4KB", func(t *testing.T) {
		largeBody := strings.Repeat("x", 10*1024) // 10KB
		resp := &http.Response{
			Body:   io.NopCloser(strings.NewReader(largeBody)),
			Header: http.Header{},
		}
		got, err := readErrorBody(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) > 4*1024 {
			t.Errorf("body should be truncated to 4KB, got %d bytes", len(got))
		}
	})

	t.Run("empty body", func(t *testing.T) {
		resp := &http.Response{
			Body:   io.NopCloser(strings.NewReader("")),
			Header: http.Header{},
		}
		got, err := readErrorBody(resp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(got) != "" {
			t.Errorf("got %q, want empty string", string(got))
		}
	})
}

func TestFormatBodyForLog(t *testing.T) {
	t.Run("empty body", func(t *testing.T) {
		got := formatBodyForLog([]byte{})
		if got != "(empty)" {
			t.Errorf("got %q, want '(empty)'", got)
		}
	})

	t.Run("nil body", func(t *testing.T) {
		got := formatBodyForLog(nil)
		if got != "(empty)" {
			t.Errorf("got %q, want '(empty)'", got)
		}
	})

	t.Run("valid JSON gets indented", func(t *testing.T) {
		body := []byte(`{"a":1}`)
		got := formatBodyForLog(body)
		if !strings.Contains(got, "{\n  \"a\": 1\n}") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("valid nested JSON gets indented", func(t *testing.T) {
		body := []byte(`{"outer":{"inner":"value"}}`)
		got := formatBodyForLog(body)
		if !strings.Contains(got, "\"outer\"") || !strings.Contains(got, "\"inner\"") {
			t.Errorf("got %q", got)
		}
	})

	t.Run("invalid JSON returned as is", func(t *testing.T) {
		body := []byte(`not json`)
		got := formatBodyForLog(body)
		if got != "not json" {
			t.Errorf("got %q", got)
		}
	})

	t.Run("large JSON payload truncated", func(t *testing.T) {
		// Create a JSON that exceeds the limit when formatted
		large := fmt.Sprintf(`{"data":"%s"}`, strings.Repeat("x", 3000))
		got := formatBodyForLog([]byte(large))
		if !strings.HasSuffix(got, "\n... (truncated)") {
			t.Errorf("expected truncation, got len %d", len(got))
		}
	})

	t.Run("large non-JSON payload truncated", func(t *testing.T) {
		large := strings.Repeat("a", 2500)
		got := formatBodyForLog([]byte(large))
		if !strings.HasSuffix(got, "\n... (truncated)") {
			t.Errorf("expected truncation, got len %d", len(got))
		}
		if len(got) > 2100 {
			t.Errorf("expected size around 2048, got %d", len(got))
		}
	})

	t.Run("exact limit size not truncated", func(t *testing.T) {
		// Create a JSON that's exactly at the limit
		body := []byte(`{"small":"value"}`)
		got := formatBodyForLog(body)
		if strings.Contains(got, "(truncated)") {
			t.Errorf("should not be truncated for small payload")
		}
	})
}
