package main

import (
	"testing"
	"time"
)

func TestResolveEnvOrValue(t *testing.T) {
	t.Setenv("TEST_HYDRALLM_VAR", "secret-value")
	t.Setenv("TEST_HYDRALLM_EMPTY", "")

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"normal value", "my-key", "my-key"},
		{"env var resolved", "$TEST_HYDRALLM_VAR", "secret-value"},
		{"undefined env var returns original", "$NO_SUCH_VAR_12345", "$NO_SUCH_VAR_12345"},
		{"just dollar sign", "$", "$"},
		{"empty env var returns original", "$TEST_HYDRALLM_EMPTY", "$TEST_HYDRALLM_EMPTY"},
		{"value with spaces", "my key with spaces", "my key with spaces"},
		{"value with special chars", "key-123_abc!@#", "key-123_abc!@#"},
		{"env var name with underscore", "$TEST_HYDRALLM_VAR", "secret-value"},
		{"dollar in middle of string", "prefix$TEST_HYDRALLM_VAR", "prefix$TEST_HYDRALLM_VAR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveEnvOrValue(tt.input); got != tt.want {
				t.Errorf("resolveEnvOrValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		check    func(*Config) bool
		expected interface{}
	}{
		{
			"host defaults to 127.0.0.1",
			func(c *Config) bool { return c.Server.Host == "127.0.0.1" },
			"127.0.0.1",
		},
		{"port defaults to 8080", func(c *Config) bool { return c.Server.Port == 8080 }, 8080},
		{
			"read timeout defaults to 1 minute",
			func(c *Config) bool { return c.Server.ReadTimeout == time.Minute },
			time.Minute,
		},
		{
			"write timeout defaults to 10 minutes",
			func(c *Config) bool { return c.Server.WriteTimeout == 10*time.Minute },
			10 * time.Minute,
		},
		{
			"log level defaults to info",
			func(c *Config) bool { return c.Log.Level == "info" },
			"info",
		},
		{"max cycles defaults to 10", func(c *Config) bool { return c.Retry.MaxCycles == 10 }, 10},
		{
			"default timeout defaults to 30s",
			func(c *Config) bool { return c.Retry.DefaultTimeout == 30*time.Second },
			30 * time.Second,
		},
		{
			"default interval defaults to 100ms",
			func(c *Config) bool { return c.Retry.DefaultInterval == 100*time.Millisecond },
			100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			applyDefaults(cfg)
			if !tt.check(cfg) {
				t.Errorf("expected %v", tt.expected)
			}
		})
	}
}

func TestApplyDefaults_DoesNotOverride(t *testing.T) {
	t.Run("does not override preset host", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Host: "0.0.0.0"}}
		applyDefaults(cfg)
		if cfg.Server.Host != "0.0.0.0" {
			t.Errorf("expected host to remain 0.0.0.0, got %s", cfg.Server.Host)
		}
	})

	t.Run("does not override preset port", func(t *testing.T) {
		cfg := &Config{Server: ServerConfig{Port: 3000}}
		applyDefaults(cfg)
		if cfg.Server.Port != 3000 {
			t.Errorf("expected port to remain 3000, got %d", cfg.Server.Port)
		}
	})

	t.Run("does not override preset log level", func(t *testing.T) {
		cfg := &Config{Log: LogConfig{Level: "debug"}}
		applyDefaults(cfg)
		if cfg.Log.Level != "debug" {
			t.Errorf("expected log level to remain debug, got %s", cfg.Log.Level)
		}
	})

	t.Run("does not override preset max cycles", func(t *testing.T) {
		cfg := &Config{Retry: RetryConfig{MaxCycles: 5}}
		applyDefaults(cfg)
		if cfg.Retry.MaxCycles != 5 {
			t.Errorf("expected max cycles to remain 5, got %d", cfg.Retry.MaxCycles)
		}
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("no models", func(t *testing.T) {
		cfg := &Config{}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for no models")
		}
	})

	t.Run("valid basic config", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "openai"},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("endpoint not found", func(t *testing.T) {
		cfg := &Config{
			Models: []Model{
				{Endpoint: "nonexistent", Model: "gpt-4", Type: "openai"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for endpoint not found")
		}
	})

	t.Run("empty endpoint name", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "", Model: "gpt-4", Type: "openai"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for empty endpoint name")
		}
	})

	t.Run("missing model field", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "e1", Type: "openai"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for missing model field")
		}
	})

	t.Run("missing type field", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for missing type field")
		}
	})

	t.Run("mixed model types are rejected", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"openai":    {URL: "http://localhost:8001"},
				"anthropic": {URL: "http://localhost:8002"},
			},
			Models: []Model{
				{Endpoint: "openai", Model: "gpt-4", Type: "openai"},
				{Endpoint: "anthropic", Model: "claude-3", Type: "anthropic"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for mixed model types")
		}
	})

	t.Run("unsupported model type is rejected", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "azure"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for unsupported model type")
		}
	})

	t.Run("invalid endpoint URL", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "://invalid-url"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "openai"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for invalid URL")
		}
	})
}

func TestValidateConfig_Defaults(t *testing.T) {
	t.Run("attempts defaults to 1 when negative", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "openai", Attempts: -5},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Models[0].Attempts != 1 {
			t.Errorf("expected attempts to default to 1, got %d", cfg.Models[0].Attempts)
		}
	})

	t.Run("attempts defaults to 1 when zero", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "openai", Attempts: 0},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Models[0].Attempts != 1 {
			t.Errorf("expected attempts to default to 1, got %d", cfg.Models[0].Attempts)
		}
	})

	t.Run("timeout defaults to retry default timeout", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "http://localhost"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "openai"},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Models[0].Timeout != time.Second {
			t.Errorf("expected timeout to default to 1s, got %v", cfg.Models[0].Timeout)
		}
	})

	t.Run("URL trailing slash is normalized", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "https://api.example.com/v1/"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "openai"},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Endpoints["e1"].ParsedURL.Path != "/v1" {
			t.Errorf("expected path to be '/v1', got '%s'", cfg.Endpoints["e1"].ParsedURL.Path)
		}
	})

	t.Run("URL double trailing slashes are normalized", func(t *testing.T) {
		cfg := &Config{
			Endpoints: map[string]Endpoint{
				"e1": {URL: "https://api.example.com/v1//"},
			},
			Models: []Model{
				{Endpoint: "e1", Model: "gpt-4", Type: "openai"},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Endpoints["e1"].ParsedURL.Path != "/v1" {
			t.Errorf("expected path to be '/v1', got '%s'", cfg.Endpoints["e1"].ParsedURL.Path)
		}
	})
}

func TestValidateBedrockCredentials(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  Endpoint
		wantError bool
	}{
		// Valid configurations
		{"no credentials is valid (skip signing)", Endpoint{}, false},
		{
			"long-term credentials (access + secret)",
			Endpoint{AWSAccessKeyID: "A", AWSSecretAccessKey: "B"},
			false,
		},
		{
			"long-term credentials with region",
			Endpoint{AWSRegion: "us-east-1", AWSAccessKeyID: "A", AWSSecretAccessKey: "B"},
			false,
		},
		{
			"temporary credentials (with session token)",
			Endpoint{AWSAccessKeyID: "A", AWSSecretAccessKey: "B", AWSSessionToken: "C"},
			false,
		},
		{
			"temporary credentials with region",
			Endpoint{
				AWSRegion:          "us-east-1",
				AWSAccessKeyID:     "A",
				AWSSecretAccessKey: "B",
				AWSSessionToken:    "C",
			},
			false,
		},
		{"only region is valid (no signing, use env)", Endpoint{AWSRegion: "us-east-1"}, false},

		// Invalid configurations
		{"only access key is invalid", Endpoint{AWSAccessKeyID: "key"}, true},
		{"only secret key is invalid", Endpoint{AWSSecretAccessKey: "secret"}, true},
		{"only session token is invalid", Endpoint{AWSSessionToken: "token"}, true},
		{
			"region + access key (missing secret) is invalid",
			Endpoint{AWSRegion: "us-east-1", AWSAccessKeyID: "key"},
			true,
		},
		{
			"region + secret key (missing access) is invalid",
			Endpoint{AWSRegion: "us-east-1", AWSSecretAccessKey: "secret"},
			true,
		},
		{
			"session token without access key is invalid",
			Endpoint{AWSSessionToken: "token", AWSSecretAccessKey: "secret"},
			true,
		},
		{
			"session token without secret key is invalid",
			Endpoint{AWSSessionToken: "token", AWSAccessKeyID: "key"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBedrockCredentials("test", tt.endpoint)
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEndpointMethods(t *testing.T) {
	t.Setenv("URL_VAR", "http://example.com")
	t.Setenv("REGION", "us-east-2")
	t.Setenv("EMPTY_VAR", "")

	tests := []struct {
		name     string
		endpoint Endpoint
		method   func(Endpoint) string
		expected string
	}{
		{
			"GetURL with env var",
			Endpoint{URL: "$URL_VAR"},
			func(e Endpoint) string { return e.GetURL() },
			"http://example.com",
		},
		{
			"GetURL with undefined env var returns original",
			Endpoint{URL: "$UNDEFINED_VAR_XYZ"},
			func(e Endpoint) string { return e.GetURL() },
			"$UNDEFINED_VAR_XYZ",
		},
		{
			"GetURL with empty value returns original",
			Endpoint{URL: "$EMPTY_VAR"},
			func(e Endpoint) string { return e.GetURL() },
			"$EMPTY_VAR",
		},
		{
			"GetURL static value",
			Endpoint{URL: "http://static.com"},
			func(e Endpoint) string { return e.GetURL() },
			"http://static.com",
		},
		{
			"GetAPIKey with env var",
			Endpoint{APIKey: "$URL_VAR"},
			func(e Endpoint) string { return e.GetAPIKey() },
			"http://example.com",
		},
		{
			"GetAPIKey static value",
			Endpoint{APIKey: "sk-12345"},
			func(e Endpoint) string { return e.GetAPIKey() },
			"sk-12345",
		},
		{
			"GetAPIKey empty",
			Endpoint{APIKey: ""},
			func(e Endpoint) string { return e.GetAPIKey() },
			"",
		},
		{
			"GetAWSRegion with env var",
			Endpoint{AWSRegion: "$REGION"},
			func(e Endpoint) string { return e.GetAWSRegion() },
			"us-east-2",
		},
		{
			"GetAWSRegion static",
			Endpoint{AWSRegion: "us-west-2"},
			func(e Endpoint) string { return e.GetAWSRegion() },
			"us-west-2",
		},
		{
			"GetAWSAccessKeyID static",
			Endpoint{AWSAccessKeyID: "AKIAIOSFODNN7EXAMPLE"},
			func(e Endpoint) string { return e.GetAWSAccessKeyID() },
			"AKIAIOSFODNN7EXAMPLE",
		},
		{
			"GetAWSSecretAccessKey static",
			Endpoint{AWSSecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"},
			func(e Endpoint) string { return e.GetAWSSecretAccessKey() },
			"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		{
			"GetAWSSessionToken static",
			Endpoint{AWSSessionToken: "token123"},
			func(e Endpoint) string { return e.GetAWSSessionToken() },
			"token123",
		},
		{
			"empty endpoint returns empty values",
			Endpoint{},
			func(e Endpoint) string { return e.GetURL() },
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.method(tt.endpoint); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetInterval(t *testing.T) {
	tests := []struct {
		name            string
		modelInterval   time.Duration
		endpoint        Endpoint
		defaultInterval time.Duration
		expected        time.Duration
	}{
		{
			"model interval takes priority",
			1 * time.Second,
			Endpoint{Interval: 2 * time.Second},
			5 * time.Second,
			1 * time.Second,
		},
		{
			"endpoint interval when model is zero",
			0,
			Endpoint{Interval: 2 * time.Second},
			5 * time.Second,
			2 * time.Second,
		},
		{
			"default interval when both are zero",
			0,
			Endpoint{Interval: 0},
			5 * time.Second,
			5 * time.Second,
		},
		{
			"negative model interval uses endpoint",
			-1 * time.Second,
			Endpoint{Interval: 2 * time.Second},
			5 * time.Second,
			2 * time.Second,
		},
		{
			"negative endpoint interval uses default",
			0,
			Endpoint{Interval: -1 * time.Second},
			5 * time.Second,
			5 * time.Second,
		},
		{"zero default interval returns zero", 0, Endpoint{Interval: 0}, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{Interval: tt.modelInterval}
			got := m.GetInterval(tt.endpoint, tt.defaultInterval)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}
