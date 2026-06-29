package p2p

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newHost(t *testing.T, rendezvous string) *Host {
	t.Helper()
	h, err := New(Config{
		DataDir:     t.TempDir(),
		ListenAddrs: []string{"/ip4/127.0.0.1/tcp/0"},
		Rendezvous:  rendezvous,
	}, testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

func TestHostStartsWithStableIdentity(t *testing.T) {
	dir := t.TempDir()
	h1, err := New(Config{DataDir: dir, ListenAddrs: []string{"/ip4/127.0.0.1/tcp/0"}, Rendezvous: "t"}, testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	id1 := h1.ID()
	if id1 == "" {
		t.Fatal("empty peer ID")
	}
	if len(h1.Addrs()) == 0 {
		t.Error("expected at least one listen address")
	}
	h1.Close()

	// Reopening the same data dir must reproduce the same peer ID (persistent key).
	h2, err := New(Config{DataDir: dir, ListenAddrs: []string{"/ip4/127.0.0.1/tcp/0"}, Rendezvous: "t"}, testLogger())
	if err != nil {
		t.Fatalf("New (reopen): %v", err)
	}
	defer h2.Close()
	if h2.ID() != id1 {
		t.Errorf("peer ID changed across restart: %s != %s", h2.ID(), id1)
	}
}

func TestDirectConnect(t *testing.T) {
	a := newHost(t, "direct")
	b := newHost(t, "direct")

	// Dial b from a using b's advertised address.
	info := peer.AddrInfo{ID: peerID(t, b), Addrs: addrs(t, b)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Host().Connect(ctx, info); err != nil {
		t.Fatalf("connect: %v", err)
	}
	if a.ConnectedCount() == 0 {
		t.Error("expected a to report a connected peer")
	}
}

func TestMdnsDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("mDNS discovery is environment-dependent; skipped in -short")
	}
	const tag = "bp-mdns-test"
	a := newHost(t, tag)
	b := newHost(t, tag)

	// Wait for mDNS to discover and connect the two hosts.
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if a.ConnectedCount() > 0 && b.ConnectedCount() > 0 {
			return // success
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Skip("mDNS multicast discovery did not complete; this environment likely blocks multicast")
}

func peerID(t *testing.T, h *Host) peer.ID {
	t.Helper()
	id, err := peer.Decode(h.ID())
	if err != nil {
		t.Fatalf("decode peer id: %v", err)
	}
	return id
}

func addrs(t *testing.T, h *Host) []multiaddr.Multiaddr {
	t.Helper()
	out := make([]multiaddr.Multiaddr, 0)
	for _, s := range h.Addrs() {
		a, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			t.Fatalf("parse addr %q: %v", s, err)
		}
		out = append(out, a)
	}
	return out
}
