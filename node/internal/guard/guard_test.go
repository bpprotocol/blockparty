package guard

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/cloudflare/circl/sign"

	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/implementations/go/identity"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

const (
	worldPhrase = "guard-test-world"
	typeURN     = "bpprotocol.org/v1/types/content.post"
	audienceURN = "bpprotocol.org/v1/audience/public-1"
)

type harness struct {
	world  derive.World
	author identity.Identity
	store  *store.Store
}

func newHarness(t *testing.T) harness {
	t.Helper()
	w := derive.OpenWorld(worldPhrase)
	st, err := store.Open(t.TempDir(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return harness{world: w, author: identity.OpenIdentity(w, "author-pass"), store: st}
}

// signedBlock builds a valid block signed by the harness World + author.
func (h harness) signedBlock(data []byte) *blockpb.Block {
	b := block.New(
		h.author.Address,
		derive.GetTypeCode(h.world, typeURN),
		derive.GetAudienceCode(h.world, audienceURN),
		1700000000,
		data,
	)
	block.Sign(b, h.world, h.author.MLDSA)
	return b
}

func (h harness) guard(opts ...Option) *Guard {
	base := []Option{WithClock(func() int64 { return 42 }), WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))}
	return New(h.world.SigningKey.Public, h.store, append(base, opts...)...)
}

func TestAcceptsValidBlockAndStores(t *testing.T) {
	h := newHarness(t)
	g := h.guard()
	b := h.signedBlock([]byte("hello"))

	res, err := g.IngestBlock("peerA", b)
	if err != nil {
		t.Fatalf("IngestBlock: %v", err)
	}
	if res.Outcome != Accepted {
		t.Fatalf("outcome = %s (%s), want accepted", res.Outcome, res.Reason)
	}
	if has, _ := h.store.Has(block.IDHex(b)); !has {
		t.Error("accepted block was not stored")
	}
}

func TestRejectsForeignWorld(t *testing.T) {
	h := newHarness(t)
	g := h.guard()

	// Block signed by a different World.
	other := derive.OpenWorld("some-other-world")
	otherAuthor := identity.OpenIdentity(other, "x")
	b := block.New(otherAuthor.Address, derive.GetTypeCode(other, typeURN),
		derive.GetAudienceCode(other, audienceURN), 1700000000, []byte("intruder"))
	block.Sign(b, other, otherAuthor.MLDSA)

	res, _ := g.IngestBlock("peerA", b)
	if res.Outcome != Rejected {
		t.Fatalf("outcome = %s, want rejected", res.Outcome)
	}
	if has, _ := h.store.Has(block.IDHex(b)); has {
		t.Error("a foreign-World block must not be stored")
	}
}

func TestRejectsTamperedData(t *testing.T) {
	h := newHarness(t)
	g := h.guard()
	b := h.signedBlock([]byte("original"))
	b.Data = []byte("tampered after signing")

	res, _ := g.IngestBlock("peerA", b)
	if res.Outcome != Rejected {
		t.Errorf("outcome = %s, want rejected for tampered data", res.Outcome)
	}
}

func TestDedupe(t *testing.T) {
	h := newHarness(t)
	g := h.guard()
	b := h.signedBlock([]byte("once"))

	if res, _ := g.IngestBlock("peerA", b); res.Outcome != Accepted {
		t.Fatalf("first ingest = %s, want accepted", res.Outcome)
	}
	res, _ := g.IngestBlock("peerA", b)
	if res.Outcome != Duplicate {
		t.Errorf("second ingest = %s, want duplicate", res.Outcome)
	}
}

func TestAuthorResolver(t *testing.T) {
	h := newHarness(t)
	b := h.signedBlock([]byte("authored"))

	// Resolver returns the correct author key → author_sig verified, address set.
	resolver := func(*blockpb.Block) (sign.PublicKey, string, bool) {
		return h.author.MLDSA.Public, string(h.author.Address), true
	}
	res, err := h.guard(WithAuthorResolver(resolver)).IngestBlock("p", b)
	if err != nil {
		t.Fatalf("IngestBlock: %v", err)
	}
	if res.Outcome != Accepted || !res.AuthorVerified {
		t.Fatalf("outcome=%s authorVerified=%v, want accepted+verified", res.Outcome, res.AuthorVerified)
	}
	if res.Author != string(h.author.Address) {
		t.Errorf("author = %q, want %q", res.Author, h.author.Address)
	}
	rec, _ := h.store.Get(block.IDHex(b))
	if rec.Author != string(h.author.Address) {
		t.Errorf("stored author = %q, want %q", rec.Author, h.author.Address)
	}
}

func TestAuthorResolverWrongKeyRejects(t *testing.T) {
	h := newHarness(t)
	b := h.signedBlock([]byte("authored"))
	stranger := identity.OpenIdentity(h.world, "stranger")

	resolver := func(*blockpb.Block) (sign.PublicKey, string, bool) {
		return stranger.MLDSA.Public, string(stranger.Address), true
	}
	res, _ := h.guard(WithAuthorResolver(resolver)).IngestBlock("p", b)
	if res.Outcome != Rejected {
		t.Errorf("outcome = %s, want rejected for bad author signature", res.Outcome)
	}
}

func TestUnknownAuthorAcceptedUnverified(t *testing.T) {
	h := newHarness(t)
	b := h.signedBlock([]byte("authored"))
	// Resolver that never knows the author.
	resolver := func(*blockpb.Block) (sign.PublicKey, string, bool) { return nil, "", false }

	res, _ := h.guard(WithAuthorResolver(resolver)).IngestBlock("p", b)
	if res.Outcome != Accepted || res.AuthorVerified {
		t.Errorf("outcome=%s authorVerified=%v, want accepted+unverified", res.Outcome, res.AuthorVerified)
	}
}

func TestRejectsGarbageBytes(t *testing.T) {
	h := newHarness(t)
	res, _ := h.guard().IngestBytes("p", []byte{0xff, 0x00, 0x12, 0x34})
	if res.Outcome != Rejected {
		t.Errorf("outcome = %s, want rejected for undecodable bytes", res.Outcome)
	}
}

func TestRateLimit(t *testing.T) {
	h := newHarness(t)
	// burst of 1, frozen clock → the second call from the same peer is limited.
	rl := NewRateLimiter(1, 1)
	frozen := time.Unix(1700000000, 0)
	rl.now = func() time.Time { return frozen }
	g := h.guard(WithRateLimiter(rl))

	if res, _ := g.IngestBlock("peerA", h.signedBlock([]byte("1"))); res.Outcome != Accepted {
		t.Fatalf("first = %s, want accepted", res.Outcome)
	}
	res, _ := g.IngestBlock("peerA", h.signedBlock([]byte("2")))
	if res.Outcome != Rejected || res.Reason != "rate limited" {
		t.Errorf("second = %s/%q, want rejected/rate limited", res.Outcome, res.Reason)
	}
	// A different peer has its own bucket.
	if res, _ := g.IngestBlock("peerB", h.signedBlock([]byte("3"))); res.Outcome != Accepted {
		t.Errorf("peerB = %s, want accepted (separate bucket)", res.Outcome)
	}
}
