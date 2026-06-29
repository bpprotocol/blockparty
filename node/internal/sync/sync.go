// Package sync provides anti-entropy reconciliation (#36): periodically, for
// each audience the node follows, it asks peers what blocks they hold for that
// audience (a per-audience digest), and fetches the ones it is missing — healing
// gaps without any explicit request. It also advertises the node's audiences as
// DHT provider records and discovers peers for an audience through them.
package sync

import (
	"context"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/bpprotocol/blockparty/node/internal/exchange"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

const (
	defaultInterval = 60 * time.Second
	digestTimeout   = 20 * time.Second
	maxProviders    = 8
)

// Host is the subset of the p2p host the reconciler needs (satisfied by
// *p2p.Host).
type Host interface {
	Provide(ctx context.Context, key string) error
	FindProviders(ctx context.Context, key string, limit int) []peer.AddrInfo
	Connect(ctx context.Context, pi peer.AddrInfo) error
	ConnectedIDs() []peer.ID
}

// AudienceSource yields the audiences to reconcile (audience_code hexes).
type AudienceSource func() []string

// Reconciler heals missing blocks via per-audience digests + fetch.
type Reconciler struct {
	host     Host
	exchange *exchange.Exchange
	store    *store.Store
	auds     AudienceSource
	log      *slog.Logger
	interval time.Duration
}

// New builds a reconciler. auds supplies the audiences to reconcile (typically
// the gossip's followed set).
func New(host Host, ex *exchange.Exchange, st *store.Store, auds AudienceSource, log *slog.Logger) *Reconciler {
	return &Reconciler{
		host:     host,
		exchange: ex,
		store:    st,
		auds:     auds,
		log:      log,
		interval: defaultInterval,
	}
}

// Run reconciles on a timer until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context) {
	t := time.NewTimer(r.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		r.ReconcileOnce(ctx)
		t.Reset(r.interval)
	}
}

// ReconcileOnce runs one reconciliation pass over all followed audiences.
func (r *Reconciler) ReconcileOnce(ctx context.Context) {
	for _, aud := range r.auds() {
		// Advertise that we serve this audience, and gather candidate peers:
		// everyone we're connected to, plus DHT providers for the audience.
		_ = r.host.Provide(ctx, aud)
		for _, pid := range r.candidates(ctx, aud) {
			r.ReconcileAudience(ctx, pid, aud)
		}
	}
}

// candidates returns peer IDs to reconcile an audience with: connected peers
// plus any DHT providers for the audience (dialed if not yet connected).
func (r *Reconciler) candidates(ctx context.Context, aud string) []peer.ID {
	seen := map[peer.ID]bool{}
	var out []peer.ID
	for _, id := range r.host.ConnectedIDs() {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	for _, pi := range r.host.FindProviders(ctx, aud, maxProviders) {
		if seen[pi.ID] {
			continue
		}
		if err := r.host.Connect(ctx, pi); err != nil {
			continue
		}
		seen[pi.ID] = true
		out = append(out, pi.ID)
	}
	return out
}

// ReconcileAudience pulls from peer any blocks for an audience that the local
// store is missing. Returns the number of blocks newly fetched.
func (r *Reconciler) ReconcileAudience(ctx context.Context, p peer.ID, audienceHex string) int {
	code, err := hex.DecodeString(audienceHex)
	if err != nil {
		return 0
	}
	dctx, cancel := context.WithTimeout(ctx, digestTimeout)
	defer cancel()
	ids, err := r.exchange.Digest(dctx, p, code)
	if err != nil {
		r.log.Debug("sync: digest failed", "peer", p.String(), "err", err)
		return 0
	}

	var missing [][]byte
	for _, id := range ids {
		has, err := r.store.Has(hex.EncodeToString(id))
		if err == nil && !has {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return 0
	}
	res, err := r.exchange.Fetch(dctx, p, missing)
	if err != nil {
		r.log.Debug("sync: fetch failed", "peer", p.String(), "err", err)
		return 0
	}
	if res.Accepted > 0 {
		r.log.Info("sync: reconciled blocks", "peer", p.String(), "audience", audienceHex, "fetched", res.Accepted)
	}
	return res.Accepted
}
