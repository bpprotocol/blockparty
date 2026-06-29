// Command bpnode is the headless BlockParty node daemon (epic #25).
//
// It participates in a single World's block-exchange network and exposes a
// local API for frontend clients. This is the #27 skeleton: configuration,
// mode selection (personal | relay), the operational API server, and graceful
// lifecycle. Networking, storage, keystore, and the full client API are added
// by the remaining #25 sub-issues.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bpprotocol/blockparty/node/internal/app"
	"github.com/bpprotocol/blockparty/node/internal/config"
	"github.com/bpprotocol/blockparty/node/internal/obs"
)

func main() {
	cfg, err := config.Load(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bpnode: "+err.Error())
		os.Exit(2)
	}

	log := obs.NewLogger(cfg.LogLevel, cfg.LogFormat, os.Stderr)

	d, err := app.New(cfg, log)
	if err != nil {
		log.Error("initialization failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := d.Run(ctx); err != nil {
		log.Error("run failed", "err", err)
		os.Exit(1)
	}
}
