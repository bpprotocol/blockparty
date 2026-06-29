package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/bpprotocol/blockparty/node/internal/api"
	"github.com/bpprotocol/blockparty/node/internal/config"
	"github.com/bpprotocol/blockparty/node/internal/obs"
)

// runDaemon boots a daemon on an ephemeral port and returns its base URL plus a
// stop func that cancels Run and waits for clean shutdown.
func runDaemon(t *testing.T, cfg config.Config) (string, func()) {
	t.Helper()
	cfg.APIAddr = "127.0.0.1:0"
	if cfg.DataDir == "" {
		cfg.DataDir = t.TempDir()
	}
	if cfg.P2PListen == nil {
		cfg.P2PListen = []string{"/ip4/127.0.0.1/tcp/0"} // hermetic: loopback only
	}
	log := obs.NewLogger("error", "text", io.Discard)

	d, err := New(cfg, log)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- d.Run(ctx) }()

	base := waitReady(t, d)
	return base, func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run returned error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("daemon did not shut down within 5s")
		}
	}
}

// waitReady polls until the API answers /healthz, returning the base URL.
func waitReady(t *testing.T, d *Daemon) string {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		addr := d.APIAddr()
		if addr != "" {
			if resp, err := http.Get("http://" + addr + "/healthz"); err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return "http://" + addr
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("daemon API never became ready")
	return ""
}

func TestDaemonBootsAndReportsMode(t *testing.T) {
	base, stop := runDaemon(t, config.Config{Mode: config.ModeRelay})
	defer stop()

	resp, err := http.Get(base + "/statusz")
	if err != nil {
		t.Fatalf("GET /statusz: %v", err)
	}
	defer resp.Body.Close()

	var st api.Status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if st.Mode != "relay" {
		t.Errorf("status mode = %q, want relay", st.Mode)
	}
	if st.WorldLoaded {
		t.Error("expected no World loaded for an unconfigured relay")
	}
	if st.Version != Version {
		t.Errorf("status version = %q, want %q", st.Version, Version)
	}
	if _, ok := st.Metrics["http_requests"]; !ok {
		t.Error("expected http_requests metric in status")
	}
}

func TestDaemonPersonalWorldLoaded(t *testing.T) {
	base, stop := runDaemon(t, config.Config{
		Mode:      config.ModePersonal,
		WorldSeed: "correct horse battery staple",
	})
	defer stop()

	resp, err := http.Get(base + "/statusz")
	if err != nil {
		t.Fatalf("GET /statusz: %v", err)
	}
	defer resp.Body.Close()

	var st api.Status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !st.WorldLoaded {
		t.Error("expected World loaded in personal mode with a seed")
	}
	if st.World == "" {
		t.Error("expected a World fingerprint when loaded")
	}
}

// TestDaemonKeystoreUnlocksAcrossRestart proves the #28 acceptance at the daemon
// level: first boot initializes the keystore from a seed; a second boot (same
// data dir, same unlock passphrase, no seed) unlocks it and derives the same
// identity.
func TestDaemonKeystoreUnlocksAcrossRestart(t *testing.T) {
	dir := t.TempDir()

	// First boot: initialize the keystore from a seed + identity passphrase.
	base, stop := runDaemon(t, config.Config{
		Mode:               config.ModePersonal,
		DataDir:            dir,
		KeystorePassphrase: "unlock",
		WorldSeed:          "correct horse battery staple",
		IdentityPassphrase: "id-pass",
	})
	first := getStatus(t, base)
	stop()

	if first.Identity == "" {
		t.Fatal("expected an identity address after keystore init")
	}

	// Second boot: NO seed — must unlock the persisted keystore and match.
	base2, stop2 := runDaemon(t, config.Config{
		Mode:               config.ModePersonal,
		DataDir:            dir,
		KeystorePassphrase: "unlock",
	})
	defer stop2()
	second := getStatus(t, base2)

	if !second.WorldLoaded {
		t.Error("expected World loaded from keystore on restart")
	}
	if second.Identity != first.Identity {
		t.Errorf("identity across restart = %q, want %q", second.Identity, first.Identity)
	}
	if second.World != first.World {
		t.Errorf("World fingerprint across restart = %q, want %q", second.World, first.World)
	}
}

func getStatus(t *testing.T, base string) api.Status {
	t.Helper()
	resp, err := http.Get(base + "/statusz")
	if err != nil {
		t.Fatalf("GET /statusz: %v", err)
	}
	defer resp.Body.Close()
	var st api.Status
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	return st
}
