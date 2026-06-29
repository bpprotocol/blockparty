// Package obs provides the daemon's structured logging and lightweight,
// dependency-free metrics counters. Richer observability is layered on later;
// this is the baseline established in #27.
package obs

import (
	"io"
	"log/slog"
	"sync/atomic"
)

// NewLogger builds a slog logger for the given level ("debug"|"info"|"warn"|
// "error") and format ("text"|"json"), writing to w. Unknown values fall back
// to info/text.
func NewLogger(level, format string, w io.Writer) *slog.Logger {
	var lv slog.Level
	switch level {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lv}
	var h slog.Handler
	if format == "json" {
		h = slog.NewJSONHandler(w, opts)
	} else {
		h = slog.NewTextHandler(w, opts)
	}
	return slog.New(h)
}

// Metrics holds the daemon's runtime counters. The zero value is ready to use
// and safe for concurrent access.
type Metrics struct {
	httpRequests atomic.Int64
	httpErrors   atomic.Int64
}

// IncHTTPRequests records one served API request.
func (m *Metrics) IncHTTPRequests() { m.httpRequests.Add(1) }

// IncHTTPErrors records one API request that resulted in a 5xx response.
func (m *Metrics) IncHTTPErrors() { m.httpErrors.Add(1) }

// Snapshot returns a copy of the current counter values for reporting.
func (m *Metrics) Snapshot() map[string]int64 {
	return map[string]int64{
		"http_requests": m.httpRequests.Load(),
		"http_errors":   m.httpErrors.Load(),
	}
}
