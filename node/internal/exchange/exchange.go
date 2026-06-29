// Package exchange implements the block-exchange stream protocol (#34): a
// want/have exchange over libp2p (/bp/exchange/1.0.0) that swaps blocks by ID.
//
// Blocks move as opaque encrypted envelopes — no decryption is needed to relay
// them. The responder serves blocks straight from the store; the requester runs
// every fetched block through the World guard (#31) before it is stored, so a
// peer can never inject a foreign-World or tampered block.
package exchange

import (
	"bufio"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"google.golang.org/protobuf/encoding/protodelim"

	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/node/internal/exchangepb"
	"github.com/bpprotocol/blockparty/node/internal/guard"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

// ProtocolID is the libp2p stream protocol for block exchange.
const ProtocolID = protocol.ID("/bp/exchange/1.0.0")

const (
	maxWantIDs      = 256
	maxMessageBytes = 16 << 20 // per-message read cap (backpressure)
	streamTimeout   = 30 * time.Second
)

// GuardFunc returns the current ingress guard, or nil if no World is loaded.
type GuardFunc func() *guard.Guard

// Exchange serves blocks to peers and fetches blocks from them.
type Exchange struct {
	host  host.Host
	store *store.Store
	guard GuardFunc
	log   *slog.Logger
}

// New builds an exchange over host h, serving from st and ingesting fetched
// blocks through the guard returned by g.
func New(h host.Host, st *store.Store, g GuardFunc, log *slog.Logger) *Exchange {
	return &Exchange{host: h, store: st, guard: g, log: log}
}

// Start registers the stream handler (responder side).
func (e *Exchange) Start() { e.host.SetStreamHandler(ProtocolID, e.handleStream) }

// Stop removes the stream handler.
func (e *Exchange) Stop() { e.host.RemoveStreamHandler(ProtocolID) }

// FetchResult summarizes a fetch.
type FetchResult struct {
	Requested int
	Received  int // blocks the peer returned
	Accepted  int // passed the guard and newly stored
	Duplicate int // already stored
}

// Fetch requests blocks by ID from peer p and ingests any received through the
// guard. Blocks that fail validation are dropped (not counted as accepted).
func (e *Exchange) Fetch(ctx context.Context, p peer.ID, ids [][]byte) (FetchResult, error) {
	res := FetchResult{Requested: len(ids)}
	g := e.guard()
	if g == nil {
		return res, errors.New("exchange: no World loaded; cannot ingest")
	}
	resp, err := e.request(ctx, p, &exchangepb.Want{Ids: ids})
	if err != nil {
		return res, err
	}
	for _, item := range resp.Items {
		if len(item.Block) == 0 {
			continue
		}
		res.Received++
		r, err := g.IngestBytes(p.String(), item.Block)
		if err != nil {
			e.log.Debug("exchange: ingest error", "peer", p.String(), "err", err)
			continue
		}
		switch r.Outcome {
		case guard.Accepted:
			res.Accepted++
		case guard.Duplicate:
			res.Duplicate++
		default:
			e.log.Debug("exchange: fetched block rejected", "peer", p.String(), "reason", r.Reason)
		}
	}
	return res, nil
}

// Have asks peer p which of ids it holds, returning the subset it reports having.
func (e *Exchange) Have(ctx context.Context, p peer.ID, ids [][]byte) ([][]byte, error) {
	resp, err := e.request(ctx, p, &exchangepb.Want{Ids: ids, HaveOnly: true})
	if err != nil {
		return nil, err
	}
	out := make([][]byte, 0, len(resp.Items))
	for _, item := range resp.Items {
		if item.Have {
			out = append(out, item.Id)
		}
	}
	return out, nil
}

func (e *Exchange) request(ctx context.Context, p peer.ID, want *exchangepb.Want) (*exchangepb.WantResponse, error) {
	if len(want.Ids) > maxWantIDs {
		return nil, fmt.Errorf("exchange: too many ids (%d > %d)", len(want.Ids), maxWantIDs)
	}
	s, err := e.host.NewStream(ctx, p, ProtocolID)
	if err != nil {
		return nil, fmt.Errorf("exchange: open stream: %w", err)
	}
	defer s.Close()
	setDeadline(ctx, s)

	if _, err := protodelim.MarshalTo(s, want); err != nil {
		_ = s.Reset()
		return nil, fmt.Errorf("exchange: write want: %w", err)
	}
	_ = s.CloseWrite()

	r := bufio.NewReader(io.LimitReader(s, maxMessageBytes))
	var resp exchangepb.WantResponse
	if err := protodelim.UnmarshalFrom(r, &resp); err != nil {
		_ = s.Reset()
		return nil, fmt.Errorf("exchange: read response: %w", err)
	}
	return &resp, nil
}

// handleStream answers a Want with the blocks (or availability) the node holds.
func (e *Exchange) handleStream(s network.Stream) {
	defer s.Close()
	_ = s.SetDeadline(time.Now().Add(streamTimeout))

	r := bufio.NewReader(io.LimitReader(s, maxMessageBytes))
	var want exchangepb.Want
	if err := protodelim.UnmarshalFrom(r, &want); err != nil {
		e.log.Debug("exchange: read want failed", "err", err)
		_ = s.Reset()
		return
	}
	if _, err := protodelim.MarshalTo(s, e.respond(&want)); err != nil {
		e.log.Debug("exchange: write response failed", "err", err)
		_ = s.Reset()
	}
}

func (e *Exchange) respond(want *exchangepb.Want) *exchangepb.WantResponse {
	ids := want.Ids
	if len(ids) > maxWantIDs {
		ids = ids[:maxWantIDs]
	}
	resp := &exchangepb.WantResponse{}
	for _, id := range ids {
		idHex := hex.EncodeToString(id)
		if want.HaveOnly {
			if has, err := e.store.Has(idHex); err == nil && has {
				resp.Items = append(resp.Items, &exchangepb.BlockItem{Id: id, Have: true})
			}
			continue
		}
		rec, err := e.store.Get(idHex)
		if err != nil {
			continue // absent or unreadable → omit
		}
		b, err := block.Encode(rec.Block)
		if err != nil {
			e.log.Debug("exchange: re-encode failed", "id", idHex, "err", err)
			continue
		}
		resp.Items = append(resp.Items, &exchangepb.BlockItem{Id: id, Block: b, Have: true})
	}
	return resp
}

func setDeadline(ctx context.Context, s network.Stream) {
	if dl, ok := ctx.Deadline(); ok {
		_ = s.SetDeadline(dl)
		return
	}
	_ = s.SetDeadline(time.Now().Add(streamTimeout))
}
