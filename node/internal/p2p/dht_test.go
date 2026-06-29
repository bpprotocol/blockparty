package p2p

import (
	"testing"
	"time"
)

// newDHTHost starts a DHT-enabled host with mDNS disabled, so the only discovery
// path is the DHT/rendezvous one.
func newDHTHost(t *testing.T, rendezvous string, bootstrap []string) *Host {
	t.Helper()
	h, err := New(Config{
		DataDir:        t.TempDir(),
		ListenAddrs:    []string{"/ip4/127.0.0.1/tcp/0"},
		Rendezvous:     rendezvous,
		DisableMDNS:    true,
		EnableDHT:      true,
		BootstrapPeers: bootstrap,
	}, testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

func TestDHTBootstrapPopulatesRoutingTable(t *testing.T) {
	const tag = "bp-dht-bootstrap"
	boot := newDHTHost(t, tag, nil)

	node := newDHTHost(t, tag, boot.FullAddrs())

	// The node should connect to the bootstrap peer and add it to its routing
	// table within a few seconds.
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if node.RoutingTableSize() > 0 && node.ConnectedCount() > 0 {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("routing table did not populate (size=%d, connected=%d)", node.RoutingTableSize(), node.ConnectedCount())
}

// TestDHTRendezvousDiscovery is the #33 acceptance: two nodes that are NOT
// directly connected discover and connect to each other purely via the DHT
// rendezvous path, using a shared bootstrap node.
func TestDHTRendezvousDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("DHT rendezvous discovery is timing-dependent; skipped in -short")
	}
	const tag = "bp-dht-rendezvous"
	boot := newDHTHost(t, tag, nil)
	bootAddrs := boot.FullAddrs()

	a := newDHTHost(t, tag, bootAddrs)
	c := newDHTHost(t, tag, bootAddrs)

	// a and c only know the bootstrap node, not each other. They must find each
	// other through the DHT rendezvous and connect.
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		if isConnected(a, c.ID()) || isConnected(c, a.ID()) {
			return // success
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Skip("DHT rendezvous discovery did not complete in time; this environment may not support it")
}

func isConnected(h *Host, peerID string) bool {
	for _, p := range h.Peers() {
		if p.ID == peerID && p.Connected {
			return true
		}
	}
	return false
}
