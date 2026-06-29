package world

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/node/internal/config"
)

const testSeed = "correct horse battery staple"

func TestPersonalUnconfigured(t *testing.T) {
	st, err := Load(config.Config{Mode: config.ModePersonal})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Loaded {
		t.Error("expected no World loaded without a seed")
	}
	if st.Fingerprint != "" {
		t.Error("expected empty fingerprint when unloaded")
	}
}

func TestPersonalLoadsWorld(t *testing.T) {
	st, err := Load(config.Config{Mode: config.ModePersonal, WorldSeed: testSeed})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !st.Loaded {
		t.Fatal("expected World loaded")
	}
	// Fingerprint must match the SDK-derived World's signing public key.
	w := derive.OpenWorld(testSeed)
	if got, want := hex.EncodeToString(st.PublicKey()), hex.EncodeToString(w.SigningKey.PublicBytes()); got != want {
		t.Errorf("public key mismatch:\n got %s\nwant %s", got, want)
	}
	if st.Fingerprint == "" {
		t.Error("expected a non-empty fingerprint")
	}
}

func TestRelayLoadsPublicKeyAndMatchesPersonal(t *testing.T) {
	// A relay configured with the personal World's public key must compute the
	// same fingerprint — the proof that both modes identify the same World.
	w := derive.OpenWorld(testSeed)
	pubHex := hex.EncodeToString(w.SigningKey.PublicBytes())

	relay, err := Load(config.Config{Mode: config.ModeRelay, WorldPublicKey: pubHex})
	if err != nil {
		t.Fatalf("Load relay: %v", err)
	}
	if !relay.Loaded {
		t.Fatal("expected relay World loaded")
	}

	personal, err := Load(config.Config{Mode: config.ModePersonal, WorldSeed: testSeed})
	if err != nil {
		t.Fatalf("Load personal: %v", err)
	}
	if relay.Fingerprint != personal.Fingerprint {
		t.Errorf("fingerprint mismatch: relay %s vs personal %s", relay.Fingerprint, personal.Fingerprint)
	}
}

func TestRelayUnconfigured(t *testing.T) {
	st, err := Load(config.Config{Mode: config.ModeRelay})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Loaded {
		t.Error("expected no World loaded without a public key")
	}
}
