// Package guard is the node's ingress validation pipeline (#31). Every block that
// enters the node — from exchange (#34), gossip (#35), or the client API (#38) —
// passes through it before it can be stored:
//
//	decode → rate-limit → World membership → dedupe → author (best-effort) → store
//
// World membership (world_sig verified against the configured World public key)
// is the mandatory cryptographic gate; it works in both personal and relay mode
// because it needs only the public key. Because world_sig covers every envelope
// field including the payload, this single check also rejects any tampering.
package guard

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/cloudflare/circl/sign"

	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

// Outcome is the disposition of an ingested block.
type Outcome int

const (
	// Accepted means the block passed validation and was stored.
	Accepted Outcome = iota
	// Duplicate means the block was already stored (no-op).
	Duplicate
	// Rejected means the block failed validation (see Result.Reason).
	Rejected
)

func (o Outcome) String() string {
	switch o {
	case Accepted:
		return "accepted"
	case Duplicate:
		return "duplicate"
	case Rejected:
		return "rejected"
	default:
		return "unknown"
	}
}

// Result describes what the guard did with a block.
type Result struct {
	Outcome        Outcome
	Reason         string // populated when Rejected
	ID             string // block ID hex (when decodable)
	Author         string // resolved author address ("" if unresolved)
	AuthorVerified bool   // author_sig was checked and valid
	CoSignatures   int    // number of co-signatures present (surfaced, not gating)
}

// AuthorResolver maps a decoded block to its author's verifying key and address,
// when the node knows them (e.g. from a stored identity block). ok=false means
// the author is unknown, in which case author_sig is left unverified and the
// block is still accepted on world_sig alone. Wired in by later issues (#16/#18).
type AuthorResolver func(b *blockpb.Block) (pub sign.PublicKey, address string, ok bool)

// AcceptHook is called (asynchronously) for each newly-accepted block, so layers
// like connection handling (#37) can react to relevant block types.
type AcceptHook func(*blockpb.Block)

// Guard validates and stores ingress blocks for a single World.
type Guard struct {
	worldPub    sign.PublicKey
	store       *store.Store
	resolver    AuthorResolver
	limiter     *RateLimiter
	now         func() int64
	log         *slog.Logger
	acceptHooks []AcceptHook
}

// Option configures a Guard.
type Option func(*Guard)

// WithAuthorResolver enables best-effort author_sig verification.
func WithAuthorResolver(r AuthorResolver) Option { return func(g *Guard) { g.resolver = r } }

// WithRateLimiter bounds work per source peer.
func WithRateLimiter(l *RateLimiter) Option { return func(g *Guard) { g.limiter = l } }

// WithClock overrides the ReceivedAt clock (for tests).
func WithClock(now func() int64) Option { return func(g *Guard) { g.now = now } }

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option { return func(g *Guard) { g.log = l } }

// WithAcceptHook registers a callback fired (in a goroutine) for each block that
// is newly accepted and stored.
func WithAcceptHook(h AcceptHook) Option {
	return func(g *Guard) { g.acceptHooks = append(g.acceptHooks, h) }
}

// New builds a Guard validating against worldPub and storing into st.
func New(worldPub sign.PublicKey, st *store.Store, opts ...Option) *Guard {
	g := &Guard{
		worldPub: worldPub,
		store:    st,
		now:      func() int64 { return time.Now().Unix() },
		log:      slog.Default(),
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

// IngestBytes decodes raw wire bytes from peer and runs the pipeline.
func (g *Guard) IngestBytes(peer string, raw []byte) (Result, error) {
	if g.limiter != nil && !g.limiter.Allow(peer) {
		return rejected("rate limited", ""), nil
	}
	b, err := block.Decode(raw)
	if err != nil {
		return rejected("decode: "+err.Error(), ""), nil
	}
	return g.validateAndStore(b)
}

// IngestBlock runs the pipeline on an already-decoded block from peer.
func (g *Guard) IngestBlock(peer string, b *blockpb.Block) (Result, error) {
	if g.limiter != nil && !g.limiter.Allow(peer) {
		return rejected("rate limited", ""), nil
	}
	return g.validateAndStore(b)
}

func (g *Guard) validateAndStore(b *blockpb.Block) (Result, error) {
	id := block.IDHex(b)
	if id == "" {
		return rejected("missing block id", ""), nil
	}

	// World membership — the mandatory gate (also catches any field tampering).
	if err := block.VerifyWorld(b, g.worldPub); err != nil {
		return rejected("world signature: "+err.Error(), id), nil
	}

	// Dedupe.
	has, err := g.store.Has(id)
	if err != nil {
		return Result{}, fmt.Errorf("guard: dedupe check %s: %w", id, err)
	}
	if has {
		return Result{Outcome: Duplicate, ID: id}, nil
	}

	// Author signature — best-effort, only when the author is resolvable.
	author := ""
	authorVerified := false
	if g.resolver != nil {
		if pub, addr, ok := g.resolver(b); ok {
			if err := block.VerifyAuthor(b, pub); err != nil {
				return rejected("author signature: "+err.Error(), id), nil
			}
			author = addr
			authorVerified = true
		}
	}

	// Co-signatures are surfaced but never gate (block.md §Co-Signatures).
	coSigs := len(block.CoSignatures(b))

	if err := g.store.Put(&store.Record{Block: b, Author: author, ReceivedAt: g.now()}); err != nil {
		return Result{}, fmt.Errorf("guard: store %s: %w", id, err)
	}

	g.log.Debug("block accepted", "id", id, "author_verified", authorVerified, "cosigs", coSigs)
	for _, h := range g.acceptHooks {
		go h(b)
	}
	return Result{
		Outcome:        Accepted,
		ID:             id,
		Author:         author,
		AuthorVerified: authorVerified,
		CoSignatures:   coSigs,
	}, nil
}

func rejected(reason, id string) Result {
	return Result{Outcome: Rejected, Reason: reason, ID: id}
}
