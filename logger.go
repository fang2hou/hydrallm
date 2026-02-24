package main

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// levelWriter routes log output based on level.
// Error level goes to stderr, others (warn/info/debug) go to stdout.
type levelWriter struct {
	stdout io.Writer
	stderr io.Writer
}

func (w *levelWriter) Write(p []byte) (n int, err error) {
	// Check if this is an error level log by looking for "level=error" or "ERROR"
	s := string(p)
	if strings.Contains(s, "level=error") || strings.Contains(s, "ERROR") {
		return w.stderr.Write(p)
	}
	return w.stdout.Write(p)
}

var logger = log.NewWithOptions(&levelWriter{
	stdout: os.Stdout,
	stderr: os.Stderr,
}, log.Options{
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
