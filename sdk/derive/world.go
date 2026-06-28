package derive

import "github.com/bpprotocol/blockparty/sdk/crypto"

// GlobalSalt is the fixed, non-secret domain-separation salt used to open a
// World from a seed phrase. It namespaces BlockParty derivations away from
// other systems that might reuse the same seed phrases.
const GlobalSalt = "bpprotocol.org/v1/global"

// World is a cryptographic domain: a ML-DSA signing key (the world_sig
// authority) plus the three salts that scope identity, type, and audience
// derivations.
type World struct {
	SigningKey   crypto.MLDSAKeyPair
	WalletSalt   []byte // scopes identity derivation
	TypeSalt     []byte // scopes type codes
	AudienceSalt []byte // scopes audience codes
}

// OpenWorld deterministically derives a World from a seed phrase. Anyone who
// knows the phrase derives the identical World, including its signing key.
func OpenWorld(seedPhrase string) World {
	worldSeed := crypto.HMACSHA256([]byte(GlobalSalt), []byte(seedPhrase))
	return GenerateWorld(crypto.MakeMLDSAPair(worldSeed), worldSeed)
}

// GenerateWorld builds a World from a signing key and the world seed. It is
// exposed to mirror the spec; most callers use OpenWorld.
func GenerateWorld(signingKey crypto.MLDSAKeyPair, worldSeed []byte) World {
	return World{
		SigningKey:   signingKey,
		WalletSalt:   crypto.HMACSHA256(worldSeed, []byte("wallets")),
		TypeSalt:     crypto.HMACSHA256(worldSeed, []byte("types")),
		AudienceSalt: crypto.HMACSHA256(worldSeed, []byte("channels")),
	}
}
