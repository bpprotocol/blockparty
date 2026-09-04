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
	"github.com/bpprotocol/blockparty/implementations/go/crypto"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	bpnode "github.com/bpprotocol/blockparty/implementations/go/node"
	"github.com/bpprotocol/blockparty/node/internal/config"
	"github.com/bpprotocol/blockparty/node/internal/connections"
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
	ErrNoKeystore     = errors.New("core: no keystore to unlock")
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
	// KeystoreExists reports an encrypted keystore on disk. With WorldLoaded
	// false it tells a client to ask for the passphrase and unlock, rather than
	// offer onboarding (which would fail: a keystore is never overwritten).
	KeystoreExists bool
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
	conns         *connections.Manager // personal mode connection handshakes (#37)
	resolver      *blocktypes.Resolver // personal mode, for reading plaintext
	publicSecrets map[string][]byte    // audience_code hex → secret (public audiences)
	publicNums    map[string]int       // audience_code hex → public-N (1..16), for the feed (#44)

	feedMu     sync.Mutex
	feeds      map[int]*feedSub // live block subscribers (#44)
	nextFeedID int
}

// FeedEvent is a live notification that a block was accepted onto an audience.
type FeedEvent struct {
	ID         string
	Audience   string
	Type       string
	Author     string
	Timestamp  int64
	ReceivedAt int64
}

type feedSub struct {
	audience string // "" = all audiences
	ch       chan FeedEvent
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
		feeds:   make(map[int]*feedSub),
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
		c.publicNums = make(map[string]int, audiences.ReservedPublicCount)
		for n := 1; n <= audiences.ReservedPublicCount; n++ {
			a := audiences.PublicAudience(w, n)
			c.publicSecrets[a.Code.Hex()] = a.Secret
			c.publicNums[a.Code.Hex()] = n
		}

		// Build the connection manager once the gossip transport is wired (#37).
		if c.gossiper != nil && c.conns == nil {
			c.conns = connections.New(c.ks, c.gossiper, c.log)
		}
	}
	// The connection manager reacts to handshake blocks as they are accepted.
	if c.conns != nil {
		opts = append(opts, guard.WithAcceptHook(c.conns.OnBlock))
	}
	// Fan accepted blocks out to live feed subscribers (#44).
	opts = append(opts, guard.WithAcceptHook(c.onAccepted))
	c.guard = guard.New(pub, c.store, opts...)
	c.followAudiencesLocked()
	return nil
}

// SetGossiper wires the gossip layer in. If a World is already loaded, it
// (re)activates so the guard picks up the connection manager and the node
// follows its audiences. Called once at startup, after gossip is built.
func (c *Core) SetGossiper(g Gossiper) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gossiper = g
	if c.world.Loaded {
		if err := c.activate(); err != nil {
			c.log.Error("core: re-activate after gossiper failed", "err", err)
		}
	}
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
		Version:        c.version,
		Mode:           string(c.cfg.Mode),
		WorldLoaded:    c.world.Loaded,
		World:          c.world.Fingerprint,
		BlockCount:     n,
		CanAuthor:      c.ks != nil,
		KeystoreExists: keystore.Exists(keystore.Path(c.cfg.DataDir)),
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

// UnlockKeystore opens the keystore already on disk and loads its World —
// the runtime counterpart of unlocking at boot with BPNODE_KEYSTORE_PASSPHRASE
// (#28). A node started without that passphrase boots unconfigured, so the
// client collects it and calls this instead of BootstrapWorld (which refuses to
// overwrite an existing keystore).
//
// The passphrase is used to derive the master key and is not retained; private
// keys stay in memory, as they do on the boot path.
func (c *Core) UnlockKeystore(keystorePass string) (Status, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cfg.Mode != config.ModePersonal {
		return Status{}, ErrRelayBootstrap
	}
	if c.world.Loaded {
		return Status{}, ErrAlreadyLoaded
	}
	if keystorePass == "" {
		return Status{}, errors.New("core: keystore passphrase required")
	}

	path := keystore.Path(c.cfg.DataDir)
	if !keystore.Exists(path) {
		return Status{}, ErrNoKeystore
	}
	ks, err := keystore.Open(path, keystorePass)
	if err != nil {
		// Wrapped, so the API can map a bad passphrase distinctly from an
		// unreadable keystore.
		return Status{}, fmt.Errorf("core: unlock keystore: %w", err)
	}
	c.ks = ks
	c.world = world.FromWorld(ks.World())
	if err := c.activate(); err != nil {
		return Status{}, err
	}
	c.log.Info("keystore unlocked via client", "world", c.world.Fingerprint)
	return c.statusLocked(), nil
}

// ClearKeystore deletes the keystore, discarding the World seed and identity
// passphrase it protects — the recovery path when the keystore passphrase is
// lost, after which a client can bootstrap a new World. Irreversible; the API
// gates it behind an explicit confirmation (#29).
//
// Only permitted while no World is loaded, i.e. from the client's unlock
// screen: a node that has already opened its keystore keeps it (an unlocked
// node's destructive operation is identity burn, not this).
//
// Blocks authored under the discarded World stay in the store; they belong to a
// World the node can no longer open, so they are inert.
func (c *Core) ClearKeystore() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cfg.Mode != config.ModePersonal {
		return ErrRelayBootstrap
	}
	if c.world.Loaded {
		return ErrAlreadyLoaded
	}
	path := keystore.Path(c.cfg.DataDir)
	if !keystore.Exists(path) {
		return ErrNoKeystore
	}
	if err := keystore.Delete(path); err != nil {
		return fmt.Errorf("core: clear keystore: %w", err)
	}
	c.log.Warn("keystore cleared via client; its World seed and identity are unrecoverable")
	return nil
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

// IdentityCard is a node's shareable connection card: its address and public
// keys, which a peer needs to open a connection.
type IdentityCard struct {
	Address  string
	KyberPub string // hex
	MLDSAPub string // hex
}

// PrivateMessage is a decrypted message on a connection's private audience.
type PrivateMessage struct {
	Author    string
	Text      string
	Timestamp int64
}

// IdentityCard returns this node's connection card.
func (c *Core) IdentityCard() (IdentityCard, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ks == nil {
		return IdentityCard{}, ErrCannotAuthor
	}
	id := c.ks.Identity()
	return IdentityCard{
		Address:  string(id.Address),
		KyberPub: hex.EncodeToString(id.Kyber.PublicBytes()),
		MLDSAPub: hex.EncodeToString(id.MLDSA.PublicBytes()),
	}, nil
}

// AddConnectionPeer registers a known peer from its hex-encoded card.
func (c *Core) AddConnectionPeer(addr, kyberPubHex, mldsaPubHex string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		return ErrCannotAuthor
	}
	kyberBytes, err := hex.DecodeString(kyberPubHex)
	if err != nil {
		return fmt.Errorf("core: kyber pub: %w", err)
	}
	kyberPub, err := crypto.KEMScheme().UnmarshalBinaryPublicKey(kyberBytes)
	if err != nil {
		return fmt.Errorf("core: kyber pub: %w", err)
	}
	mldsaBytes, err := hex.DecodeString(mldsaPubHex)
	if err != nil {
		return fmt.Errorf("core: mldsa pub: %w", err)
	}
	mldsaPub, err := crypto.SigScheme().UnmarshalBinaryPublicKey(mldsaBytes)
	if err != nil {
		return fmt.Errorf("core: mldsa pub: %w", err)
	}
	c.conns.AddPeer(derive.Address(addr), kyberPub, mldsaPub)
	return nil
}

// RotateConnection advances a connection to a new epoch.
func (c *Core) RotateConnection(addr string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		return ErrCannotAuthor
	}
	return c.conns.Rotate(derive.Address(addr))
}

// CloseConnection tears a connection down.
func (c *Core) CloseConnection(addr string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		return ErrCannotAuthor
	}
	return c.conns.Close(derive.Address(addr))
}

// SendPrivateText posts an encrypted message on a connection's private audience.
func (c *Core) SendPrivateText(addr, text string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		return "", ErrCannotAuthor
	}
	return c.conns.SendText(derive.Address(addr), text)
}

// ConnectionMessages returns the decrypted messages on a connection's current
// private audience.
func (c *Core) ConnectionMessages(addr string) ([]PrivateMessage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		return nil, ErrCannotAuthor
	}
	var audHex string
	for _, ci := range c.conns.Connections() {
		if ci.Peer == addr {
			audHex = ci.AudienceCode
			break
		}
	}
	if audHex == "" {
		return nil, nil // no connection / no messages yet
	}
	entries, err := c.store.ByAudience(audHex)
	if err != nil {
		return nil, err
	}
	out := make([]PrivateMessage, 0, len(entries))
	for _, e := range entries {
		rec, err := c.store.Get(e.ID)
		if err != nil {
			continue
		}
		if text, ok := c.conns.OpenPrivatePost(rec.Block); ok {
			out = append(out, PrivateMessage{Author: rec.Author, Text: text, Timestamp: rec.Block.Timestamp})
		}
	}
	return out, nil
}

// StartConnection initiates a connection handshake to a registered peer.
func (c *Core) StartConnection(addr derive.Address) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		return "", ErrCannotAuthor
	}
	return c.conns.Start(addr)
}

// Connections lists the node's active connections.
func (c *Core) Connections() []connections.ConnInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conns == nil {
		return nil
	}
	return c.conns.Connections()
}

// SubscribeBlocks registers a live subscriber for an audience ("" = all). It
// returns a buffered channel of events and an unsubscribe function that must be
// called to release it.
// PublicAudienceNum maps an audience_code hex to its public-N (1..16), or 0 if
// the code is not one of the World's well-known public audiences (e.g. a private
// connection audience or the inbox). The feed (#44) uses it to scope to and
// label public posts. Only a node that can derive the World's audiences (personal
// mode) knows these codes, which is why the mapping lives here, not in clients.
func (c *Core) PublicAudienceNum(audienceHex string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.publicNums[audienceHex]
}

func (c *Core) SubscribeBlocks(audienceHex string) (<-chan FeedEvent, func()) {
	c.feedMu.Lock()
	defer c.feedMu.Unlock()
	id := c.nextFeedID
	c.nextFeedID++
	sub := &feedSub{audience: audienceHex, ch: make(chan FeedEvent, 64)}
	c.feeds[id] = sub
	return sub.ch, func() {
		c.feedMu.Lock()
		defer c.feedMu.Unlock()
		if _, ok := c.feeds[id]; ok {
			delete(c.feeds, id)
			close(sub.ch)
		}
	}
}

// onAccepted is the guard accept hook: it fans an accepted block out to matching
// live subscribers. It runs in its own goroutine (the guard invokes hooks async).
func (c *Core) onAccepted(b *blockpb.Block) {
	audHex := hex.EncodeToString(b.AudienceCode)
	ev := FeedEvent{
		ID:        hex.EncodeToString(b.Id),
		Audience:  audHex,
		Type:      hex.EncodeToString(b.TypeCode),
		Timestamp: b.Timestamp,
	}
	// Author + receivedAt come from the just-stored record.
	if rec, err := c.store.Get(ev.ID); err == nil {
		ev.Author = rec.Author
		ev.ReceivedAt = rec.ReceivedAt
	}

	c.feedMu.Lock()
	defer c.feedMu.Unlock()
	for _, s := range c.feeds {
		if s.audience != "" && s.audience != audHex {
			continue
		}
		select {
		case s.ch <- ev:
		default: // slow subscriber: drop rather than block ingestion
		}
	}
}

// RotateIdentity replaces the identity passphrase, deriving a new identity, and
// re-activates so the node follows the new inbox and rebuilds connections under
// the new keys. Returns the new identity address.
func (c *Core) RotateIdentity(newPassphrase string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ks == nil {
		return "", ErrCannotAuthor
	}
	if err := c.ks.RotateIdentity(newPassphrase); err != nil {
		return "", fmt.Errorf("core: rotate identity: %w", err)
	}
	// The connection manager was bound to the old identity; rebuild it.
	c.conns = nil
	if err := c.activate(); err != nil {
		return "", err
	}
	addr := string(c.ks.Identity().Address)
	c.log.Info("identity rotated", "identity", addr)
	return addr, nil
}

// BurnIdentity authors and publishes an identity.burn block on the public lobby,
// revealing this identity's root private keys so peers can revoke trust in it.
// Returns the published block id.
func (c *Core) BurnIdentity(notice string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.ks == nil {
		return "", ErrCannotAuthor
	}
	w := c.ks.World()
	self := c.ks.Identity()
	burn, err := blocktypes.BuildBurn(self, notice)
	if err != nil {
		return "", fmt.Errorf("core: build burn: %w", err)
	}
	payload, err := blocktypes.MarshalPayload(burn)
	if err != nil {
		return "", fmt.Errorf("core: marshal burn: %w", err)
	}
	typeCode := derive.GetTypeCode(w, blocktypes.TypeIdentityBurn)
	aud := audiences.PublicAudience(w, 1)
	b := block.New(self.Address, typeCode, aud.Code, c.now(), payload)
	block.Sign(b, w, self.MLDSA)

	res, err := c.guard.IngestBlock("local", b)
	if err != nil {
		return "", fmt.Errorf("core: ingest burn: %w", err)
	}
	if res.Outcome == guard.Rejected {
		return "", fmt.Errorf("core: burn rejected: %s", res.Reason)
	}
	if c.gossiper != nil {
		if err := c.gossiper.Publish(aud.Code.Hex(), b); err != nil {
			c.log.Debug("core: gossip burn failed", "err", err)
		}
	}
	c.log.Warn("identity burned", "identity", self.Address, "notice", notice, "block", res.ID)
	return res.ID, nil
}

// Lock zeroizes secret material in the current keystore (shutdown).
func (c *Core) Lock() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ks != nil {
		c.ks.Lock()
	}
}
