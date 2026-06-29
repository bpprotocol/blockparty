// Package p2p stands up the node's libp2p host and local-network (mDNS) peer
// discovery (#32). It is the transport foundation the block-exchange (#34),
// gossip (#35), and DHT discovery (#33) layers build on.
//
// The host identity (its libp2p peer ID) is a persistent Ed25519 key in the data
// dir — distinct from the BlockParty World/identity keys. mDNS discovery is
// scoped by a service tag so only nodes sharing it (the same World, when one is
// loaded) auto-connect on the LAN.
package p2p

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

const (
	keyFile        = "p2p.key"
	connectTimeout = 10 * time.Second

	// DefaultRendezvous is the mDNS service tag shared by all BlockParty nodes.
	// Discovery is World-agnostic at this layer; World membership is enforced on
	// the blocks themselves by the guard (#31).
	DefaultRendezvous = "blockparty-mdns"
)

// DefaultListenAddrs is used when no listen multiaddrs are configured.
var DefaultListenAddrs = []string{"/ip4/0.0.0.0/tcp/0"}

// Config configures the host.
type Config struct {
	DataDir     string
	ListenAddrs []string // libp2p multiaddrs; defaults to DefaultListenAddrs
	Rendezvous  string   // mDNS service tag (same tag → mutual discovery)
}

// PeerInfo is a discovered peer's status, for the client API.
type PeerInfo struct {
	ID        string `json:"id"`
	Connected bool   `json:"connected"`
}

// Host wraps a libp2p host plus mDNS discovery and a tracker of discovered peers.
type Host struct {
	h    host.Host
	mdns mdns.Service
	log  *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	mu   sync.Mutex
	seen map[peer.ID]time.Time
}

// New builds and starts the host: it binds the listen addresses and begins mDNS
// discovery on the configured rendezvous tag.
func New(cfg Config, log *slog.Logger) (*Host, error) {
	key, err := loadOrCreateKey(filepath.Join(cfg.DataDir, keyFile))
	if err != nil {
		return nil, fmt.Errorf("p2p: host key: %w", err)
	}
	listen := cfg.ListenAddrs
	if len(listen) == 0 {
		listen = DefaultListenAddrs
	}
	rendezvous := cfg.Rendezvous
	if rendezvous == "" {
		rendezvous = DefaultRendezvous
	}
	h, err := libp2p.New(libp2p.Identity(key), libp2p.ListenAddrStrings(listen...))
	if err != nil {
		return nil, fmt.Errorf("p2p: new host: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	ph := &Host{
		h:      h,
		log:    log,
		ctx:    ctx,
		cancel: cancel,
		seen:   make(map[peer.ID]time.Time),
	}

	svc := mdns.NewMdnsService(h, rendezvous, ph)
	if err := svc.Start(); err != nil {
		cancel()
		_ = h.Close()
		return nil, fmt.Errorf("p2p: start mdns: %w", err)
	}
	ph.mdns = svc

	log.Info("p2p host started", "id", h.ID().String(), "rendezvous", rendezvous, "addrs", ph.Addrs())
	return ph, nil
}

// HandlePeerFound implements mdns.Notifee: it records and dials a discovered peer.
func (ph *Host) HandlePeerFound(pi peer.AddrInfo) {
	if pi.ID == ph.h.ID() {
		return
	}
	ph.mu.Lock()
	ph.seen[pi.ID] = time.Now()
	ph.mu.Unlock()

	ctx, cancel := context.WithTimeout(ph.ctx, connectTimeout)
	defer cancel()
	if err := ph.h.Connect(ctx, pi); err != nil {
		ph.log.Debug("p2p: mdns dial failed", "peer", pi.ID.String(), "err", err)
		return
	}
	ph.log.Info("p2p: connected to mDNS peer", "peer", pi.ID.String())
}

// ID returns the host's peer ID.
func (ph *Host) ID() string { return ph.h.ID().String() }

// Addrs returns the host's listen multiaddrs as strings.
func (ph *Host) Addrs() []string {
	addrs := ph.h.Addrs()
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.String())
	}
	return out
}

// Peers returns the discovered peers and whether each is currently connected.
func (ph *Host) Peers() []PeerInfo {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	out := make([]PeerInfo, 0, len(ph.seen))
	for id := range ph.seen {
		out = append(out, PeerInfo{
			ID:        id.String(),
			Connected: ph.h.Network().Connectedness(id) == network.Connected,
		})
	}
	return out
}

// ConnectedCount returns the number of currently-connected peers.
func (ph *Host) ConnectedCount() int {
	return len(ph.h.Network().Peers())
}

// Host exposes the underlying libp2p host for the exchange/gossip layers.
func (ph *Host) Host() host.Host { return ph.h }

// Close stops discovery and shuts the host down.
func (ph *Host) Close() error {
	ph.cancel()
	if ph.mdns != nil {
		_ = ph.mdns.Close()
	}
	return ph.h.Close()
}

func loadOrCreateKey(path string) (crypto.PrivKey, error) {
	if b, err := os.ReadFile(path); err == nil {
		return crypto.UnmarshalPrivateKey(b)
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	b, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		return nil, err
	}
	return priv, nil
}
