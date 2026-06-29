// Package config defines the bpnode daemon configuration and how it is resolved
// from (in increasing order of precedence): built-in defaults, an optional JSON
// config file, environment variables, and command-line flags.
package config

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Mode selects how much cryptographic authority the node carries. See issue #25
// ("Node roles & key custody") for the decision behind the two modes.
type Mode string

const (
	// ModePersonal holds the keystore (World seed + identity keys) and can
	// author, sign, decrypt, and run connections. Keystore lands in #28.
	ModePersonal Mode = "personal"
	// ModeRelay holds only the World public key: it validates world_sig,
	// stores, and gossips ciphertext, but cannot author or decrypt. Safe to run
	// on untrusted/remote hosts.
	ModeRelay Mode = "relay"
)

// mldsa65PublicKeyBytes is the encoded size of an ML-DSA-65 (FIPS 204) public
// key, used to sanity-check a relay-mode World public key.
const mldsa65PublicKeyBytes = 1952

// Config is the fully-resolved daemon configuration.
type Config struct {
	Mode      Mode     `json:"mode"`
	DataDir   string   `json:"data_dir"`
	APIAddr   string   `json:"api_addr"`   // client API listen address; loopback by default (#29)
	P2PListen []string `json:"p2p_listen"` // reserved for libp2p (#32)
	Bootstrap []string `json:"bootstrap"`  // reserved for DHT bootstrap peers (#33)
	LogLevel  string   `json:"log_level"`
	LogFormat string   `json:"log_format"`

	// World bootstrap material.
	//
	// TEMPORARY: in personal mode the seed will move into the OS-keychain-backed
	// keystore (#28), and a World may instead be bootstrapped at runtime by a
	// client (#38, #26 Q5). Either field may be empty, in which case the node
	// boots with no World loaded and reports so via /statusz.
	WorldSeed      string `json:"-"`            // personal only; secret — sourced from env, never a flag, never persisted here
	WorldPublicKey string `json:"world_pubkey"` // relay; ML-DSA-65 public key, hex
}

// Defaults returns the baseline configuration before any overlay.
func Defaults() Config {
	return Config{
		Mode:      ModePersonal,
		DataDir:   defaultDataDir(),
		APIAddr:   "127.0.0.1:4400",
		LogLevel:  "info",
		LogFormat: "text",
	}
}

func defaultDataDir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "blockparty", "node")
}

// Load resolves the configuration from args and the environment. getenv is
// injected (pass os.Getenv) so loading is testable without touching the real
// environment.
func Load(args []string, getenv func(string) string) (Config, error) {
	cfg := Defaults()

	fs := flag.NewFlagSet("bpnode", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // errors are returned, not printed here
	var (
		fMode      = fs.String("mode", string(cfg.Mode), "operating mode: personal | relay")
		fDataDir   = fs.String("data-dir", cfg.DataDir, "data directory")
		fAPIAddr   = fs.String("api-addr", cfg.APIAddr, "client API listen address (loopback recommended)")
		fLogLevel  = fs.String("log-level", cfg.LogLevel, "log level: debug | info | warn | error")
		fLogFormat = fs.String("log-format", cfg.LogFormat, "log format: text | json")
		fP2P       = fs.String("p2p-listen", strings.Join(cfg.P2PListen, ","), "comma-separated libp2p listen multiaddrs (reserved, #32)")
		fBoot      = fs.String("bootstrap", strings.Join(cfg.Bootstrap, ","), "comma-separated bootstrap peers (reserved, #33)")
		fPubKey    = fs.String("world-pubkey", cfg.WorldPublicKey, "relay mode: World ML-DSA-65 public key (hex)")
		fConfig    = fs.String("config", "", "path to a JSON config file")
	)
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	// 1. optional config file overlays the defaults (absent keys are left as-is).
	cfgPath := *fConfig
	if cfgPath == "" {
		cfgPath = getenv("BPNODE_CONFIG")
	}
	if cfgPath != "" {
		if err := overlayFile(&cfg, cfgPath); err != nil {
			return Config{}, err
		}
	}

	// 2. environment overlays the file.
	overlayEnv(&cfg, getenv)

	// 3. explicitly-set flags overlay everything (highest precedence).
	if set["mode"] {
		cfg.Mode = Mode(*fMode)
	}
	if set["data-dir"] {
		cfg.DataDir = *fDataDir
	}
	if set["api-addr"] {
		cfg.APIAddr = *fAPIAddr
	}
	if set["log-level"] {
		cfg.LogLevel = *fLogLevel
	}
	if set["log-format"] {
		cfg.LogFormat = *fLogFormat
	}
	if set["p2p-listen"] {
		cfg.P2PListen = splitList(*fP2P)
	}
	if set["bootstrap"] {
		cfg.Bootstrap = splitList(*fBoot)
	}
	if set["world-pubkey"] {
		cfg.WorldPublicKey = *fPubKey
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func overlayFile(c *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}
	// Unmarshalling into the existing struct leaves absent keys untouched.
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("parse config file %s: %w", path, err)
	}
	return nil
}

func overlayEnv(c *Config, getenv func(string) string) {
	if v := getenv("BPNODE_MODE"); v != "" {
		c.Mode = Mode(v)
	}
	if v := getenv("BPNODE_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := getenv("BPNODE_API_ADDR"); v != "" {
		c.APIAddr = v
	}
	if v := getenv("BPNODE_LOG_LEVEL"); v != "" {
		c.LogLevel = v
	}
	if v := getenv("BPNODE_LOG_FORMAT"); v != "" {
		c.LogFormat = v
	}
	if v := getenv("BPNODE_P2P_LISTEN"); v != "" {
		c.P2PListen = splitList(v)
	}
	if v := getenv("BPNODE_BOOTSTRAP"); v != "" {
		c.Bootstrap = splitList(v)
	}
	if v := getenv("BPNODE_WORLD_PUBKEY"); v != "" {
		c.WorldPublicKey = v
	}
	if v := getenv("BPNODE_WORLD_SEED"); v != "" {
		c.WorldSeed = v
	}
}

// Validate checks the resolved configuration for internal consistency.
func (c Config) Validate() error {
	switch c.Mode {
	case ModePersonal, ModeRelay:
	default:
		return fmt.Errorf("invalid mode %q (want %q or %q)", c.Mode, ModePersonal, ModeRelay)
	}
	if c.APIAddr == "" {
		return errors.New("api-addr must not be empty")
	}
	if c.DataDir == "" {
		return errors.New("data-dir must not be empty")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid log-level %q (want debug|info|warn|error)", c.LogLevel)
	}
	switch c.LogFormat {
	case "text", "json":
	default:
		return fmt.Errorf("invalid log-format %q (want text|json)", c.LogFormat)
	}
	// A relay must never be handed the World seed — it holds only the public key.
	if c.Mode == ModeRelay && c.WorldSeed != "" {
		return errors.New("relay mode must not be given a World seed (it holds only the World public key)")
	}
	if c.WorldPublicKey != "" {
		b, err := hex.DecodeString(c.WorldPublicKey)
		if err != nil {
			return fmt.Errorf("world-pubkey: invalid hex: %w", err)
		}
		if len(b) != mldsa65PublicKeyBytes {
			return fmt.Errorf("world-pubkey: got %d bytes, want %d (ML-DSA-65)", len(b), mldsa65PublicKeyBytes)
		}
	}
	return nil
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
