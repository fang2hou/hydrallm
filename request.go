package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/tidwall/sjson"
)

// setModel overrides the model field in a JSON request body.
func setModel(body []byte, model string) ([]byte, error) {
	return sjson.SetBytes(body, "model", model)
}

// isStreamingRequest checks if the request is a streaming request.
func isStreamingRequest(req *http.Request, body []byte) bool {
	// Check URL path for streaming endpoints
	path := req.URL.Path
	if strings.Contains(path, "-stream") || strings.Contains(path, "/stream") {
		return true
	}

	// Check Accept header for SSE
	accept := req.Header.Get("Accept")
	if strings.Contains(accept, "text/event-stream") {
		return true
	}

	// Check body for stream:true (OpenAI style)
	var reqBody struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &reqBody); err == nil && reqBody.Stream {
		return true
	}

	return false
}

// readErrorBody reads and optionally decompresses an error response body.
func readErrorBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer func() { _ = gzReader.Close() }()
		reader = gzReader
	}

	return io.ReadAll(io.LimitReader(reader, 4*1024))
}

// formatBodyForLog formats a request body for logging, truncating if too large.
func formatBodyForLog(body []byte) string {
	const maxLogSize = 2048

	if len(body) == 0 {
		return "(empty)"
	}

	var jsonBody any
	if err := json.Unmarshal(body, &jsonBody); err == nil {
		formatted, err := json.MarshalIndent(jsonBody, "", "  ")
		if err == nil {
			if len(formatted) > maxLogSize {
				return string(formatted[:maxLogSize]) + "\n... (truncated)"
			}
			return string(formatted)
		}
	}

	if len(body) > maxLogSize {
		return string(body[:maxLogSize]) + "\n... (truncated)"
	}
	return string(body)
}
