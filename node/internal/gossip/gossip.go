// Package gossip propagates blocks across the network via libp2p gossipsub (#35).
//
// The node subscribes to one gossipsub topic per audience_code it follows. A
// posted/accepted block is announced on its audience topic; subscribers receive
// the announcement and store the block after validation (#31). Small blocks are
// inlined in the announcement; for larger ones only the ID is gossiped and the
// receiver pulls the full block from the publisher via the exchange protocol
// (#34). Topics are keyed by audience_code, so propagation is audience-scoped.
package gossip

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"google.golang.org/protobuf/proto"

	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/node/internal/exchange"
	"github.com/bpprotocol/blockparty/node/internal/exchangepb"
	"github.com/bpprotocol/blockparty/node/internal/guard"
)

const (
	topicPrefix      = "/bp/audience/1.0.0/"
	defaultInlineMax = 64 << 10 // inline blocks ≤ 64 KiB in the announcement
	fetchTimeout     = 30 * time.Second
)

// GuardFunc returns the current ingress guard, or nil if no World is loaded.
type GuardFunc func() *guard.Guard

// Gossip manages per-audience gossipsub topics.
type Gossip struct {
	host      host.Host
	ps        *pubsub.PubSub
	exchange  *exchange.Exchange
	guard     GuardFunc
	log       *slog.Logger
	inlineMax int

	ctx    context.Context
	cancel context.CancelFunc

	mu     sync.Mutex
	topics map[string]*topicHandle // audience_code hex → handle
}

type topicHandle struct {
	topic  *pubsub.Topic
	sub    *pubsub.Subscription
	cancel context.CancelFunc
}

// New builds gossip over host h. ex is used to pull full blocks announced by ID;
// g supplies the guard that validates everything received before storage.
func New(ctx context.Context, h host.Host, ex *exchange.Exchange, g GuardFunc, log *slog.Logger) (*Gossip, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, fmt.Errorf("gossip: new gossipsub: %w", err)
	}
	cctx, cancel := context.WithCancel(ctx)
	return &Gossip{
		host:      h,
		ps:        ps,
		exchange:  ex,
		guard:     g,
		log:       log,
		inlineMax: defaultInlineMax,
		ctx:       cctx,
		cancel:    cancel,
		topics:    make(map[string]*topicHandle),
	}, nil
}

func topicName(audienceHex string) string { return topicPrefix + audienceHex }

// Follow subscribes to an audience's topic (idempotent).
func (gs *Gossip) Follow(audienceHex string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if _, ok := gs.topics[audienceHex]; ok {
		return nil
	}
	topic, err := gs.ps.Join(topicName(audienceHex))
	if err != nil {
		return fmt.Errorf("gossip: join %s: %w", audienceHex, err)
	}
	sub, err := topic.Subscribe()
	if err != nil {
		_ = topic.Close()
		return fmt.Errorf("gossip: subscribe %s: %w", audienceHex, err)
	}
	tctx, tcancel := context.WithCancel(gs.ctx)
	gs.topics[audienceHex] = &topicHandle{topic: topic, sub: sub, cancel: tcancel}
	go gs.readLoop(tctx, sub)
	gs.log.Debug("gossip: following audience", "audience", audienceHex)
	return nil
}

// Unfollow leaves an audience's topic (idempotent).
func (gs *Gossip) Unfollow(audienceHex string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	th, ok := gs.topics[audienceHex]
	if !ok {
		return nil
	}
	th.cancel()
	th.sub.Cancel()
	err := th.topic.Close()
	delete(gs.topics, audienceHex)
	return err
}

// FollowedCount returns the number of audiences currently followed.
func (gs *Gossip) FollowedCount() int {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	return len(gs.topics)
}

// FollowedAudiences returns the audience_code hexes currently followed.
func (gs *Gossip) FollowedAudiences() []string {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	out := make([]string, 0, len(gs.topics))
	for aud := range gs.topics {
		out = append(out, aud)
	}
	return out
}

// Publish announces a block on its audience topic, inlining the block when small
// enough. Auto-follows the topic if not already.
func (gs *Gossip) Publish(audienceHex string, b *blockpb.Block) error {
	gs.mu.Lock()
	th, ok := gs.topics[audienceHex]
	gs.mu.Unlock()
	if !ok {
		if err := gs.Follow(audienceHex); err != nil {
			return err
		}
		gs.mu.Lock()
		th = gs.topics[audienceHex]
		gs.mu.Unlock()
	}

	item := &exchangepb.BlockItem{Id: b.Id, Have: true}
	if enc, err := block.Encode(b); err == nil && len(enc) <= gs.inlineMax {
		item.Block = enc
	}
	data, err := proto.Marshal(item)
	if err != nil {
		return fmt.Errorf("gossip: marshal announce: %w", err)
	}
	return th.topic.Publish(gs.ctx, data)
}

func (gs *Gossip) readLoop(ctx context.Context, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			return // context cancelled / subscription closed
		}
		if msg.GetFrom() == gs.host.ID() {
			continue // our own announcement
		}
		gs.handleMessage(msg)
	}
}

func (gs *Gossip) handleMessage(msg *pubsub.Message) {
	g := gs.guard()
	if g == nil {
		return // no World loaded; nothing to validate against
	}
	var item exchangepb.BlockItem
	if err := proto.Unmarshal(msg.Data, &item); err != nil {
		gs.log.Debug("gossip: bad announcement", "err", err)
		return
	}

	// Inlined block → validate and store directly.
	if len(item.Block) > 0 {
		if _, err := g.IngestBytes(msg.GetFrom().String(), item.Block); err != nil {
			gs.log.Debug("gossip: ingest failed", "err", err)
		}
		return
	}

	// ID only → pull the full block from the publisher via exchange.
	if len(item.Id) == 0 {
		return
	}
	src := msg.GetFrom()
	fctx, cancel := context.WithTimeout(gs.ctx, fetchTimeout)
	defer cancel()
	if _, err := gs.exchange.Fetch(fctx, src, [][]byte{item.Id}); err != nil {
		gs.log.Debug("gossip: fetch announced block failed", "peer", src.String(), "err", err)
	}
}

// Close leaves all topics and stops gossip.
func (gs *Gossip) Close() error {
	gs.mu.Lock()
	for k, th := range gs.topics {
		th.cancel()
		th.sub.Cancel()
		_ = th.topic.Close()
		delete(gs.topics, k)
	}
	gs.mu.Unlock()
	gs.cancel()
	return nil
}
