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
		setup    func(*Config)
		check    func(*Config) bool
		expected interface{}
	}{
		{
			"log level defaults to info",
			func(c *Config) {},
			func(c *Config) bool { return c.Log.Level == "info" },
			"info",
		},
		{
			"max cycles defaults to 10",
			func(c *Config) {},
			func(c *Config) bool { return c.Retry.MaxCycles == 10 },
			10,
		},
		{
			"default timeout defaults to 30s",
			func(c *Config) {},
			func(c *Config) bool { return c.Retry.DefaultTimeout == 30*time.Second },
			30 * time.Second,
		},
		{
			"default interval defaults to 100ms",
			func(c *Config) {},
			func(c *Config) bool { return c.Retry.DefaultInterval == 100*time.Millisecond },
			100 * time.Millisecond,
		},
		{
			"listener host defaults to 127.0.0.1",
			func(c *Config) { c.Listeners = []Listener{{}} },
			func(c *Config) bool { return c.Listeners[0].Host == "127.0.0.1" },
			"127.0.0.1",
		},
		{
			"listener read timeout defaults to 1 minute",
			func(c *Config) { c.Listeners = []Listener{{}} },
			func(c *Config) bool { return c.Listeners[0].ReadTimeout == time.Minute },
			time.Minute,
		},
		{
			"listener write timeout defaults to 10 minutes",
			func(c *Config) { c.Listeners = []Listener{{}} },
			func(c *Config) bool { return c.Listeners[0].WriteTimeout == 10*time.Minute },
			10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			tt.setup(cfg)
			applyDefaults(cfg)
			if !tt.check(cfg) {
				t.Errorf("expected %v", tt.expected)
			}
		})
	}
}

func TestApplyDefaults_DoesNotOverride(t *testing.T) {
	t.Run("does not override preset listener host", func(t *testing.T) {
		cfg := &Config{Listeners: []Listener{{Host: "0.0.0.0"}}}
		applyDefaults(cfg)
		if cfg.Listeners[0].Host != "0.0.0.0" {
			t.Errorf("expected host to remain 0.0.0.0, got %s", cfg.Listeners[0].Host)
		}
	})

	t.Run("does not override preset listener port", func(t *testing.T) {
		cfg := &Config{Listeners: []Listener{{Port: 3000}}}
		applyDefaults(cfg)
		if cfg.Listeners[0].Port != 3000 {
			t.Errorf("expected port to remain 3000, got %d", cfg.Listeners[0].Port)
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
	t.Run("no providers", func(t *testing.T) {
		cfg := &Config{}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for no providers")
		}
	})

	t.Run("no models", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for no models")
		}
	})

	t.Run("no listeners", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for no listeners")
		}
	})

	t.Run("valid basic config", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("provider not found", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "nonexistent", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for provider not found")
		}
	})

	t.Run("empty provider name", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for empty provider name")
		}
	})

	t.Run("missing model field", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for missing model field")
		}
	})

	t.Run("missing type field", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for missing type field")
		}
	})

	t.Run("listener model not found", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"nonexistent"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for listener model not found")
		}
	})

	t.Run("listener missing name", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for listener missing name")
		}
	})

	t.Run("listener missing port", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for listener missing port")
		}
	})

	t.Run("listener port out of range", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 70000, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for listener port out of range")
		}
	})

	t.Run("duplicate listener names are rejected", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "same", Host: "127.0.0.1", Port: 8080, Models: []string{"m1"}},
				{Name: "same", Host: "127.0.0.1", Port: 8081, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for duplicate listener names")
		}
	})

	t.Run("duplicate listener addresses are rejected", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Host: "127.0.0.1", Port: 8080, Models: []string{"m1"}},
				{Name: "l2", Host: "127.0.0.1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for duplicate listener addresses")
		}
	})

	t.Run("listener empty models", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for listener empty models")
		}
	})

	t.Run("mixed model types in listener are rejected", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"openai":    {URL: "http://localhost:8001"},
				"anthropic": {URL: "http://localhost:8002"},
			},
			Models: map[string]Model{
				"m1": {Provider: "openai", Model: "gpt-4", Type: "openai"},
				"m2": {Provider: "anthropic", Model: "claude-3", Type: "anthropic"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1", "m2"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for mixed model types in listener")
		}
	})

	t.Run("different listeners can have different types", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"openai":    {URL: "http://localhost:8001"},
				"anthropic": {URL: "http://localhost:8002"},
			},
			Models: map[string]Model{
				"m1": {Provider: "openai", Model: "gpt-4", Type: "openai"},
				"m2": {Provider: "anthropic", Model: "claude-3", Type: "anthropic"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
				{Name: "l2", Port: 8081, Models: []string{"m2"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if cfg.Listeners[0].ConfigType != "openai" {
			t.Errorf("expected listener 0 type openai, got %s", cfg.Listeners[0].ConfigType)
		}
		if cfg.Listeners[1].ConfigType != "anthropic" {
			t.Errorf("expected listener 1 type anthropic, got %s", cfg.Listeners[1].ConfigType)
		}
	})

	t.Run("unsupported model type is rejected", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "azure"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for unsupported model type")
		}
	})

	t.Run("invalid provider URL", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "://invalid-url"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for invalid URL")
		}
	})

	t.Run("provider URL missing scheme is rejected", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "api.example.com/v1"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for provider URL missing scheme")
		}
	})

	t.Run("provider URL unsupported scheme is rejected", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "ftp://api.example.com/v1"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for provider URL unsupported scheme")
		}
	})

	t.Run("provider URL missing host is rejected", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "https:///v1"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
		}
		if err := cfg.validate(); err == nil {
			t.Error("expected error for provider URL missing host")
		}
	})
}

func TestValidateConfig_Defaults(t *testing.T) {
	t.Run("attempts defaults to 1 when negative", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai", Attempts: -5},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Models["m1"].Attempts != 1 {
			t.Errorf("expected attempts to default to 1, got %d", cfg.Models["m1"].Attempts)
		}
	})

	t.Run("attempts defaults to 1 when zero", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai", Attempts: 0},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Models["m1"].Attempts != 1 {
			t.Errorf("expected attempts to default to 1, got %d", cfg.Models["m1"].Attempts)
		}
	})

	t.Run("timeout defaults to retry default timeout", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Models["m1"].Timeout != time.Second {
			t.Errorf("expected timeout to default to 1s, got %v", cfg.Models["m1"].Timeout)
		}
	})

	t.Run("URL trailing slash is normalized", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "https://api.example.com/v1/"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Providers["p1"].ParsedURL.Path != "/v1" {
			t.Errorf("expected path to be '/v1', got '%s'", cfg.Providers["p1"].ParsedURL.Path)
		}
	})

	t.Run("URL double trailing slashes are normalized", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "https://api.example.com/v1//"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.Providers["p1"].ParsedURL.Path != "/v1" {
			t.Errorf("expected path to be '/v1', got '%s'", cfg.Providers["p1"].ParsedURL.Path)
		}
	})

	t.Run("resolved models are populated", func(t *testing.T) {
		cfg := &Config{
			Providers: map[string]Provider{
				"p1": {URL: "http://localhost"},
			},
			Models: map[string]Model{
				"m1": {Provider: "p1", Model: "gpt-4", Type: "openai"},
				"m2": {Provider: "p1", Model: "gpt-4o-mini", Type: "openai"},
			},
			Listeners: []Listener{
				{Name: "l1", Port: 8080, Models: []string{"m1", "m2"}},
			},
			Retry: RetryConfig{DefaultTimeout: time.Second},
		}
		if err := cfg.validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Listeners[0].ResolvedModels) != 2 {
			t.Errorf("expected 2 resolved models, got %d", len(cfg.Listeners[0].ResolvedModels))
		}
		if cfg.Listeners[0].ResolvedModels[0].ID != "m1" {
			t.Errorf(
				"expected first resolved model ID m1, got %s",
				cfg.Listeners[0].ResolvedModels[0].ID,
			)
		}
		if cfg.Listeners[0].ResolvedModels[1].ID != "m2" {
			t.Errorf(
				"expected second resolved model ID m2, got %s",
				cfg.Listeners[0].ResolvedModels[1].ID,
			)
		}
	})
}

func TestValidateBedrockCredentials(t *testing.T) {
	tests := []struct {
		name      string
		provider  Provider
		wantError bool
	}{
		// Valid configurations
		{"no credentials is valid (skip signing)", Provider{}, false},
		{
			"long-term credentials (access + secret)",
			Provider{AWSAccessKeyID: "A", AWSSecretAccessKey: "B"},
			false,
		},
		{
			"long-term credentials with region",
			Provider{AWSRegion: "us-east-1", AWSAccessKeyID: "A", AWSSecretAccessKey: "B"},
			false,
		},
		{
			"temporary credentials (with session token)",
			Provider{AWSAccessKeyID: "A", AWSSecretAccessKey: "B", AWSSessionToken: "C"},
			false,
		},
		{
			"temporary credentials with region",
			Provider{
				AWSRegion:          "us-east-1",
				AWSAccessKeyID:     "A",
				AWSSecretAccessKey: "B",
				AWSSessionToken:    "C",
			},
			false,
		},
		{"only region is valid (no signing, use env)", Provider{AWSRegion: "us-east-1"}, false},

		// Invalid configurations
		{"only access key is invalid", Provider{AWSAccessKeyID: "key"}, true},
		{"only secret key is invalid", Provider{AWSSecretAccessKey: "secret"}, true},
		{"only session token is invalid", Provider{AWSSessionToken: "token"}, true},
		{
			"region + access key (missing secret) is invalid",
			Provider{AWSRegion: "us-east-1", AWSAccessKeyID: "key"},
			true,
		},
		{
			"region + secret key (missing access) is invalid",
			Provider{AWSRegion: "us-east-1", AWSSecretAccessKey: "secret"},
			true,
		},
		{
			"session token without access key is invalid",
			Provider{AWSSessionToken: "token", AWSSecretAccessKey: "secret"},
			true,
		},
		{
			"session token without secret key is invalid",
			Provider{AWSSessionToken: "token", AWSAccessKeyID: "key"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBedrockCredentials("test", tt.provider)
			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestProviderMethods(t *testing.T) {
	t.Setenv("URL_VAR", "http://example.com")
	t.Setenv("REGION", "us-east-2")
	t.Setenv("EMPTY_VAR", "")

	tests := []struct {
		name     string
		provider Provider
		method   func(Provider) string
		expected string
	}{
		{
			"GetURL with env var",
			Provider{URL: "$URL_VAR"},
			func(p Provider) string { return p.GetURL() },
			"http://example.com",
		},
		{
			"GetURL with undefined env var returns original",
			Provider{URL: "$UNDEFINED_VAR_XYZ"},
			func(p Provider) string { return p.GetURL() },
			"$UNDEFINED_VAR_XYZ",
		},
		{
			"GetURL with empty value returns original",
			Provider{URL: "$EMPTY_VAR"},
			func(p Provider) string { return p.GetURL() },
			"$EMPTY_VAR",
		},
		{
			"GetURL static value",
			Provider{URL: "http://static.com"},
			func(p Provider) string { return p.GetURL() },
			"http://static.com",
		},
		{
			"GetAPIKey with env var",
			Provider{APIKey: "$URL_VAR"},
			func(p Provider) string { return p.GetAPIKey() },
			"http://example.com",
		},
		{
			"GetAPIKey static value",
			Provider{APIKey: "sk-12345"},
			func(p Provider) string { return p.GetAPIKey() },
			"sk-12345",
		},
		{
			"GetAPIKey empty",
			Provider{APIKey: ""},
			func(p Provider) string { return p.GetAPIKey() },
			"",
		},
		{
			"GetAWSRegion with env var",
			Provider{AWSRegion: "$REGION"},
			func(p Provider) string { return p.GetAWSRegion() },
			"us-east-2",
		},
		{
			"GetAWSRegion static",
			Provider{AWSRegion: "us-west-2"},
			func(p Provider) string { return p.GetAWSRegion() },
			"us-west-2",
		},
		{
			"GetAWSAccessKeyID static",
			Provider{AWSAccessKeyID: "AKIAIOSFODNN7EXAMPLE"},
			func(p Provider) string { return p.GetAWSAccessKeyID() },
			"AKIAIOSFODNN7EXAMPLE",
		},
		{
			"GetAWSSecretAccessKey static",
			Provider{AWSSecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"},
			func(p Provider) string { return p.GetAWSSecretAccessKey() },
			"wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		},
		{
			"GetAWSSessionToken static",
			Provider{AWSSessionToken: "token123"},
			func(p Provider) string { return p.GetAWSSessionToken() },
			"token123",
		},
		{
			"empty provider returns empty values",
			Provider{},
			func(p Provider) string { return p.GetURL() },
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.method(tt.provider); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGetInterval(t *testing.T) {
	tests := []struct {
		name            string
		modelInterval   time.Duration
		provider        Provider
		defaultInterval time.Duration
		expected        time.Duration
	}{
		{
			"model interval takes priority",
			1 * time.Second,
			Provider{Interval: 2 * time.Second},
			5 * time.Second,
			1 * time.Second,
		},
		{
			"provider interval when model is zero",
			0,
			Provider{Interval: 2 * time.Second},
			5 * time.Second,
			2 * time.Second,
		},
		{
			"default interval when both are zero",
			0,
			Provider{Interval: 0},
			5 * time.Second,
			5 * time.Second,
		},
		{
			"negative model interval uses provider",
			-1 * time.Second,
			Provider{Interval: 2 * time.Second},
			5 * time.Second,
			2 * time.Second,
		},
		{
			"negative provider interval uses default",
			0,
			Provider{Interval: -1 * time.Second},
			5 * time.Second,
			5 * time.Second,
		},
		{"zero default interval returns zero", 0, Provider{Interval: 0}, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{Interval: tt.modelInterval}
			got := m.GetInterval(tt.provider, tt.defaultInterval)
			if got != tt.expected {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}
