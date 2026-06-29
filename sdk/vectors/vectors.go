// Package vectors computes BlockParty conformance test vectors (issue #9):
// deterministic outputs of the protocol's core derivations from fixed inputs,
// reproducible by any implementation via the public SDK API. The committed
// vectors.json is the cross-client reference; vectors_test.go re-derives it and
// asserts equality so a change to any derivation is caught.
package vectors

import (
	"bytes"
	"encoding/hex"

	"github.com/bpprotocol/blockparty/sdk/audiences"
	"github.com/bpprotocol/blockparty/sdk/block"
	"github.com/bpprotocol/blockparty/sdk/connections"
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
	Crypto     CryptoVectors     `json:"crypto"`
	World      WorldVectors      `json:"world"`
	Identity   IdentityVectors   `json:"identity"`
	BlockID    BlockIDVectors    `json:"blockID"`
	Audiences  AudienceVectors   `json:"audiences"`
	AEAD       AEADVectors       `json:"aead"`
	Block      BlockVectors      `json:"block"`
	Connection ConnectionVectors `json:"connection"`
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
	Plaintext  string `json:"plaintext"`
	ContentKey string `json:"contentKey"`
	AAD        string `json:"aad"`
	Data       string `json:"data"` // nonce || ciphertext, deterministic with Nonce
}

// ConnectionVectors locks the deterministic connection derivations over fixed
// KEM shared secrets and nonces (the live handshake is non-deterministic).
// AudienceCode1 is the code after one rotation (to epoch 1).
type ConnectionVectors struct {
	SS1               string `json:"ss1"`
	SS2               string `json:"ss2"`
	Nonce             string `json:"nonce"`
	NonceResponse     string `json:"nonceResponse"`
	SS3               string `json:"ss3"`
	CT3               string `json:"ct3"`
	ConnectionSecret0 string `json:"connectionSecret0"`
	AudienceCode0     string `json:"audienceCode0"`
	AudienceCode1     string `json:"audienceCode1"`
}

// BlockVectors is a fully-signed block over fixed inputs. The author is the
// identity derived from (WorldSeed, Passphrase); the payload is raw bytes (no
// encryption — this fixture exercises the block envelope, signatures, and
// canonical encoding). EncodedKeccak is keccak256 of the canonical Protobuf
// encoding, so a conforming implementation that produces a byte-identical
// signed block reproduces it.
type BlockVectors struct {
	Passphrase    string `json:"passphrase"`
	TypeURN       string `json:"typeURN"`
	AudienceURN   string `json:"audienceURN"`
	Timestamp     int64  `json:"timestamp"`
	Data          string `json:"data"`
	IDHex         string `json:"idHex"`
	EncodedKeccak string `json:"encodedKeccak"`
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

	// Connection derivation fixtures over fixed KEM secrets and nonces.
	cSS1 := bytes.Repeat([]byte{0x11}, 32)
	cSS2 := bytes.Repeat([]byte{0x22}, 32)
	cNonce := bytes.Repeat([]byte{0x33}, 32)
	cNonceResp := bytes.Repeat([]byte{0x44}, 32)
	cSS3 := bytes.Repeat([]byte{0x55}, 32)
	cCT3 := bytes.Repeat([]byte{0x66}, 32)
	cSecret0 := connections.DeriveConnectionSecret0(cSS1, cSS2, cNonce, cNonceResp)
	cSecret1 := connections.DeriveRotatedSecret(cSecret0, cSS3, cCT3, 1)

	const aeadPlaintext = "hello, audience"
	aeadData, err := encryption.SealWithNonce(aeadSecret, []byte(audCode), []byte(AEADNonce), []byte(aeadPlaintext), aad)
	if err != nil {
		panic("vectors: sealing AEAD fixture: " + err.Error())
	}

	const blockData = "hello"

	const blockFixtureData = "block fixture payload"
	blk := block.New(id.Address, typeCode, audCode, Timestamp, []byte(blockFixtureData))
	block.Sign(blk, w, id.MLDSA)
	encoded, err := block.Encode(blk)
	if err != nil {
		panic("vectors: encoding block fixture: " + err.Error())
	}

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
			Plaintext:  aeadPlaintext,
			ContentKey: hex.EncodeToString(contentKey),
			AAD:        hex.EncodeToString(aad),
			Data:       hex.EncodeToString(aeadData),
		},
		Block: BlockVectors{
			Passphrase:    Passphrase,
			TypeURN:       TypeURN,
			AudienceURN:   AudienceURN,
			Timestamp:     Timestamp,
			Data:          blockFixtureData,
			IDHex:         block.IDHex(blk),
			EncodedKeccak: hexKeccak(encoded),
		},
		Connection: ConnectionVectors{
			SS1:               hex.EncodeToString(cSS1),
			SS2:               hex.EncodeToString(cSS2),
			Nonce:             hex.EncodeToString(cNonce),
			NonceResponse:     hex.EncodeToString(cNonceResp),
			SS3:               hex.EncodeToString(cSS3),
			CT3:               hex.EncodeToString(cCT3),
			ConnectionSecret0: hex.EncodeToString(cSecret0),
			AudienceCode0:     connections.DeriveAudienceCode(cSecret0, 0).Hex(),
			AudienceCode1:     connections.DeriveAudienceCode(cSecret1, 1).Hex(),
		},
	}
}

func hexKeccak(b []byte) string { return hex.EncodeToString(crypto.Keccak256(b)) }
