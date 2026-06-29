package p2p

import (
	"context"
	"fmt"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

const (
	dhtInitialFind = 2 * time.Second
	dhtFindEvery   = 30 * time.Second
)

// startDHT brings up the Kademlia DHT (#33): it connects to the configured
// bootstrap peers, advertises this node under the rendezvous tag, and
// periodically searches for and dials other nodes advertising the same tag —
// the wide-area counterpart to mDNS. A node behind NAT acts as a DHT client; a
// reachable node serves (ModeAutoServer).
func (ph *Host) startDHT(bootstrap []string, rendezvous string) error {
	kad, err := dht.New(ph.ctx, ph.h, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		return fmt.Errorf("new dht: %w", err)
	}
	ph.kad = kad

	ph.connectBootstrap(bootstrap)
	if err := kad.Bootstrap(ph.ctx); err != nil {
		ph.log.Debug("p2p: dht bootstrap refresh", "err", err)
	}

	rd := drouting.NewRoutingDiscovery(kad)
	dutil.Advertise(ph.ctx, rd, rendezvous)
	go ph.dhtFindLoop(rd, rendezvous)

	ph.log.Info("p2p dht started", "bootstrap_peers", len(bootstrap), "rendezvous", rendezvous)
	return nil
}

func (ph *Host) connectBootstrap(addrs []string) {
	for _, a := range addrs {
		pi, err := peer.AddrInfoFromString(a)
		if err != nil {
			ph.log.Warn("p2p: invalid bootstrap addr", "addr", a, "err", err)
			continue
		}
		ctx, cancel := context.WithTimeout(ph.ctx, connectTimeout)
		if err := ph.h.Connect(ctx, *pi); err != nil {
			ph.log.Debug("p2p: bootstrap connect failed", "peer", pi.ID.String(), "err", err)
		} else {
			ph.markSeen(pi.ID)
			ph.log.Info("p2p: connected to bootstrap peer", "peer", pi.ID.String())
		}
		cancel()
	}
}

func (ph *Host) dhtFindLoop(rd *drouting.RoutingDiscovery, rendezvous string) {
	timer := time.NewTimer(dhtInitialFind)
	defer timer.Stop()
	for {
		select {
		case <-ph.ctx.Done():
			return
		case <-timer.C:
		}
		ph.dhtFindAndConnect(rd, rendezvous)
		timer.Reset(dhtFindEvery)
	}
}

func (ph *Host) dhtFindAndConnect(rd *drouting.RoutingDiscovery, rendezvous string) {
	ctx, cancel := context.WithTimeout(ph.ctx, connectTimeout)
	defer cancel()
	peers, err := rd.FindPeers(ctx, rendezvous)
	if err != nil {
		ph.log.Debug("p2p: dht find failed", "err", err)
		return
	}
	for pi := range peers {
		if pi.ID == ph.h.ID() || pi.ID == "" {
			continue
		}
		if ph.h.Network().Connectedness(pi.ID) == network.Connected {
			ph.markSeen(pi.ID)
			continue
		}
		cctx, ccancel := context.WithTimeout(ph.ctx, connectTimeout)
		if err := ph.h.Connect(cctx, pi); err == nil {
			ph.markSeen(pi.ID)
			ph.log.Info("p2p: connected to DHT peer", "peer", pi.ID.String())
		}
		ccancel()
	}
}
