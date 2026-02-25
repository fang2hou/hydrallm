package main

import (
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

var logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportCaller:    true,
	ReportTimestamp: true,
	TimeFormat:      time.Kitchen,
})

// parseLogLevel converts string level to log.Level.
func parseLogLevel(level string) log.Level {
	switch strings.ToLower(level) {
	case "debug":
		return log.DebugLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

// isDebugEnabled checks if debug logging is enabled.
func isDebugEnabled(l *log.Logger) bool {
	return l.GetLevel() <= log.DebugLevel
}
