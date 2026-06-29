package sync_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/bpprotocol/blockparty/implementations/go/audiences"
	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
	"github.com/bpprotocol/blockparty/node/internal/exchange"
	"github.com/bpprotocol/blockparty/node/internal/guard"
	"github.com/bpprotocol/blockparty/node/internal/p2p"
	"github.com/bpprotocol/blockparty/node/internal/store"
	syncpkg "github.com/bpprotocol/blockparty/node/internal/sync"
)

func discard() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func newHost(t *testing.T) *p2p.Host {
	t.Helper()
	h, err := p2p.New(p2p.Config{
		DataDir:     t.TempDir(),
		ListenAddrs: []string{"/ip4/127.0.0.1/tcp/0"},
		DisableMDNS: true,
	}, discard())
	if err != nil {
		t.Fatalf("p2p.New: %v", err)
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

// TestReconcilesMissingBlock is the #36 acceptance: node A heals a block it is
// missing by reconciling with peer B — answering "who has what" for the audience
// and fetching the gap — without any explicit, direct request for that block.
func TestReconcilesMissingBlock(t *testing.T) {
	w := derive.OpenWorld("sync-test-world")
	author := identity.OpenIdentity(w, "author")
	audHex := audiences.PublicAudience(w, 1).Code.Hex()

	// B holds two blocks for the audience; A holds neither.
	storeA, storeB := openStore(t), openStore(t)
	b1 := signedBlock(w, author, []byte("block one"))
	b2 := signedBlock(w, author, []byte("block two"))
	for _, b := range []*blockpb.Block{b1, b2} {
		if err := storeB.Put(&store.Record{Block: b}); err != nil {
			t.Fatalf("seed B: %v", err)
		}
	}

	hostA, hostB := newHost(t), newHost(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := hostA.Connect(ctx, peer.AddrInfo{ID: hostB.Host().ID(), Addrs: hostB.Host().Addrs()}); err != nil {
		t.Fatalf("connect: %v", err)
	}

	// B serves blocks + digests; A reconciles into its store through the guard.
	exB := exchange.New(hostB.Host(), storeB, func() *guard.Guard { return nil }, discard())
	exB.Start()
	t.Cleanup(exB.Stop)

	guardA := guard.New(w.SigningKey.Public, storeA, guard.WithLogger(discard()))
	exA := exchange.New(hostA.Host(), storeA, func() *guard.Guard { return guardA }, discard())

	rec := syncpkg.New(hostA, exA, storeA, func() []string { return []string{audHex} }, discard())

	// One automatic reconciliation pass — no direct fetch for either block.
	rec.ReconcileOnce(ctx)

	for _, b := range []*blockpb.Block{b1, b2} {
		if has, _ := storeA.Has(block.IDHex(b)); !has {
			t.Errorf("block %s was not reconciled onto A", block.IDHex(b)[:8])
		}
	}

	// A second pass is a no-op (nothing missing) — idempotent.
	if n := rec.ReconcileAudience(ctx, hostB.Host().ID(), audHex); n != 0 {
		t.Errorf("second reconcile fetched %d blocks, want 0", n)
	}
}
