package config

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// noEnv is an empty environment.
func noEnv(string) string { return "" }

// envFrom builds a getenv from a map.
func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(nil, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mode != ModePersonal {
		t.Errorf("default mode = %q, want personal", cfg.Mode)
	}
	if cfg.APIAddr != "127.0.0.1:4400" {
		t.Errorf("default api-addr = %q, want loopback", cfg.APIAddr)
	}
}

func TestInvalidModeRejected(t *testing.T) {
	if _, err := Load([]string{"--mode", "bogus"}, noEnv); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestModeFromFlagAndEnv(t *testing.T) {
	cfg, err := Load([]string{"--mode", "relay"}, noEnv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mode != ModeRelay {
		t.Errorf("mode = %q, want relay", cfg.Mode)
	}

	cfg, err = Load(nil, envFrom(map[string]string{"BPNODE_MODE": "relay"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Mode != ModeRelay {
		t.Errorf("env mode = %q, want relay", cfg.Mode)
	}
}

func TestFlagOverridesEnv(t *testing.T) {
	cfg, err := Load([]string{"--api-addr", "127.0.0.1:9999"},
		envFrom(map[string]string{"BPNODE_API_ADDR": "127.0.0.1:1111"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIAddr != "127.0.0.1:9999" {
		t.Errorf("api-addr = %q, want the flag value", cfg.APIAddr)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{"api_addr":"127.0.0.1:2222","log_level":"debug"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load([]string{"--config", path},
		envFrom(map[string]string{"BPNODE_API_ADDR": "127.0.0.1:3333"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.APIAddr != "127.0.0.1:3333" {
		t.Errorf("api-addr = %q, want env to override file", cfg.APIAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("log-level = %q, want file value to survive", cfg.LogLevel)
	}
}

func TestRelayWithSeedRejected(t *testing.T) {
	_, err := Load([]string{"--mode", "relay"},
		envFrom(map[string]string{"BPNODE_WORLD_SEED": "correct horse battery staple"}))
	if err == nil {
		t.Fatal("expected relay+seed to be rejected")
	}
	if !strings.Contains(err.Error(), "seed") {
		t.Errorf("error = %q, want it to mention the seed", err)
	}
}

func TestWorldPubkeyValidation(t *testing.T) {
	// Wrong length is rejected.
	if _, err := Load([]string{"--mode", "relay", "--world-pubkey", "abcd"}, noEnv); err == nil {
		t.Fatal("expected short pubkey to be rejected")
	}
	// Non-hex is rejected.
	if _, err := Load([]string{"--mode", "relay", "--world-pubkey", "zz"}, noEnv); err == nil {
		t.Fatal("expected non-hex pubkey to be rejected")
	}
	// Correct length passes.
	good := hex.EncodeToString(make([]byte, mldsa65PublicKeyBytes))
	if _, err := Load([]string{"--mode", "relay", "--world-pubkey", good}, noEnv); err != nil {
		t.Fatalf("valid-length pubkey rejected: %v", err)
	}
}

func TestWorldSeedFromEnvOnly(t *testing.T) {
	cfg, err := Load(nil, envFrom(map[string]string{"BPNODE_WORLD_SEED": "a seed phrase"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.WorldSeed != "a seed phrase" {
		t.Errorf("WorldSeed = %q, want it sourced from env", cfg.WorldSeed)
	}
}
