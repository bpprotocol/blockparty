// Command bp is the BlockParty reference CLI (issue #9): a minimal node over a
// filesystem transport for opening worlds, deriving identities, exchanging
// audience-scoped posts, and demonstrating the connection handshake.
package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bpprotocol/blockparty/sdk/audiences"
	"github.com/bpprotocol/blockparty/sdk/block"
	"github.com/bpprotocol/blockparty/sdk/blocktypes"
	"github.com/bpprotocol/blockparty/sdk/connections"
	"github.com/bpprotocol/blockparty/sdk/derive"
	"github.com/bpprotocol/blockparty/sdk/identity"
	"github.com/bpprotocol/blockparty/sdk/node"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "address":
		err = cmdAddress(os.Args[2:])
	case "send":
		err = cmdSend(os.Args[2:])
	case "inbox":
		err = cmdInbox(os.Args[2:])
	case "ping":
		err = cmdPing(os.Args[2:])
	case "demo":
		err = cmdDemo(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "bp: unknown command %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "bp:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `bp - BlockParty reference CLI

Usage:
  bp address --world SEED --pass PHRASE
  bp send    --net DIR --world SEED --pass PHRASE --audience N --text MSG
  bp inbox   --net DIR --world SEED --audience N
  bp ping    --node NAME
  bp demo    [--net DIR]
`)
}

func cmdAddress(args []string) error {
	fs := flag.NewFlagSet("address", flag.ExitOnError)
	world := fs.String("world", "", "world seed phrase")
	pass := fs.String("pass", "", "identity passphrase")
	_ = fs.Parse(args)
	id := identity.OpenIdentity(derive.OpenWorld(*world), *pass)
	fmt.Println("address:  ", id.Address)
	fmt.Println("kyber-pub:", base64.StdEncoding.EncodeToString(id.Kyber.PublicBytes()))
	return nil
}

func cmdSend(args []string) error {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	dir := fs.String("net", "", "transport directory")
	world := fs.String("world", "", "world seed phrase")
	pass := fs.String("pass", "", "identity passphrase")
	aud := fs.Int("audience", 1, "public audience number")
	text := fs.String("text", "", "message body")
	_ = fs.Parse(args)

	w := derive.OpenWorld(*world)
	id := identity.OpenIdentity(w, *pass)
	a := audiences.PublicAudience(w, *aud)
	b, err := node.BuildPost(w, id, a.Code, a.Secret, time.Now().Unix(), *text)
	if err != nil {
		return err
	}
	path, err := node.FileStore{Dir: *dir}.Write(b)
	if err != nil {
		return err
	}
	fmt.Printf("sent block %s to %s\n", block.IDHex(b), path)
	return nil
}

func cmdInbox(args []string) error {
	fs := flag.NewFlagSet("inbox", flag.ExitOnError)
	dir := fs.String("net", "", "transport directory")
	world := fs.String("world", "", "world seed phrase")
	aud := fs.Int("audience", 1, "public audience number")
	_ = fs.Parse(args)

	w := derive.OpenWorld(*world)
	a := audiences.PublicAudience(w, *aud)
	r := blocktypes.NewResolver(w)
	blocks, err := node.FileStore{Dir: *dir}.ReadAll()
	if err != nil {
		return err
	}
	count := 0
	for _, b := range blocks {
		var text string
		ok, err := node.OpenPost(w, r, b, a.Secret, &text)
		if err != nil {
			fmt.Printf("[%s] undecryptable: %v\n", block.IDHex(b)[:12], err)
			continue
		}
		if ok {
			count++
			fmt.Printf("[%s] %s\n", block.IDHex(b)[:12], text)
		}
	}
	fmt.Printf("(%d post(s) on public-%d)\n", count, *aud)
	return nil
}

func cmdPing(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	name := fs.String("node", "bp-cli", "responder node name")
	_ = fs.Parse(args)

	sent := time.Now().Unix()
	req := blocktypes.BuildPing("ping-1", sent)
	resp, err := blocktypes.HandlePing(req, time.Now().Unix(), time.Now().Unix(), *name)
	if err != nil {
		return err
	}
	res, err := blocktypes.ParsePingResponse(resp)
	if err != nil {
		return err
	}
	fmt.Printf("pong from %q: sent=%d received=%d server=%d\n", res.Node, res.Timestamp, res.ReceivedAt, res.ServerTime)
	return nil
}

// cmdDemo runs the full two-party connection handshake and a private encrypted
// exchange over the filesystem transport, in one process, printing each step.
func cmdDemo(args []string) error {
	fs := flag.NewFlagSet("demo", flag.ExitOnError)
	dir := fs.String("net", "", "transport directory (default: a temp dir)")
	_ = fs.Parse(args)

	netDir := *dir
	if netDir == "" {
		var err error
		if netDir, err = os.MkdirTemp("", "bp-demo-net-"); err != nil {
			return err
		}
	}
	store := node.FileStore{Dir: netDir}

	w := derive.OpenWorld("bpprotocol.org/v1/demo-world")
	alice := identity.OpenIdentity(w, "alice-demo")
	bob := identity.OpenIdentity(w, "bob-demo")
	fmt.Printf("world opened; alice=%s bob=%s\n", alice.Address, bob.Address)

	alicePeer := connections.Peer{Address: alice.Address, KyberPub: alice.Kyber.Public}
	bobPeer := connections.Peer{Address: bob.Address, KyberPub: bob.Kyber.Public}

	pending, req, err := connections.StartRequest(alice, bobPeer)
	if err != nil {
		return err
	}
	fmt.Println("1. alice -> connect.request")
	resp, bobConn, err := connections.AcceptRequest(bob, alicePeer, req, "req-id")
	if err != nil {
		return err
	}
	fmt.Println("2. bob   -> connect.response")
	aliceConn, err := pending.Complete(resp)
	if err != nil {
		return err
	}
	if aliceConn.AudienceCode().Hex() != bobConn.AudienceCode().Hex() {
		return fmt.Errorf("handshake disagreement: %s != %s", aliceConn.AudienceCode().Hex(), bobConn.AudienceCode().Hex())
	}
	fmt.Printf("3. both derive private audience %s\n", aliceConn.AudienceCode().Hex())

	msg := "the only winning move is to share the keys"
	b, err := node.BuildPost(w, alice, aliceConn.AudienceCode(), aliceConn.AudienceSecret(), time.Now().Unix(), msg)
	if err != nil {
		return err
	}
	path, err := store.Write(b)
	if err != nil {
		return err
	}
	fmt.Printf("4. alice writes encrypted block -> %s\n", path)

	blocks, err := store.ReadAll()
	if err != nil {
		return err
	}
	r := blocktypes.NewResolver(w)
	for _, blk := range blocks {
		if err := block.Verify(blk, w, alice.Dilithium.Public); err != nil {
			return fmt.Errorf("verify: %w", err)
		}
		var text string
		ok, err := node.OpenPost(w, r, blk, bobConn.AudienceSecret(), &text)
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("5. bob verifies + decrypts: %q\n", text)
			if text != msg {
				return fmt.Errorf("decrypted text mismatch")
			}
		}
	}
	fmt.Println("demo OK")
	return nil
}
