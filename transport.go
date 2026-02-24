package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/charmbracelet/log"
)

var versionPrefixRegex = regexp.MustCompile(`^/v\d+`)

// RetryTransport implements http.RoundTripper with retry and fallback logic.
type RetryTransport struct {
	config          *Config
	logger          *log.Logger
	defaultInterval time.Duration
	client          *http.Client
}

// newRetryTransport creates a transport with retry and model fallback capabilities.
func newRetryTransport(cfg *Config, logger *log.Logger) *RetryTransport {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &RetryTransport{
		config:          cfg,
		logger:          logger,
		defaultInterval: cfg.Retry.DefaultInterval,
		client:          &http.Client{Transport: transport},
	}
}

// RoundTrip implements http.RoundTripper with retry logic.
func (t *RetryTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	ctx := req.Context()

	// Read and buffer body with limit to prevent memory exhaustion
	var body []byte
	if req.Body != nil {
		const maxBodySize = 100 * 1024 * 1024 // 100MB max
		body, err = io.ReadAll(io.LimitReader(req.Body, maxBodySize))
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		_ = req.Body.Close()
	}

	isStreaming := isStreamingRequest(req, body)
	debugEnabled := isDebugEnabled(t.logger)
	maxCycles := max(t.config.Retry.MaxCycles, 1)
	exponentialBackoff := t.config.Retry.ExponentialBackoff

	var lastErr error
	var lastResp *http.Response
	totalAttempts := 0

	for cycle := range maxCycles {
		for modelIdx, model := range t.config.Models {
			endpoint := t.config.Endpoints[model.Endpoint]
			interval := model.GetInterval(endpoint, t.defaultInterval)

			for attempt := range model.Attempts {
				if err = ctx.Err(); err != nil {
					return nil, err
				}

				totalAttempts++
				t.logger.Debug(
					"trying model",
					"endpoint",
					model.Endpoint,
					"model",
					model.Model,
					"cycle",
					cycle+1,
					"attempt",
					attempt+1,
					"total_attempts",
					totalAttempts,
				)
				resp, err = t.tryModel(ctx, req, body, model, isStreaming, debugEnabled)
				if err != nil {
					t.logger.Debug("model request failed", "endpoint", model.Endpoint, "error", err)
					lastErr = err

					// Wait before next attempt
					if t.shouldWait(
						cycle,
						modelIdx,
						attempt,
						len(t.config.Models),
						model.Attempts,
						maxCycles,
					) {
						t.wait(ctx, interval, totalAttempts, exponentialBackoff)
					}
					continue
				}

				t.logger.Info(
					"response",
					"endpoint",
					model.Endpoint,
					"model",
					model.Model,
					"status",
					resp.StatusCode,
					"streaming",
					isStreaming,
				)

				if isRetryable(resp.StatusCode) {
					t.handleRetryableResponse(resp, model.Endpoint)
					lastResp = resp

					// Wait before next attempt
					if t.shouldWait(
						cycle,
						modelIdx,
						attempt,
						len(t.config.Models),
						model.Attempts,
						maxCycles,
					) {
						t.wait(ctx, interval, totalAttempts, exponentialBackoff)
					}
					continue
				}

				if resp.StatusCode >= 400 {
					t.handleErrorResponse(resp, model)
				}

				return resp, nil
			}
		}
	}

	if lastResp != nil {
		return lastResp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("all attempts exhausted")
}

// shouldWait determines if we should wait before the next attempt.
func (t *RetryTransport) shouldWait(
	cycle, modelIdx, attempt, numModels, modelAttempts, maxCycles int,
) bool {
	// Don't wait if this is the last attempt of the last model in the last cycle
	if cycle == maxCycles-1 && modelIdx == numModels-1 && attempt == modelAttempts-1 {
		return false
	}
	return true
}

// wait pauses execution with optional exponential backoff.
func (t *RetryTransport) wait(
	ctx context.Context,
	interval time.Duration,
	totalAttempts int,
	exponentialBackoff bool,
) {
	waitDuration := interval
	if exponentialBackoff {
		waitDuration = interval * time.Duration(totalAttempts)
	}

	t.logger.Debug(
		"waiting before retry",
		"duration",
		waitDuration,
		"exponential",
		exponentialBackoff,
	)
	select {
	case <-ctx.Done():
	case <-time.After(waitDuration):
	}
}

// tryModel attempts to send a request through a specific model endpoint.
func (t *RetryTransport) tryModel(
	ctx context.Context,
	originalReq *http.Request,
	body []byte,
	model Model,
	isStreaming bool,
	debugEnabled bool,
) (*http.Response, error) {
	endpoint, ok := t.config.Endpoints[model.Endpoint]
	if !ok {
		return nil, fmt.Errorf("endpoint %q not found", model.Endpoint)
	}

	// Modify body with model override
	newBody, err := setModel(body, model.Model)
	if err != nil {
		return nil, fmt.Errorf("failed to set model: %w", err)
	}

	if debugEnabled {
		t.logger.Debug("request body", "body", formatBodyForLog(newBody))
	}

	// Clone request
	newReq := originalReq.Clone(ctx)
	newReq.Body = io.NopCloser(bytes.NewReader(newBody))
	newReq.ContentLength = int64(len(newBody))
	newReq.RequestURI = "" // Must be empty for client requests

	// Build target URL
	t.buildTargetURL(newReq, originalReq, endpoint)

	if debugEnabled {
		t.logger.Debug("request url", "url", newReq.URL.String())
	}

	// Set authorization headers
	t.setAuthHeaders(newReq, model.Type, endpoint)

	// Set context with timeout (skip for streaming to avoid mid-stream cancellation)
	if !isStreaming {
		reqCtx, cancel := context.WithTimeout(ctx, model.Timeout)
		defer cancel()
		newReq = newReq.WithContext(reqCtx)
	}

	return t.client.Do(newReq)
}

// buildTargetURL constructs the target URL for the upstream request.
func (t *RetryTransport) buildTargetURL(
	newReq *http.Request,
	originalReq *http.Request,
	endpoint Endpoint,
) {
	targetURL := endpoint.ParsedURL
	reqPath := originalReq.URL.Path

	if endpoint.StripVersionPrefix {
		reqPath = versionPrefixRegex.ReplaceAllString(reqPath, "")
	}

	// Normalize paths to avoid double slashes
	// Remove all trailing slashes from the base path
	basePath := strings.TrimRight(targetURL.Path, "/")

	urlCopy := *originalReq.URL
	newReq.URL = &urlCopy
	newReq.URL.Scheme = targetURL.Scheme
	newReq.URL.Host = targetURL.Host
	newReq.URL.Path = basePath + reqPath
	newReq.Host = targetURL.Host
}

// setAuthHeaders configures authorization headers based on provider type.
func (t *RetryTransport) setAuthHeaders(req *http.Request, modelType string, endpoint Endpoint) {
	apiKey := endpoint.GetAPIKey()

	switch modelType {
	case "anthropic":
		if apiKey == "-" {
			req.Header.Del("x-api-key")
		} else if apiKey != "" {
			req.Header.Set("x-api-key", apiKey)
		}
		req.Header.Set("anthropic-version", "2023-06-01")
	case "bedrock":
		t.signAWSRequest(req, endpoint)
	default: // openai
		if apiKey == "-" {
			req.Header.Del("Authorization")
		} else if apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}
}

// handleRetryableResponse logs and closes a retryable response.
func (t *RetryTransport) handleRetryableResponse(resp *http.Response, endpoint string) {
	if t.config.Log.IncludeErrorBody {
		errBody, err := readErrorBody(resp)
		if err != nil {
			t.logger.Warn("failed to read error body", "error", err)
		}
		_ = resp.Body.Close()
		t.logger.Info(
			"retryable status",
			"endpoint",
			endpoint,
			"status",
			resp.StatusCode,
			"error",
			string(errBody),
		)
	} else {
		_, _ = io.Copy(io.Discard, resp.Body)
		t.logger.Info("retryable status", "endpoint", endpoint, "status", resp.StatusCode)
		_ = resp.Body.Close()
	}
}

// handleErrorResponse logs error response details.
func (t *RetryTransport) handleErrorResponse(resp *http.Response, model Model) {
	if t.config.Log.IncludeErrorBody {
		errBody, err := readErrorBody(resp)
		if err != nil {
			t.logger.Warn("failed to read error body", "error", err)
		}
		_ = resp.Body.Close()
		t.logger.Info(
			"error status",
			"endpoint",
			model.Endpoint,
			"model",
			model.Model,
			"status",
			resp.StatusCode,
			"error",
			string(errBody),
		)
		resp.Body = io.NopCloser(bytes.NewReader(errBody))
	} else {
		t.logger.Info(
			"error status",
			"endpoint",
			model.Endpoint,
			"model",
			model.Model,
			"status",
			resp.StatusCode,
		)
	}
}

// isRetryable returns true if the status code indicates a retryable error.
func isRetryable(statusCode int) bool {
	return statusCode >= 500 || statusCode == 429
}

// signAWSRequest signs the request with AWS SigV4 for Bedrock using AWS SDK.
// Only signs if AWS credentials are configured in the endpoint; otherwise skips signing.
func (t *RetryTransport) signAWSRequest(req *http.Request, endpoint Endpoint) {
	// Check if credentials are configured in endpoint (not environment variables)
	if endpoint.AWSAccessKeyID == "" {
		return
	}

	region := endpoint.GetAWSRegion()
	if region == "" {
		region = "us-east-1"
	}

	accessKeyID := endpoint.GetAWSAccessKeyID()
	secretAccessKey := endpoint.GetAWSSecretAccessKey()
	sessionToken := endpoint.GetAWSSessionToken()

	credsProvider := credentials.NewStaticCredentialsProvider(
		accessKeyID,
		secretAccessKey,
		sessionToken,
	)
	creds, err := credsProvider.Retrieve(req.Context())
	if err != nil {
		t.logger.Warn("failed to retrieve AWS credentials", "error", err)
		return
	}

	signer := v4.NewSigner()

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Read body for signing
	var bodyBytes []byte
	if req.Body != nil {
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			t.logger.Warn("failed to read request body for signing", "error", err)
			return
		}
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	hash := sha256.Sum256(bodyBytes)
	payloadHash := hex.EncodeToString(hash[:])

	err = signer.SignHTTP(req.Context(), creds, req, payloadHash, "bedrock", region, time.Now())
	if err != nil {
		t.logger.Warn("failed to sign AWS request", "error", err)
	}
}
