package main

import (
	"io"
	"testing"

	"github.com/charmbracelet/log"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected log.Level
	}{
		{"debug", log.DebugLevel},
		{"DEBUG", log.DebugLevel},
		{"warn", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"info", log.InfoLevel},
		{"unknown", log.InfoLevel},
		{"", log.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseLogLevel(tt.input); got != tt.expected {
				t.Errorf("parseLogLevel(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsDebugEnabled(t *testing.T) {
	tests := []struct {
		name          string
		level         log.Level
		expectEnabled bool
	}{
		{"info level", log.InfoLevel, false},
		{"warn level", log.WarnLevel, false},
		{"error level", log.ErrorLevel, false},
		{"debug level", log.DebugLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := log.New(io.Discard)
			l.SetLevel(tt.level)
			got := isDebugEnabled(l)
			if got != tt.expectEnabled {
				t.Errorf("isDebugEnabled() = %v, want %v", got, tt.expectEnabled)
			}
		})
	}
}
