// Package app wires the bpnode daemon together and runs its lifecycle.
//
// This is the #27 skeleton: configuration, mode selection, the operational API
// server, and graceful start/stop. The subsystems it announces at startup are
// stubs with their tracking issues — storage (#30), keystore (#28), libp2p
// (#32) — and are wired in by those issues.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bpprotocol/blockparty/node/internal/api"
	"github.com/bpprotocol/blockparty/node/internal/config"
	"github.com/bpprotocol/blockparty/node/internal/obs"
	"github.com/bpprotocol/blockparty/node/internal/world"
)

// Version is the bpnode build version.
const Version = "0.1.0-dev"

// shutdownTimeout bounds graceful draining of the API server.
const shutdownTimeout = 10 * time.Second

// Daemon is the assembled, runnable node.
type Daemon struct {
	cfg     config.Config
	log     *slog.Logger
	metrics *obs.Metrics
	world   *world.State
	api     *api.Server
	start   time.Time
}

// New assembles a Daemon from resolved configuration. It resolves the World
// (which may be unloaded) and constructs the API server, but binds nothing
// until Run.
func New(cfg config.Config, log *slog.Logger) (*Daemon, error) {
	st, err := world.Load(cfg)
	if err != nil {
		return nil, fmt.Errorf("load world: %w", err)
	}
	d := &Daemon{
		cfg:     cfg,
		log:     log,
		metrics: &obs.Metrics{},
		world:   st,
	}
	d.api = api.New(cfg.APIAddr, log, d.metrics, d.status)
	return d, nil
}

// Run starts the daemon and blocks until ctx is cancelled, then drains
// gracefully. It is safe to cancel ctx via signal handling in main.
func (d *Daemon) Run(ctx context.Context) error {
	d.start = time.Now()
	d.logStartup()

	// Stores (#30), keystore (#28), and the p2p host (#32) are wired in by their
	// issues. Announce their pending state so startup is transparent.
	d.log.Debug("subsystem pending", "subsystem", "storage", "issue", 30)
	if d.cfg.Mode == config.ModePersonal {
		d.log.Debug("subsystem pending", "subsystem", "keystore", "issue", 28)
	}
	d.log.Debug("subsystem pending", "subsystem", "p2p", "issue", 32)

	if err := d.api.Start(); err != nil {
		return fmt.Errorf("start api server: %w", err)
	}
	d.log.Info("bpnode ready", "api_addr", d.api.Addr(), "mode", d.cfg.Mode)

	<-ctx.Done()
	d.log.Info("shutting down", "reason", context.Cause(ctx))

	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := d.api.Shutdown(shutCtx); err != nil {
		d.log.Error("api server shutdown", "err", err)
		return err
	}
	d.log.Info("stopped")
	return nil
}

// APIAddr returns the resolved API listen address (valid after Run has started).
func (d *Daemon) APIAddr() string { return d.api.Addr() }

func (d *Daemon) logStartup() {
	d.log.Info("starting bpnode",
		"version", Version,
		"mode", d.cfg.Mode,
		"world_loaded", d.world.Loaded,
		"world", d.world.Fingerprint,
		"data_dir", d.cfg.DataDir,
		"api_addr", d.cfg.APIAddr,
	)
	if !d.world.Loaded {
		d.log.Warn("no World loaded; awaiting client bootstrap", "issue", 38)
	}
}

func (d *Daemon) status() api.Status {
	return api.Status{
		Version:     Version,
		Mode:        string(d.cfg.Mode),
		WorldLoaded: d.world.Loaded,
		World:       d.world.Fingerprint,
		StartedAt:   d.start.UTC().Format(time.RFC3339),
		UptimeSec:   time.Since(d.start).Seconds(),
		Metrics:     d.metrics.Snapshot(),
	}
}
