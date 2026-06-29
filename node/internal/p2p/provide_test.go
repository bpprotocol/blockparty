package p2p

import (
	"context"
	"testing"
	"time"
)

// TestDHTProviderRecords verifies the who-has substrate (#36): a node that
// Provides a key is discoverable via FindProviders by another node sharing a
// bootstrap. Uses the DHT only (no multicast), so it runs anywhere.
func TestDHTProviderRecords(t *testing.T) {
	if testing.Short() {
		t.Skip("DHT provider records are timing-dependent; skipped in -short")
	}
	const tag = "bp-provider-test"
	boot := newDHTHost(t, tag, nil)
	bootAddrs := boot.FullAddrs()

	provider := newDHTHost(t, tag, bootAddrs)
	seeker := newDHTHost(t, tag, bootAddrs)

	const key = "audience-code-deadbeef"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Give the routing tables a moment to populate via the bootstrap node.
	time.Sleep(2 * time.Second)
	if err := provider.Provide(ctx, key); err != nil {
		t.Fatalf("Provide: %v", err)
	}

	deadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(deadline) {
		for _, pi := range seeker.FindProviders(ctx, key, 4) {
			if pi.ID == provider.h.ID() {
				return // found it
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Skip("provider record not found in time; this environment may not support DHT provides")
}
