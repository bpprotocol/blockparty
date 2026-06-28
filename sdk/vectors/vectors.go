// Package vectors computes BlockParty conformance test vectors (issue #9):
// deterministic outputs of the protocol's core derivations from fixed inputs,
// reproducible by any implementation via the public SDK API. The committed
// vectors.json is the cross-client reference; vectors_test.go re-derives it and
// asserts equality so a change to any derivation is caught.
package vectors

import (
	"encoding/hex"

	"github.com/bpprotocol/blockparty/sdk/audiences"
	"github.com/bpprotocol/blockparty/sdk/crypto"
	"github.com/bpprotocol/blockparty/sdk/derive"
	"github.com/bpprotocol/blockparty/sdk/encryption"
	"github.com/bpprotocol/blockparty/sdk/identity"
)

// Fixed inputs for the conformance vectors.
const (
	WorldSeed      = "bpprotocol.org/v1/test-world"
	Passphrase     = "correct horse battery staple"
	KeySeed        = "bpprotocol.org/v1/test-identity"
	TypeURN        = "bpprotocol.org/v1/types/content.post"
	AudienceURN    = "bpprotocol.org/v1/audience/public-1"
	AEADSecretSeed = "bpprotocol.org/v1/test-secret"
	AEADNonce      = "bpprotocol/v1/test-nonce" // 24 bytes
	Timestamp      = int64(1700000000)
	Version        = uint32(1)
)

// Vectors is the full set of conformance vectors. All byte values are
// lowercase hex; large keys are fingerprinted with Keccak-256.
type Vectors struct {
	Crypto    CryptoVectors   `json:"crypto"`
	World     WorldVectors    `json:"world"`
	Identity  IdentityVectors `json:"identity"`
	BlockID   BlockIDVectors  `json:"blockID"`
	Audiences AudienceVectors `json:"audiences"`
	AEAD      AEADVectors     `json:"aead"`
}

type CryptoVectors struct {
	Seed           string `json:"seed"`
	KyberPubKeccak string `json:"kyberPubKeccak"`
	MLDSAPubKeccak string `json:"mldsaPubKeccak"`
}

type WorldVectors struct {
	Seed             string `json:"seed"`
	WalletSalt       string `json:"walletSalt"`
	TypeSalt         string `json:"typeSalt"`
	AudienceSalt     string `json:"audienceSalt"`
	SigningKeyKeccak string `json:"signingKeyKeccak"`
	TypeCode         string `json:"typeCode"`     // for TypeURN
	AudienceCode     string `json:"audienceCode"` // for AudienceURN
}

type IdentityVectors struct {
	World          string `json:"world"`
	Passphrase     string `json:"passphrase"`
	Address        string `json:"address"`
	MLDSAPubKeccak string `json:"mldsaPubKeccak"`
	KyberPubKeccak string `json:"kyberPubKeccak"`
}

type BlockIDVectors struct {
	Version   uint32 `json:"version"`
	Timestamp int64  `json:"timestamp"`
	Data      string `json:"data"` // utf-8 source of the payload
	ID        string `json:"id"`
}

type AudienceVectors struct {
	PublicAudienceSecret string `json:"publicAudienceSecret"`
	InboxCode            string `json:"inboxCode"`
}

type AEADVectors struct {
	SecretSeed string `json:"secretSeed"`
	Nonce      string `json:"nonce"`
	ContentKey string `json:"contentKey"`
	AAD        string `json:"aad"`
}

// Compute derives all vectors from the fixed inputs via the public SDK API.
func Compute() *Vectors {
	w := derive.OpenWorld(WorldSeed)
	typeCode := derive.GetTypeCode(w, TypeURN)
	audCode := derive.GetAudienceCode(w, AudienceURN)

	id := identity.OpenIdentity(w, Passphrase)

	kyber := crypto.MakeKyberPair([]byte(KeySeed))
	dil := crypto.MakeMLDSAPair([]byte(KeySeed))

	aeadSecret := crypto.Keccak256([]byte(AEADSecretSeed))
	contentKey := encryption.ContentKey(aeadSecret, []byte(audCode), []byte(AEADNonce))
	aad := encryption.AAD(Version, []byte(typeCode), []byte(audCode), Timestamp)

	const blockData = "hello"
	return &Vectors{
		Crypto: CryptoVectors{
			Seed:           KeySeed,
			KyberPubKeccak: hexKeccak(kyber.PublicBytes()),
			MLDSAPubKeccak: hexKeccak(dil.PublicBytes()),
		},
		World: WorldVectors{
			Seed:             WorldSeed,
			WalletSalt:       hex.EncodeToString(w.WalletSalt),
			TypeSalt:         hex.EncodeToString(w.TypeSalt),
			AudienceSalt:     hex.EncodeToString(w.AudienceSalt),
			SigningKeyKeccak: hexKeccak(w.SigningKey.PublicBytes()),
			TypeCode:         typeCode.Hex(),
			AudienceCode:     audCode.Hex(),
		},
		Identity: IdentityVectors{
			World:          WorldSeed,
			Passphrase:     Passphrase,
			Address:        string(id.Address),
			MLDSAPubKeccak: hexKeccak(id.MLDSA.PublicBytes()),
			KyberPubKeccak: hexKeccak(id.Kyber.PublicBytes()),
		},
		BlockID: BlockIDVectors{
			Version:   Version,
			Timestamp: Timestamp,
			Data:      blockData,
			ID:        derive.GetBlockID(Version, Timestamp, audCode, id.Address, typeCode, []byte(blockData)),
		},
		Audiences: AudienceVectors{
			PublicAudienceSecret: hex.EncodeToString(audiences.PublicAudienceSecret(w, AudienceURN)),
			InboxCode:            audiences.InboxAudience(w, id.Address).Code.Hex(),
		},
		AEAD: AEADVectors{
			SecretSeed: AEADSecretSeed,
			Nonce:      AEADNonce,
			ContentKey: hex.EncodeToString(contentKey),
			AAD:        hex.EncodeToString(aad),
		},
	}
}

func hexKeccak(b []byte) string { return hex.EncodeToString(crypto.Keccak256(b)) }
