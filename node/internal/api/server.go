// Package api serves the node's client-facing HTTP surface.
//
// API framework decision (#27): the client API is Connect
// (https://connectrpc.com/connect), which serves over net/http and generates
// first-class TypeScript clients for the desktop frontend (#26). This package
// establishes the net/http server and the operational endpoints (/healthz,
// /statusz); the Connect service handlers (read/post/subscribe/identity/
// connections, world-status & bootstrap) are mounted onto the same mux in #38.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/bpprotocol/blockparty/node/internal/obs"
)

// Status is the operational snapshot returned by /statusz. It is the
// hand-rolled precursor to the typed WorldStatus RPC added in #38.
type Status struct {
	Version     string           `json:"version"`
	Mode        string           `json:"mode"`
	WorldLoaded bool             `json:"world_loaded"`
	World       string           `json:"world,omitempty"` // World fingerprint when loaded
	Blocks      int              `json:"blocks"`          // stored block count (-1 if unavailable)
	StartedAt   string           `json:"started_at"`
	UptimeSec   float64          `json:"uptime_seconds"`
	Metrics     map[string]int64 `json:"metrics"`
}

// StatusFunc supplies a fresh Status on each request.
type StatusFunc func() Status

// Server is the node's HTTP API server.
type Server struct {
	addr    string
	log     *slog.Logger
	metrics *obs.Metrics
	status  StatusFunc
	http    *http.Server
}

// New builds the API server bound (on Start) to addr.
func New(addr string, log *slog.Logger, metrics *obs.Metrics, status StatusFunc) *Server {
	s := &Server{addr: addr, log: log, metrics: metrics, status: status}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/statusz", s.handleStatus)
	// #38 mounts the Connect service handlers onto this same mux.

	s.http = &http.Server{Handler: s.instrument(mux)}
	return s
}

// Start binds the listener and serves in the background. It resolves the bound
// address (so a configured port of :0 becomes concrete and observable via Addr).
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.addr = ln.Addr().String()
	go func() {
		if err := s.http.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("api server stopped unexpectedly", "err", err)
		}
	}()
	return nil
}

// Addr returns the resolved listen address (valid after Start).
func (s *Server) Addr() string { return s.addr }

// Shutdown gracefully drains the server.
func (s *Server) Shutdown(ctx context.Context) error { return s.http.Shutdown(ctx) }

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.status())
}

// instrument counts requests and 5xx responses.
func (s *Server) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.metrics.IncHTTPRequests()
		rec := &statusRecorder{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.code >= http.StatusInternalServerError {
			s.metrics.IncHTTPErrors()
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	code int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
