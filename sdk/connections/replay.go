package connections

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/bpprotocol/blockparty/sdk/derive"
)

// DefaultReplayWindow is the default acceptance window for handshake timestamps.
const DefaultReplayWindow = 300 * time.Second

var (
	// ErrReplay is returned when a (target, nonce) pair has been seen before.
	ErrReplay = errors.New("connections: replayed handshake nonce")
	// ErrStale is returned when a handshake timestamp is outside the window.
	ErrStale = errors.New("connections: handshake timestamp outside acceptance window")
)

// ReplayGuard rejects replayed and stale connect.request blocks: a (target,
// nonce) pair may be accepted only once, and only if its timestamp is within
// Window of the guard's clock. It is not safe for concurrent use.
type ReplayGuard struct {
	Window time.Duration
	now    func() time.Time
	seen   map[string]struct{}
}

// NewReplayGuard returns a guard with the given acceptance window (or
// DefaultReplayWindow if non-positive).
func NewReplayGuard(window time.Duration) *ReplayGuard {
	if window <= 0 {
		window = DefaultReplayWindow
	}
	return &ReplayGuard{Window: window, now: time.Now, seen: make(map[string]struct{})}
}

// Check validates a handshake's freshness and uniqueness, recording the nonce
// on success. It returns ErrStale if the timestamp is too far from now, or
// ErrReplay if the (target, nonce) pair was already seen.
func (g *ReplayGuard) Check(target derive.Address, nonce []byte, timestamp int64) error {
	delta := g.now().Sub(time.Unix(timestamp, 0))
	if delta < -g.Window || delta > g.Window {
		return ErrStale
	}
	key := string(target) + "/" + hex.EncodeToString(nonce)
	if _, ok := g.seen[key]; ok {
		return ErrReplay
	}
	g.seen[key] = struct{}{}
	return nil
}
