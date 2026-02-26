package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Log       LogConfig           `mapstructure:"log"`
	Retry     RetryConfig         `mapstructure:"retry"`
	Providers map[string]Provider `mapstructure:"providers"`
	Models    map[string]Model    `mapstructure:"models"`
	Listeners []Listener          `mapstructure:"listeners"`
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level            string `mapstructure:"level"`
	IncludeErrorBody bool   `mapstructure:"include_error_body"`
}

// RetryConfig holds retry-related configuration.
type RetryConfig struct {
	MaxCycles          int           `mapstructure:"max_cycles"`
	DefaultTimeout     time.Duration `mapstructure:"default_timeout"`
	DefaultInterval    time.Duration `mapstructure:"default_interval"`
	ExponentialBackoff bool          `mapstructure:"exponential_backoff"`
}

// Provider represents an upstream API provider.
type Provider struct {
	URL                string        `mapstructure:"url"`
	APIKey             string        `mapstructure:"api_key"`
	StripVersionPrefix bool          `mapstructure:"strip_version_prefix"`
	Interval           time.Duration `mapstructure:"interval"`
	AWSRegion          string        `mapstructure:"aws_region"`
	AWSAccessKeyID     string        `mapstructure:"aws_access_key_id"`
	AWSSecretAccessKey string        `mapstructure:"aws_secret_access_key"`
	AWSSessionToken    string        `mapstructure:"aws_session_token"`
	ParsedURL          *url.URL      `mapstructure:"-"`
}

// Model represents a model configuration with retry settings.
type Model struct {
	ID       string        // Global unique ID (map key)
	Provider string        `mapstructure:"provider"`
	Model    string        `mapstructure:"model"`
	Type     string        `mapstructure:"type"`
	Attempts int           `mapstructure:"attempts"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Interval time.Duration `mapstructure:"interval"`
}

// Listener represents a local listening configuration.
type Listener struct {
	Name         string        `mapstructure:"name"`
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	Models       []string      `mapstructure:"models"` // Model IDs

	// Resolved at runtime
	ResolvedModels []Model `mapstructure:"-"`
	ConfigType     string  `mapstructure:"-"` // Unified API type for this listener
}

// GetURL resolves the URL, supporting environment variable expansion.
func (p *Provider) GetURL() string {
	return resolveEnvOrValue(p.URL)
}

// GetAPIKey resolves the API key, supporting environment variable expansion.
func (p *Provider) GetAPIKey() string {
	return resolveEnvOrValue(p.APIKey)
}

// GetInterval returns the model's interval, or the provider's interval if not set.
func (m *Model) GetInterval(provider Provider, defaultInterval time.Duration) time.Duration {
	if m.Interval > 0 {
		return m.Interval
	}
	if provider.Interval > 0 {
		return provider.Interval
	}
	return defaultInterval
}

// GetAWSRegion returns the AWS region, falling back to environment variables.
func (p *Provider) GetAWSRegion() string {
	return resolveEnvOrValue(p.AWSRegion)
}

// GetAWSAccessKeyID returns the AWS access key ID, falling back to environment variables.
func (p *Provider) GetAWSAccessKeyID() string {
	return resolveEnvOrValue(p.AWSAccessKeyID)
}

// GetAWSSecretAccessKey returns the AWS secret access key, falling back to environment variables.
func (p *Provider) GetAWSSecretAccessKey() string {
	return resolveEnvOrValue(p.AWSSecretAccessKey)
}

// GetAWSSessionToken returns the AWS session token, falling back to environment variables.
func (p *Provider) GetAWSSessionToken() string {
	return resolveEnvOrValue(p.AWSSessionToken)
}

// resolveEnvOrValue returns the environment variable value if the input starts with $,
// otherwise returns the input as-is.
func resolveEnvOrValue(v string) string {
	if v == "" {
		return ""
	}
	if len(v) > 1 && v[0] == '$' {
		if envVal := os.Getenv(v[1:]); envVal != "" {
			return envVal
		}
	}
	return v
}

// loadConfig reads and validates the configuration from viper.
func loadConfig() (*Config, error) {
	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	applyDefaults(&cfg)

	// Set log level early so validation logs are visible
	logger.SetLevel(parseLogLevel(cfg.Log.Level))

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// applyDefaults sets default values for unset configuration fields.
func applyDefaults(c *Config) {
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Retry.MaxCycles == 0 {
		c.Retry.MaxCycles = 10
	}
	if c.Retry.DefaultTimeout == 0 {
		c.Retry.DefaultTimeout = 30 * time.Second
	}
	if c.Retry.DefaultInterval == 0 {
		c.Retry.DefaultInterval = 100 * time.Millisecond
	}

	// Apply listener defaults
	for i := range c.Listeners {
		l := &c.Listeners[i]
		if l.Host == "" {
			l.Host = "127.0.0.1"
		}
		if l.ReadTimeout == 0 {
			l.ReadTimeout = time.Minute
		}
		if l.WriteTimeout == 0 {
			l.WriteTimeout = 10 * time.Minute
		}
	}
}

// validate checks the configuration for errors and parses derived fields.
func (c *Config) validate() error {
	// Validate providers
	if len(c.Providers) == 0 {
		return errors.New("at least one provider must be configured")
	}

	// Parse and validate provider URLs
	for name, p := range c.Providers {
		resolvedURL := p.GetURL()
		parsedURL, err := url.Parse(resolvedURL)
		if err != nil {
			return fmt.Errorf("invalid URL for provider %q: %w", name, err)
		}
		if parsedURL.Scheme == "" || parsedURL.Host == "" {
			return fmt.Errorf(
				"invalid URL for provider %q: must include scheme and host, got %q",
				name,
				resolvedURL,
			)
		}

		scheme := strings.ToLower(parsedURL.Scheme)
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf(
				"invalid URL for provider %q: unsupported scheme %q (supported: http, https)",
				name,
				parsedURL.Scheme,
			)
		}

		// Normalize path by removing trailing slashes
		parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
		p.ParsedURL = parsedURL
		c.Providers[name] = p
	}

	// Validate models
	if len(c.Models) == 0 {
		return errors.New("at least one model must be configured")
	}

	for id, m := range c.Models {
		m.ID = id

		if m.Provider == "" {
			return fmt.Errorf("model %q: provider is required", id)
		}
		provider, ok := c.Providers[m.Provider]
		if !ok {
			return fmt.Errorf("model %q: provider %q not found", id, m.Provider)
		}
		if m.Model == "" {
			return fmt.Errorf("model %q: model is required", id)
		}
		if m.Type == "" {
			return fmt.Errorf("model %q: type is required", id)
		}
		if !isSupportedModelType(m.Type) {
			return fmt.Errorf(
				"model %q: unsupported type %q (supported: openai, anthropic, bedrock)",
				id,
				m.Type,
			)
		}
		if m.Attempts <= 0 {
			m.Attempts = 1
		}
		if m.Timeout == 0 {
			m.Timeout = c.Retry.DefaultTimeout
		}

		// Validate bedrock provider credentials
		if m.Type == "bedrock" {
			if err := validateBedrockCredentials(m.Provider, provider); err != nil {
				return fmt.Errorf("model %q: %w", id, err)
			}
		}

		c.Models[id] = m
	}

	// Validate listeners
	if len(c.Listeners) == 0 {
		return errors.New("at least one listener must be configured")
	}

	listenerNames := make(map[string]struct{}, len(c.Listeners))
	listenerAddrs := make(map[string]string, len(c.Listeners))

	for i := range c.Listeners {
		l := &c.Listeners[i]

		if l.Name == "" {
			return fmt.Errorf("listener %d: name is required", i)
		}
		if _, exists := listenerNames[l.Name]; exists {
			return fmt.Errorf("listener %q: duplicate name", l.Name)
		}
		listenerNames[l.Name] = struct{}{}

		if l.Port == 0 {
			return fmt.Errorf("listener %q: port is required", l.Name)
		}
		if l.Port < 1 || l.Port > 65535 {
			return fmt.Errorf(
				"listener %q: port must be between 1 and 65535, got %d",
				l.Name,
				l.Port,
			)
		}

		listenerAddr := net.JoinHostPort(l.Host, strconv.Itoa(l.Port))
		if existingName, exists := listenerAddrs[listenerAddr]; exists {
			return fmt.Errorf(
				"listener %q: duplicate listen address %q (already used by listener %q)",
				l.Name,
				listenerAddr,
				existingName,
			)
		}
		listenerAddrs[listenerAddr] = l.Name

		if len(l.Models) == 0 {
			return fmt.Errorf("listener %q: must reference at least one model", l.Name)
		}

		// Resolve models and validate type consistency
		l.ResolvedModels = make([]Model, 0, len(l.Models))
		listenerType := ""

		for _, modelID := range l.Models {
			m, ok := c.Models[modelID]
			if !ok {
				return fmt.Errorf("listener %q: model %q not found", l.Name, modelID)
			}

			if listenerType == "" {
				listenerType = m.Type
			} else if m.Type != listenerType {
				return fmt.Errorf(
					"listener %q: mixed model types are not allowed (expected %q, got %q from model %q)",
					l.Name,
					listenerType,
					m.Type,
					modelID,
				)
			}

			l.ResolvedModels = append(l.ResolvedModels, m)
		}

		l.ConfigType = listenerType
	}

	return nil
}

func isSupportedModelType(modelType string) bool {
	switch modelType {
	case "openai", "anthropic", "bedrock":
		return true
	default:
		return false
	}
}

// validateBedrockCredentials validates AWS credentials for bedrock providers.
// For long-term credentials: aws_access_key_id + aws_secret_access_key are required.
// For temporary credentials: aws_session_token is additionally required.
// If no credentials are configured, signing is skipped (use environment variables or IAM roles).
func validateBedrockCredentials(providerName string, p Provider) error {
	hasAccessKeyID := p.AWSAccessKeyID != ""
	hasSecretAccessKey := p.AWSSecretAccessKey != ""
	hasSessionToken := p.AWSSessionToken != ""

	// access_key_id and secret_access_key must be configured together
	if hasAccessKeyID != hasSecretAccessKey {
		return fmt.Errorf(
			"provider %q: bedrock requires both aws_access_key_id and aws_secret_access_key to be configured together",
			providerName,
		)
	}

	// session_token requires access_key_id and secret_access_key
	if hasSessionToken && !hasAccessKeyID {
		return fmt.Errorf(
			"provider %q: bedrock aws_session_token requires aws_access_key_id and aws_secret_access_key",
			providerName,
		)
	}

	return nil
}
