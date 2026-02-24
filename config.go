package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Server    ServerConfig        `mapstructure:"server"`
	Log       LogConfig           `mapstructure:"log"`
	Retry     RetryConfig         `mapstructure:"retry"`
	Endpoints map[string]Endpoint `mapstructure:"endpoints"`
	Models    []Model             `mapstructure:"models"`
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
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

// Endpoint represents an upstream API endpoint.
type Endpoint struct {
	URL                string        `mapstructure:"url"`
	APIKey             string        `mapstructure:"api_key"`
	Type               string        `mapstructure:"type"`
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
	Endpoint string        `mapstructure:"endpoint"`
	Type     string        `mapstructure:"type"`
	Model    string        `mapstructure:"model"`
	Attempts int           `mapstructure:"attempts"`
	Timeout  time.Duration `mapstructure:"timeout"`
	Interval time.Duration `mapstructure:"interval"`
}

// GetURL resolves the URL, supporting environment variable expansion.
func (e *Endpoint) GetURL() string {
	return resolveEnvOrValue(e.URL)
}

// GetAPIKey resolves the API key, supporting environment variable expansion.
func (e *Endpoint) GetAPIKey() string {
	return resolveEnvOrValue(e.APIKey)
}

// GetInterval returns the model's interval, or the endpoint's interval if not set.
func (m *Model) GetInterval(endpoint Endpoint, defaultInterval time.Duration) time.Duration {
	if m.Interval > 0 {
		return m.Interval
	}
	if endpoint.Interval > 0 {
		return endpoint.Interval
	}
	return defaultInterval
}

// GetAWSRegion returns the AWS region, falling back to environment variables.
func (e *Endpoint) GetAWSRegion() string {
	return resolveEnvOrValue(e.AWSRegion)
}

// GetAWSAccessKeyID returns the AWS access key ID, falling back to environment variables.
func (e *Endpoint) GetAWSAccessKeyID() string {
	return resolveEnvOrValue(e.AWSAccessKeyID)
}

// GetAWSSecretAccessKey returns the AWS secret access key, falling back to environment variables.
func (e *Endpoint) GetAWSSecretAccessKey() string {
	return resolveEnvOrValue(e.AWSSecretAccessKey)
}

// GetAWSSessionToken returns the AWS session token, falling back to environment variables.
func (e *Endpoint) GetAWSSessionToken() string {
	return resolveEnvOrValue(e.AWSSessionToken)
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
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = time.Minute
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 10 * time.Minute
	}
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
}

// validate checks the configuration for errors and parses derived fields.
func (c *Config) validate() error {
	if len(c.Models) == 0 {
		return errors.New("at least one model must be configured")
	}

	// Parse endpoint URLs
	for name, ep := range c.Endpoints {
		resolvedURL := ep.GetURL()
		parsedURL, err := url.Parse(resolvedURL)
		if err != nil {
			return fmt.Errorf("invalid URL for endpoint %q: %w", name, err)
		}
		// Normalize path by removing trailing slashes
		parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
		ep.ParsedURL = parsedURL
		c.Endpoints[name] = ep
	}

	// Validate models
	configType := ""
	for i := range c.Models {
		m := &c.Models[i]

		if m.Endpoint == "" {
			return fmt.Errorf("model %d: endpoint is required", i)
		}
		endpoint, ok := c.Endpoints[m.Endpoint]
		if !ok {
			return fmt.Errorf("model %d: endpoint %q not found", i, m.Endpoint)
		}
		if m.Model == "" {
			return fmt.Errorf("model %d: model is required", i)
		}
		if m.Type == "" {
			return fmt.Errorf("model %d: type is required", i)
		}
		if !isSupportedModelType(m.Type) {
			return fmt.Errorf("model %d: unsupported type %q (supported: openai, anthropic, bedrock)", i, m.Type)
		}
		if configType == "" {
			configType = m.Type
		} else if m.Type != configType {
			return fmt.Errorf("model %d: mixed model types are not allowed (expected %q, got %q)", i, configType, m.Type)
		}
		if m.Attempts <= 0 {
			m.Attempts = 1
		}
		if m.Timeout == 0 {
			m.Timeout = c.Retry.DefaultTimeout
		}

		// Validate bedrock endpoint credentials
		if m.Type == "bedrock" {
			if err := validateBedrockCredentials(m.Endpoint, endpoint); err != nil {
				return fmt.Errorf("model %d: %w", i, err)
			}
		}
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

// validateBedrockCredentials validates AWS credentials for bedrock endpoints.
// For long-term credentials: aws_access_key_id + aws_secret_access_key are required.
// For temporary credentials: aws_session_token is additionally required.
// If no credentials are configured, signing is skipped (use environment variables or IAM roles).
func validateBedrockCredentials(endpointName string, ep Endpoint) error {
	hasAccessKeyID := ep.AWSAccessKeyID != ""
	hasSecretAccessKey := ep.AWSSecretAccessKey != ""
	hasSessionToken := ep.AWSSessionToken != ""

	// access_key_id and secret_access_key must be configured together
	if hasAccessKeyID != hasSecretAccessKey {
		return fmt.Errorf(
			"endpoint %q: bedrock requires both aws_access_key_id and aws_secret_access_key to be configured together",
			endpointName,
		)
	}

	// session_token requires access_key_id and secret_access_key
	if hasSessionToken && !hasAccessKeyID {
		return fmt.Errorf(
			"endpoint %q: bedrock aws_session_token requires aws_access_key_id and aws_secret_access_key",
			endpointName,
		)
	}

	return nil
}
