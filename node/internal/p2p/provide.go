package p2p

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

// keyCID maps an arbitrary string key (e.g. an audience_code hex) to a CID used
// as the DHT provider-record key.
func keyCID(key string) (cid.Cid, error) {
	mh, err := multihash.Sum([]byte(key), multihash.SHA2_256, -1)
	if err != nil {
		return cid.Undef, err
	}
	return cid.NewCidV1(cid.Raw, mh), nil
}

// Provide announces, via the DHT, that this node holds content for key (an
// audience_code). Other nodes find it with FindProviders. No-op without a DHT.
func (ph *Host) Provide(ctx context.Context, key string) error {
	if ph.kad == nil {
		return nil
	}
	c, err := keyCID(key)
	if err != nil {
		return err
	}
	if err := ph.kad.Provide(ctx, c, true); err != nil {
		return fmt.Errorf("p2p: provide %s: %w", key, err)
	}
	return nil
}

// FindProviders returns up to limit peers that have advertised key in the DHT.
func (ph *Host) FindProviders(ctx context.Context, key string, limit int) []peer.AddrInfo {
	if ph.kad == nil {
		return nil
	}
	c, err := keyCID(key)
	if err != nil {
		return nil
	}
	var out []peer.AddrInfo
	for pi := range ph.kad.FindProvidersAsync(ctx, c, limit) {
		if pi.ID == ph.h.ID() {
			continue
		}
		out = append(out, pi)
	}
	return out
}
