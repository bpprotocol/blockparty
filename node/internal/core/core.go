// Package core is the node's stateful controller: it owns the (mutable) World,
// keystore, guard, and the read/write operations the client API (#38) exposes.
// World state is mutable because a client can bootstrap a World at runtime
// (#26 Q5), so all access is guarded by a RWMutex.
package core

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/cloudflare/circl/sign"

	"github.com/bpprotocol/blockparty/implementations/go/audiences"
	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/blocktypes"
	bpnode "github.com/bpprotocol/blockparty/implementations/go/node"
	"github.com/bpprotocol/blockparty/node/internal/config"
	"github.com/bpprotocol/blockparty/node/internal/guard"
	"github.com/bpprotocol/blockparty/node/internal/keystore"
	"github.com/bpprotocol/blockparty/node/internal/store"
	"github.com/bpprotocol/blockparty/node/internal/world"
)

// Sentinel errors returned by Core operations (mapped to RPC codes by the API).
var (
	ErrNoWorld        = errors.New("core: no World loaded")
	ErrAlreadyLoaded  = errors.New("core: a World is already loaded")
	ErrRelayBootstrap = errors.New("core: relay mode cannot bootstrap or author")
	ErrCannotAuthor   = errors.New("core: node cannot author (no unlocked keystore)")
)

// Per-peer ingress limits applied by the guard.
const (
	ingressRatePerSec = 50
	ingressBurst      = 100
)

// Status is a snapshot of the node's state.
type Status struct {
	Version     string
	Mode        string
	WorldLoaded bool
	World       string // fingerprint
	Identity    string // address
	BlockCount  int
	CanAuthor   bool
}

// Gossiper is the gossip layer (#35) the core drives: it follows audience topics
// and publishes authored blocks. Implemented by *gossip.Gossip; kept as an
// interface here to avoid an import cycle.
type Gossiper interface {
	Follow(audienceHex string) error
	Publish(audienceHex string, b *blockpb.Block) error
}

// Filter selects blocks for ListBlocks. Provide exactly one of the exact-match
// fields or a time range.
type Filter struct {
	Audience string
	Type     string
	Author   string
	From     int64
	To       int64
}

// Core holds the node's runtime state.
type Core struct {
	cfg     config.Config
	version string
	log     *slog.Logger
	store   *store.Store
	now     func() int64

	mu            sync.RWMutex
	world         *world.State
	ks            *keystore.Keystore
	guard         *guard.Guard
	gossiper      Gossiper
	resolver      *blocktypes.Resolver // personal mode, for reading plaintext
	publicSecrets map[string][]byte    // audience_code hex → secret (public audiences)
}

// New builds a Core seeded with the startup-resolved World/keystore (either may
// be nil/unloaded). It activates the guard when a World is loaded.
func New(cfg config.Config, version string, log *slog.Logger, st *store.Store, w *world.State, ks *keystore.Keystore) (*Core, error) {
	c := &Core{
		cfg:     cfg,
		version: version,
		log:     log,
		store:   st,
		now:     func() int64 { return time.Now().Unix() },
		world:   w,
		ks:      ks,
	}
	if w.Loaded {
		if err := c.activate(); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// activate builds the guard (and, in personal mode, the resolver + public
// audience secrets) for the currently-loaded World. Caller holds mu for writes.
func (c *Core) activate() error {
	pub, err := c.world.SigPublicKey()
	if err != nil {
		return fmt.Errorf("core: world public key: %w", err)
	}
	opts := []guard.Option{
		guard.WithRateLimiter(guard.NewRateLimiter(ingressRatePerSec, ingressBurst)),
		guard.WithLogger(c.log),
	}
	// In personal mode the node knows its own identity, so it can resolve and
	// verify the author of its own (and any self-authored) blocks.
	if c.ks != nil {
		id := c.ks.Identity()
		selfPub := id.MLDSA.Public
		selfAddr := string(id.Address)
		opts = append(opts, guard.WithAuthorResolver(func(b *blockpb.Block) (sign.PublicKey, string, bool) {
			if block.VerifyAuthor(b, selfPub) == nil {
				return selfPub, selfAddr, true
			}
			return nil, "", false
		}))

		w := c.ks.World()
		c.resolver = blocktypes.NewResolver(w)
		c.publicSecrets = make(map[string][]byte, audiences.ReservedPublicCount)
		for n := 1; n <= audiences.ReservedPublicCount; n++ {
			a := audiences.PublicAudience(w, n)
			c.publicSecrets[a.Code.Hex()] = a.Secret
		}
	}
	c.guard = guard.New(pub, c.store, opts...)
	c.followAudiencesLocked()
	return nil
}

// SetGossiper wires the gossip layer in. If a World is already loaded, it follows
// that World's audiences immediately. Called once at startup, after gossip is
// built (it depends on the host/exchange, which come up after the core).
func (c *Core) SetGossiper(g Gossiper) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gossiper = g
	c.followAudiencesLocked()
}

// followAudiencesLocked subscribes to the World's public audiences and the
// node's inbox. It needs the full World (personal mode) to derive the codes, so
// it is a no-op in relay mode or before the gossiper is wired. Caller holds mu.
func (c *Core) followAudiencesLocked() {
	if c.gossiper == nil || c.ks == nil {
		return
	}
	w := c.ks.World()
	for n := 1; n <= audiences.ReservedPublicCount; n++ {
		if err := c.gossiper.Follow(audiences.PublicAudience(w, n).Code.Hex()); err != nil {
			c.log.Debug("core: follow public audience failed", "n", n, "err", err)
		}
	}
	inbox := audiences.InboxAudience(w, c.ks.Identity().Address)
	if err := c.gossiper.Follow(inbox.Code.Hex()); err != nil {
		c.log.Debug("core: follow inbox failed", "err", err)
	}
}

// Guard returns the active ingress guard (nil until a World is loaded). Used by
// exchange/gossip (#34/#35) to validate incoming blocks.
func (c *Core) Guard() *guard.Guard {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.guard
}

// Status returns a snapshot of the node state.
func (c *Core) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.statusLocked()
}

func (c *Core) statusLocked() Status {
	n, _ := c.store.Count()
	s := Status{
		Version:     c.version,
		Mode:        string(c.cfg.Mode),
		WorldLoaded: c.world.Loaded,
		World:       c.world.Fingerprint,
		BlockCount:  n,
		CanAuthor:   c.ks != nil,
	}
	if c.ks != nil {
		s.Identity = string(c.ks.Identity().Address)
	}
	return s
}

// BootstrapWorld configures the node's World at runtime (personal mode, when
// none is loaded) by initializing the keystore from the supplied secrets.
func (c *Core) BootstrapWorld(seed, identityPass, keystorePass string) (Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cfg.Mode != config.ModePersonal {
		return Status{}, ErrRelayBootstrap
	}
	if c.world.Loaded {
		return Status{}, ErrAlreadyLoaded
	}
	if seed == "" {
		return Status{}, errors.New("core: world seed required")
	}
	if keystorePass == "" {
		return Status{}, errors.New("core: keystore passphrase required")
	}

	ks, err := keystore.Init(keystore.Path(c.cfg.DataDir), keystorePass, keystore.Secrets{
		WorldSeed:          seed,
		IdentityPassphrase: identityPass,
	})
	if err != nil {
		return Status{}, fmt.Errorf("core: init keystore: %w", err)
	}
	c.ks = ks
	c.world = world.FromWorld(ks.World())
	if err := c.activate(); err != nil {
		return Status{}, err
	}
	c.log.Info("world bootstrapped via client", "world", c.world.Fingerprint)
	return c.statusLocked(), nil
}

// PostText authors a content.post to public audience n (1..16), signs and
// encrypts it, and stores it through the guard. Returns the block ID.
func (c *Core) PostText(n int, text string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.world.Loaded {
		return "", ErrNoWorld
	}
	if c.ks == nil {
		if c.cfg.Mode == config.ModeRelay {
			return "", ErrRelayBootstrap
		}
		return "", ErrCannotAuthor
	}
	if n < 1 || n > audiences.ReservedPublicCount {
		return "", fmt.Errorf("core: public audience must be 1..%d", audiences.ReservedPublicCount)
	}

	w := c.ks.World()
	aud := audiences.PublicAudience(w, n)
	b, err := bpnode.BuildPost(w, c.ks.Identity(), aud.Code, aud.Secret, c.now(), text)
	if err != nil {
		return "", fmt.Errorf("core: build post: %w", err)
	}
	res, err := c.guard.IngestBlock("local", b)
	if err != nil {
		return "", fmt.Errorf("core: ingest: %w", err)
	}
	if res.Outcome == guard.Rejected {
		return "", fmt.Errorf("core: post rejected: %s", res.Reason)
	}
	// Announce to the audience's gossip topic so subscribers receive it.
	if c.gossiper != nil {
		if err := c.gossiper.Publish(aud.Code.Hex(), b); err != nil {
			c.log.Debug("core: gossip publish failed", "err", err)
		}
	}
	return res.ID, nil
}

// GetResult is a fetched block plus any decrypted content.
type GetResult struct {
	Record    *store.Record
	Decrypted bool
	Text      string
}

// GetBlock fetches a block by ID, decrypting the payload when the node can open
// the audience (public audience in personal mode). Returns store.ErrNotFound if
// absent.
func (c *Core) GetBlock(id string) (GetResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rec, err := c.store.Get(id)
	if err != nil {
		return GetResult{}, err
	}
	out := GetResult{Record: rec}
	if c.resolver != nil {
		if secret, ok := c.publicSecrets[hex.EncodeToString(rec.Block.AudienceCode)]; ok {
			var text string
			if opened, err := bpnode.OpenPost(c.ks.World(), c.resolver, rec.Block, secret, &text); err == nil && opened {
				out.Decrypted = true
				out.Text = text
			}
		}
	}
	return out, nil
}

// ListBlocks queries the index. Exactly one filter must be set.
func (c *Core) ListBlocks(f Filter) ([]*store.IndexEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch {
	case f.Audience != "":
		return c.store.ByAudience(f.Audience)
	case f.Type != "":
		return c.store.ByType(f.Type)
	case f.Author != "":
		return c.store.ByAuthor(f.Author)
	case f.From != 0 || f.To != 0:
		to := f.To
		if to == 0 {
			to = math.MaxInt64
		}
		return c.store.ByTimeRange(f.From, to)
	default:
		return nil, errors.New("core: a list filter is required")
	}
}

// Lock zeroizes secret material in the current keystore (shutdown).
func (c *Core) Lock() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ks != nil {
		c.ks.Lock()
	}
}
