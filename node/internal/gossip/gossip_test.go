package gossip

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/bpprotocol/blockparty/implementations/go/audiences"
	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
	"github.com/bpprotocol/blockparty/node/internal/exchange"
	"github.com/bpprotocol/blockparty/node/internal/guard"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

const worldPhrase = "gossip-test-world"

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newHost(t *testing.T) host.Host {
	t.Helper()
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("libp2p.New: %v", err)
	}
	t.Cleanup(func() { h.Close() })
	return h
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), discard())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func signedBlock(w derive.World, a identity.Identity, data []byte) *blockpb.Block {
	b := block.New(a.Address,
		derive.GetTypeCode(w, "bpprotocol.org/v1/types/content.post"),
		derive.GetAudienceCode(w, audiences.PublicAudienceID(1)),
		1700000000, data)
	block.Sign(b, w, a.MLDSA)
	return b
}

func connect(t *testing.T, a, b host.Host) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Connect(ctx, peer.AddrInfo{ID: b.ID(), Addrs: b.Addrs()}); err != nil {
		t.Fatalf("connect: %v", err)
	}
}

// waitForBlock polls until st has id, or fails after the timeout.
func waitForBlock(t *testing.T, st *store.Store, id string, why string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if has, _ := st.Has(id); has {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("block did not propagate (%s)", why)
}

// TestGossipPropagatesInlinedBlock is the #35 acceptance: a block published on an
// audience by node A reaches subscribed node B, validates, and is stored.
func TestGossipPropagatesInlinedBlock(t *testing.T) {
	w := derive.OpenWorld(worldPhrase)
	author := identity.OpenIdentity(w, "author")
	audHex := audiences.PublicAudience(w, 1).Code.Hex()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hostA, hostB := newHost(t), newHost(t)
	connect(t, hostA, hostB)

	storeB := openStore(t)
	guardB := guard.New(w.SigningKey.Public, storeB, guard.WithLogger(discard()))
	gsB, err := New(ctx, hostB, exchange.New(hostB, storeB, func() *guard.Guard { return guardB }, discard()), func() *guard.Guard { return guardB }, discard())
	if err != nil {
		t.Fatalf("gossip B: %v", err)
	}
	defer gsB.Close()
	if err := gsB.Follow(audHex); err != nil {
		t.Fatalf("B follow: %v", err)
	}

	storeA := openStore(t)
	gsA, err := New(ctx, hostA, exchange.New(hostA, storeA, func() *guard.Guard { return nil }, discard()), func() *guard.Guard { return nil }, discard())
	if err != nil {
		t.Fatalf("gossip A: %v", err)
	}
	defer gsA.Close()
	if err := gsA.Follow(audHex); err != nil {
		t.Fatalf("A follow: %v", err)
	}

	// Let the gossipsub mesh form before publishing.
	time.Sleep(1500 * time.Millisecond)

	b := signedBlock(w, author, []byte("hello gossip"))
	if err := gsA.Publish(audHex, b); err != nil {
		t.Fatalf("publish: %v", err)
	}

	waitForBlock(t, storeB, block.IDHex(b), "inlined")
}

// TestGossipIDOnlyTriggersExchangeFetch covers the large-block path: when only
// the ID is gossiped, the receiver pulls the full block via the exchange.
func TestGossipIDOnlyTriggersExchangeFetch(t *testing.T) {
	w := derive.OpenWorld(worldPhrase)
	author := identity.OpenIdentity(w, "author")
	audHex := audiences.PublicAudience(w, 1).Code.Hex()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hostA, hostB := newHost(t), newHost(t)
	connect(t, hostA, hostB)

	// A serves blocks over the exchange and has the block in its store.
	storeA := openStore(t)
	b := signedBlock(w, author, []byte("fetch me"))
	if err := storeA.Put(&store.Record{Block: b}); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	exA := exchange.New(hostA, storeA, func() *guard.Guard { return nil }, discard())
	exA.Start()
	t.Cleanup(exA.Stop)
	gsA, err := New(ctx, hostA, exA, func() *guard.Guard { return nil }, discard())
	if err != nil {
		t.Fatalf("gossip A: %v", err)
	}
	defer gsA.Close()
	gsA.inlineMax = 0 // force ID-only announcements
	if err := gsA.Follow(audHex); err != nil {
		t.Fatalf("A follow: %v", err)
	}

	// B receives the ID and must fetch the block from A via the exchange.
	storeB := openStore(t)
	guardB := guard.New(w.SigningKey.Public, storeB, guard.WithLogger(discard()))
	exB := exchange.New(hostB, storeB, func() *guard.Guard { return guardB }, discard())
	gsB, err := New(ctx, hostB, exB, func() *guard.Guard { return guardB }, discard())
	if err != nil {
		t.Fatalf("gossip B: %v", err)
	}
	defer gsB.Close()
	if err := gsB.Follow(audHex); err != nil {
		t.Fatalf("B follow: %v", err)
	}

	time.Sleep(1500 * time.Millisecond)
	if err := gsA.Publish(audHex, b); err != nil {
		t.Fatalf("publish: %v", err)
	}
	waitForBlock(t, storeB, block.IDHex(b), "id-only + exchange fetch")
}
