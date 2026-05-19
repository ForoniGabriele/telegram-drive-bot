package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger based on environment variables:
//
//	LOG_FORMAT  text (default) | json
//	LOG_LEVEL   debug | info (default) | warn | error
//
// Once set, code elsewhere should use slog.Info / slog.Warn / slog.Error / slog.Debug
// via the default package logger.
//
// Returns the configured *slog.Logger so callers may attach it to other systems
// (e.g. a GORM logger bridge).
func Init(w io.Writer) (*slog.Logger, error) {
	if w == nil {
		w = os.Stderr
	}

	level, err := parseLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT"))) {
	case "json":
		handler = slog.NewJSONHandler(w, opts)
	case "", "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		return nil, fmt.Errorf("LOG_FORMAT must be 'text' or 'json'")
	}

	l := slog.New(handler)
	slog.SetDefault(l)
	return l, nil
}

// parseLevel maps a string to a slog.Level. Empty string → Info (default).
func parseLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error (got %q)", raw)
	}
}
