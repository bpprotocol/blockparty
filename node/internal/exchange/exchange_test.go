package exchange

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
	"github.com/bpprotocol/blockparty/node/internal/guard"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

const worldPhrase = "exchange-test-world"

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

// signedBlock builds and signs a block for World w.
func signedBlock(w derive.World, author identity.Identity, data []byte) *blockpb.Block {
	b := block.New(author.Address,
		derive.GetTypeCode(w, "bpprotocol.org/v1/types/content.post"),
		derive.GetAudienceCode(w, "bpprotocol.org/v1/audience/public-1"),
		1700000000, data)
	block.Sign(b, w, author.MLDSA)
	return b
}

// connect dials b from a and waits for the connection.
func connect(t *testing.T, a, b host.Host) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.Connect(ctx, peer.AddrInfo{ID: b.ID(), Addrs: b.Addrs()}); err != nil {
		t.Fatalf("connect: %v", err)
	}
}

func TestFetchByID(t *testing.T) {
	w := derive.OpenWorld(worldPhrase)
	author := identity.OpenIdentity(w, "author")

	// B has the block; A wants it.
	storeA, storeB := openStore(t), openStore(t)
	b := signedBlock(w, author, []byte("payload"))
	if err := storeB.Put(&store.Record{Block: b}); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	hostA, hostB := newHost(t), newHost(t)
	connect(t, hostA, hostB)

	guardA := guard.New(w.SigningKey.Public, storeA, guard.WithLogger(discard()))
	exA := New(hostA, storeA, func() *guard.Guard { return guardA }, discard())
	exB := New(hostB, storeB, func() *guard.Guard { return nil }, discard())
	exB.Start()
	t.Cleanup(exB.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := exA.Fetch(ctx, hostB.ID(), [][]byte{b.Id})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if res.Accepted != 1 || res.Received != 1 {
		t.Fatalf("fetch result = %+v, want received=1 accepted=1", res)
	}
	if has, _ := storeA.Has(block.IDHex(b)); !has {
		t.Error("fetched block was not stored on A")
	}

	// Fetching again is a no-op dedupe.
	res2, _ := exA.Fetch(ctx, hostB.ID(), [][]byte{b.Id})
	if res2.Duplicate != 1 || res2.Accepted != 0 {
		t.Errorf("second fetch = %+v, want duplicate=1 accepted=0", res2)
	}
}

func TestHaveQuery(t *testing.T) {
	w := derive.OpenWorld(worldPhrase)
	author := identity.OpenIdentity(w, "author")
	storeB := openStore(t)
	have := signedBlock(w, author, []byte("present"))
	missing := signedBlock(w, author, []byte("absent"))
	if err := storeB.Put(&store.Record{Block: have}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	hostA, hostB := newHost(t), newHost(t)
	connect(t, hostA, hostB)
	exB := New(hostB, storeB, func() *guard.Guard { return nil }, discard())
	exB.Start()
	t.Cleanup(exB.Stop)
	exA := New(hostA, openStore(t), func() *guard.Guard { return nil }, discard())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	got, err := exA.Have(ctx, hostB.ID(), [][]byte{have.Id, missing.Id})
	if err != nil {
		t.Fatalf("Have: %v", err)
	}
	if len(got) != 1 || string(got[0]) != string(have.Id) {
		t.Errorf("Have = %d ids, want exactly the present one", len(got))
	}
}

func TestFetchRejectsForeignWorld(t *testing.T) {
	// B serves a block from a DIFFERENT world; A's guard must reject it.
	other := derive.OpenWorld("a-different-world")
	otherAuthor := identity.OpenIdentity(other, "x")
	foreign := signedBlock(other, otherAuthor, []byte("intruder"))

	storeA, storeB := openStore(t), openStore(t)
	if err := storeB.Put(&store.Record{Block: foreign}); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	hostA, hostB := newHost(t), newHost(t)
	connect(t, hostA, hostB)

	myWorld := derive.OpenWorld(worldPhrase)
	guardA := guard.New(myWorld.SigningKey.Public, storeA, guard.WithLogger(discard()))
	exA := New(hostA, storeA, func() *guard.Guard { return guardA }, discard())
	exB := New(hostB, storeB, func() *guard.Guard { return nil }, discard())
	exB.Start()
	t.Cleanup(exB.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := exA.Fetch(ctx, hostB.ID(), [][]byte{foreign.Id})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if res.Accepted != 0 {
		t.Errorf("accepted = %d, want 0 (foreign-World block must be rejected)", res.Accepted)
	}
	if has, _ := storeA.Has(block.IDHex(foreign)); has {
		t.Error("foreign-World block must not be stored")
	}
}
