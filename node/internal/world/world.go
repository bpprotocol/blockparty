// Package world resolves which World the node serves and exposes its status.
//
// In personal mode a World is opened from its seed phrase (full World, including
// the signing key); in relay mode only the World public key is loaded (enough to
// validate world_sig, never to author or decrypt). Either mode may boot with no
// World loaded — a client can bootstrap one at runtime (#38, #26 Q5).
package world

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/cloudflare/circl/sign"

	"github.com/bpprotocol/blockparty/implementations/go/crypto"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/node/internal/config"
)

// State captures the loaded World (if any) for the node.
type State struct {
	// Loaded reports whether a World is configured.
	Loaded bool
	// Fingerprint is hex(Keccak256(world signing public key)) — a stable,
	// non-secret identifier a client uses to tell which World is loaded.
	Fingerprint string

	mode  config.Mode
	world *derive.World // personal mode: the full World (nil otherwise)
	pub   []byte        // the World ML-DSA-65 public key (both modes when loaded)
}

// Load resolves the World for the given configuration. It returns a State even
// when no World is configured (Loaded == false); it only errors on malformed
// configured material.
func Load(cfg config.Config) (*State, error) {
	st := &State{mode: cfg.Mode}

	switch cfg.Mode {
	case config.ModePersonal:
		if cfg.WorldSeed == "" {
			return st, nil // unconfigured; a client bootstraps later (#38)
		}
		w := derive.OpenWorld(cfg.WorldSeed)
		st.world = &w
		st.pub = w.SigningKey.PublicBytes()
		st.Loaded = true

	case config.ModeRelay:
		if cfg.WorldPublicKey == "" {
			return st, nil // unconfigured
		}
		pub, err := hex.DecodeString(cfg.WorldPublicKey)
		if err != nil {
			return nil, fmt.Errorf("decode world public key: %w", err)
		}
		st.pub = pub
		st.Loaded = true

	default:
		return nil, fmt.Errorf("unknown mode %q", cfg.Mode)
	}

	if st.Loaded {
		st.Fingerprint = hex.EncodeToString(crypto.Keccak256(st.pub))
	}
	return st, nil
}

// FromWorld builds a loaded personal-mode State from a World already derived
// elsewhere (e.g. unlocked from the keystore, #28).
func FromWorld(w derive.World) *State {
	pub := w.SigningKey.PublicBytes()
	return &State{
		Loaded:      true,
		Fingerprint: hex.EncodeToString(crypto.Keccak256(pub)),
		mode:        config.ModePersonal,
		world:       &w,
		pub:         pub,
	}
}

// Mode returns the node's operating mode.
func (s *State) Mode() config.Mode { return s.mode }

// PublicKey returns the World ML-DSA-65 public key bytes, or nil if no World is
// loaded. The slice is owned by the State and must not be mutated.
func (s *State) PublicKey() []byte { return s.pub }

// SigPublicKey parses the World public key into a verifying key, usable in both
// modes (personal and relay) to check world_sig. It errors if no World is loaded.
func (s *State) SigPublicKey() (sign.PublicKey, error) {
	if len(s.pub) == 0 {
		return nil, errors.New("world: no World loaded")
	}
	return crypto.SigScheme().UnmarshalBinaryPublicKey(s.pub)
}
