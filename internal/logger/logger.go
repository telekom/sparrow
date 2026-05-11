// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package logger

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	// formatText is the text format for logging
	formatText = "TEXT"
	// formatJSON is the JSON format for logging
	formatJSON = "JSON"

	// levelDebug is the debug log level
	levelDebug = "DEBUG"
	// levelInfo is the info log level
	levelInfo = "INFO"
	// levelWarn is the warn log level
	levelWarn = "WARN"
	// levelError is the error log level
	levelError = "ERROR"
)

type logger struct{}

// NewLogger creates a new slog.Logger instance.
// If handlers are provided, the first handler in the slice is used; otherwise,
// a default JSON handler writing to os.Stderr is used. This function allows for
// custom configuration of logging handlers.
func NewLogger(h ...slog.Handler) *slog.Logger {
	var handler slog.Handler
	if len(h) > 0 {
		handler = h[0]
	} else {
		handler = newHandler()
	}
	return slog.New(handler)
}

// NewContextWithLogger creates a new context based on the provided parent context.
// It embeds a logger into this new context.
// It also returns a cancel function to cancel the new context.
func NewContextWithLogger(parent context.Context) (context.Context, context.CancelFunc) {
	//gosec:disable G118 -- This is a false positive, since it's the callers responsibility to call the cancel function
	ctx, cancel := context.WithCancel(parent)
	return IntoContext(ctx, FromContext(parent)), cancel
}

// IntoContext embeds the provided slog.Logger into the given context and returns the modified context.
// This function is used for passing loggers through context, allowing for context-aware logging.
func IntoContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, logger{}, log)
}

// FromContext extracts the slog.Logger from the provided context.
// If the context does not have a logger, it returns a new logger with the default configuration.
// This function is useful for retrieving loggers from context in different parts of an application.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(logger{}).(*slog.Logger); ok {
			return logger
		}
	}
	return NewLogger()
}

// Middleware takes the logger from the context and adds it to the request context
func Middleware(ctx context.Context) func(http.Handler) http.Handler {
	log := FromContext(ctx)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqCtx := IntoContext(r.Context(), log)
			next.ServeHTTP(w, r.WithContext(reqCtx))
		})
	}
}

// newHandler creates a new slog.Handler based on the environment variables.
func newHandler() slog.Handler {
	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     getLevel(os.Getenv("LOG_LEVEL")),
	}

	if strings.ToUpper(os.Getenv("LOG_FORMAT")) == formatText {
		opts.ReplaceAttr = func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				v := a.Value.Any().(time.Time)
				a.Value = slog.StringValue(v.Format(time.TimeOnly))
			}
			return a
		}
		return slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.NewJSONHandler(os.Stderr, opts)
}

// getLevel takes a level string and maps it to the corresponding slog.Level
// Returns the level if no mapped level is found it returns info level
func getLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case levelDebug:
		return slog.LevelDebug
	case levelInfo:
		return slog.LevelInfo
	case levelWarn, "WARNING":
		return slog.LevelWarn
	case levelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
