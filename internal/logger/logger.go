package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

// 根据环境变量配置默认的 slog 日志记录器：
//
//	LOG_FORMAT  text (默认值) | json
//	LOG_LEVEL   debug | info (默认值) | warn | error
//
// 一旦设置，其他地方的代码应通过默认的包级日志记录器使用
// slog.Info / slog.Warn / slog.Error / slog.Debug
//
// 返回配置好的 *slog.Logger，以便调用者可以将其附加到其他系统（例如 GORM 的日志桥接器）
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
