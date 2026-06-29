// Package app wires the bpnode daemon together and runs its lifecycle:
// configuration, mode selection, the keystore (#28, personal mode), local
// storage (#30), the operational API server, and graceful start/stop. The
// remaining subsystems (libp2p #32 onward) are wired in by their issues.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/bpprotocol/blockparty/node/internal/api"
	"github.com/bpprotocol/blockparty/node/internal/authz"
	"github.com/bpprotocol/blockparty/node/internal/config"
	"github.com/bpprotocol/blockparty/node/internal/core"
	"github.com/bpprotocol/blockparty/node/internal/exchange"
	"github.com/bpprotocol/blockparty/node/internal/gossip"
	"github.com/bpprotocol/blockparty/node/internal/keystore"
	"github.com/bpprotocol/blockparty/node/internal/nodeapi"
	"github.com/bpprotocol/blockparty/node/internal/nodepb/nodepbconnect"
	"github.com/bpprotocol/blockparty/node/internal/obs"
	"github.com/bpprotocol/blockparty/node/internal/p2p"
	"github.com/bpprotocol/blockparty/node/internal/store"
	"github.com/bpprotocol/blockparty/node/internal/world"
)

// Version is the bpnode build version.
const Version = "0.1.0-dev"

// shutdownTimeout bounds graceful draining of the API server.
const shutdownTimeout = 10 * time.Second

// Daemon is the assembled, runnable node.
type Daemon struct {
	cfg      config.Config
	log      *slog.Logger
	metrics  *obs.Metrics
	world    *world.State
	keystore *keystore.Keystore
	store    *store.Store
	core     *core.Core
	p2p      *p2p.Host
	exchange *exchange.Exchange
	gossip   *gossip.Gossip
	token    string
	api      *api.Server
	start    time.Time
}

// New assembles a Daemon from resolved configuration. It resolves the World
// (which may be unloaded) and constructs the API server, but binds nothing
// until Run.
func New(cfg config.Config, log *slog.Logger) (*Daemon, error) {
	st, ks, err := resolveWorld(cfg, log)
	if err != nil {
		return nil, err
	}
	token, err := authz.LoadOrCreateToken(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("api token: %w", err)
	}
	d := &Daemon{
		cfg:      cfg,
		log:      log,
		metrics:  &obs.Metrics{},
		world:    st,
		keystore: ks,
		token:    token,
	}
	d.api = api.New(cfg.APIAddr, log, d.metrics, d.status, token, cfg.APIAllowPublic)
	return d, nil
}

// resolveWorld determines the node's World and, in personal mode with a keystore
// passphrase, unlocks (or first-time initializes) the keystore and derives the
// World from it. Without a keystore passphrase, personal mode falls back to the
// in-memory seed path (no persistence); relay and unconfigured modes are
// unchanged.
func resolveWorld(cfg config.Config, log *slog.Logger) (*world.State, *keystore.Keystore, error) {
	if cfg.Mode != config.ModePersonal || cfg.KeystorePassphrase == "" {
		st, err := world.Load(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("load world: %w", err)
		}
		return st, nil, nil
	}

	path := keystore.Path(cfg.DataDir)
	var (
		ks  *keystore.Keystore
		err error
	)
	switch {
	case keystore.Exists(path):
		if ks, err = keystore.Open(path, cfg.KeystorePassphrase); err != nil {
			return nil, nil, fmt.Errorf("open keystore: %w", err)
		}
		log.Info("keystore unlocked", "path", path)
	case cfg.WorldSeed != "":
		if ks, err = keystore.Init(path, cfg.KeystorePassphrase, keystore.Secrets{
			WorldSeed:          cfg.WorldSeed,
			IdentityPassphrase: cfg.IdentityPassphrase,
		}); err != nil {
			return nil, nil, fmt.Errorf("init keystore: %w", err)
		}
		log.Info("keystore initialized", "path", path)
	default:
		// Passphrase set but nothing to unlock or initialize: boot unconfigured;
		// a client bootstraps a World later (#38).
		log.Warn("no keystore and no seed to initialize one; booting unconfigured", "issue", 38)
		st, err := world.Load(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("load world: %w", err)
		}
		return st, nil, nil
	}
	return world.FromWorld(ks.World()), ks, nil
}

// Run starts the daemon and blocks until ctx is cancelled, then drains
// gracefully. It is safe to cancel ctx via signal handling in main.
func (d *Daemon) Run(ctx context.Context) error {
	d.start = time.Now()
	d.logStartup()

	// Open the local stores (#30): durable block store, in-memory index, blobs.
	st, err := store.Open(d.cfg.DataDir, d.log)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	d.store = st
	if n, err := st.Count(); err == nil {
		d.log.Info("storage open", "data_dir", d.cfg.DataDir, "blocks", n)
	}

	// The core controller owns the World/keystore/guard and the read/write
	// operations the client API exposes. It activates the World guard (#31) when
	// a World is loaded, or stays unconfigured until a client bootstraps one.
	c, err := core.New(d.cfg, Version, d.log, d.store, d.world, d.keystore)
	if err != nil {
		_ = d.store.Close()
		return fmt.Errorf("init core: %w", err)
	}
	d.core = c
	if d.world.Loaded {
		d.log.Info("world guard ready", "world", d.world.Fingerprint)
	} else {
		d.log.Warn("no World loaded; ingress validation unavailable until client bootstrap", "issue", 38)
	}

	// Mount the token-protected client API (#38) on the same server.
	path, handler := nodepbconnect.NewNodeServiceHandler(nodeapi.New(d.core))
	d.api.Handle(path, handler)

	// Start the libp2p host with mDNS local-network discovery (#32).
	ph, err := p2p.New(p2p.Config{DataDir: d.cfg.DataDir, ListenAddrs: d.cfg.P2PListen}, d.log)
	if err != nil {
		_ = d.store.Close()
		return fmt.Errorf("start p2p host: %w", err)
	}
	d.p2p = ph

	// Block exchange (#34): serve blocks to peers and fetch by ID, validating
	// every fetched block through the core's guard before storage.
	d.exchange = exchange.New(d.p2p.Host(), d.store, d.core.Guard, d.log)
	d.exchange.Start()
	d.log.Info("block exchange ready", "protocol", exchange.ProtocolID)

	// Gossip (#35): propagate blocks over per-audience gossipsub topics. The core
	// follows its World's audiences and publishes authored blocks through it.
	gsp, err := gossip.New(ctx, d.p2p.Host(), d.exchange, d.core.Guard, d.log)
	if err != nil {
		_ = d.store.Close()
		return fmt.Errorf("start gossip: %w", err)
	}
	d.gossip = gsp
	d.core.SetGossiper(gsp)
	d.log.Info("gossip ready", "audiences", gsp.FollowedCount())

	if err := d.api.Start(); err != nil {
		_ = d.store.Close()
		return fmt.Errorf("start api server: %w", err)
	}
	d.log.Info("bpnode ready", "api_addr", d.api.Addr(), "mode", d.cfg.Mode)

	<-ctx.Done()
	d.log.Info("shutting down", "reason", context.Cause(ctx))

	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	apiErr := d.api.Shutdown(shutCtx)
	if apiErr != nil {
		d.log.Error("api server shutdown", "err", apiErr)
	}
	if d.gossip != nil {
		_ = d.gossip.Close()
	}
	if d.exchange != nil {
		d.exchange.Stop()
	}
	if d.p2p != nil {
		if err := d.p2p.Close(); err != nil {
			d.log.Error("p2p close", "err", err)
		}
	}
	if err := d.store.Close(); err != nil {
		d.log.Error("store close", "err", err)
	}
	if d.core != nil {
		d.core.Lock()
		d.log.Debug("keystore locked")
	}
	if apiErr != nil {
		return apiErr
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
	s := api.Status{
		Version:     Version,
		Mode:        string(d.cfg.Mode),
		WorldLoaded: d.world.Loaded,
		World:       d.world.Fingerprint,
		Blocks:      -1,
		StartedAt:   d.start.UTC().Format(time.RFC3339),
		UptimeSec:   time.Since(d.start).Seconds(),
		Metrics:     d.metrics.Snapshot(),
	}
	// Once the core is up it is the source of truth (it can change at runtime via
	// client bootstrap).
	if d.core != nil {
		cs := d.core.Status()
		s.WorldLoaded = cs.WorldLoaded
		s.World = cs.World
		s.Identity = cs.Identity
		s.Blocks = cs.BlockCount
	}
	if d.p2p != nil {
		s.PeerID = d.p2p.ID()
		s.Peers = d.p2p.ConnectedCount()
	}
	return s
}
