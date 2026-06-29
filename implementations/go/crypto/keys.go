package crypto

import (
	"github.com/cloudflare/circl/kem"
	"github.com/cloudflare/circl/kem/mlkem/mlkem768"
	"github.com/cloudflare/circl/sign"
	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

// kemScheme is ML-KEM768 (FIPS 203); sigScheme is ML-DSA-65 (FIPS 204).
var (
	kemScheme = mlkem768.Scheme()
	sigScheme = mldsa65.Scheme()
)

// KEMScheme returns the ML-KEM768 scheme, e.g. for unmarshalling public keys
// and ciphertexts carried in blocks.
func KEMScheme() kem.Scheme { return kemScheme }

// SigScheme returns the ML-DSA-65 signature scheme, e.g. for unmarshalling
// public keys carried in blocks.
func SigScheme() sign.Scheme { return sigScheme }

// KyberKeyPair is an ML-KEM768 key-encapsulation keypair.
type KyberKeyPair struct {
	Public  kem.PublicKey
	Private kem.PrivateKey
}

// MLDSAKeyPair is a ML-DSA-65 signature keypair.
type MLDSAKeyPair struct {
	Public  sign.PublicKey
	Private sign.PrivateKey
}

// MakeKyberPair deterministically derives an ML-KEM768 keypair from seed.
// The same seed always yields the same keypair (derivations.md §MakeKyberPair):
// the seed is reduced with Keccak256, expanded through a SHAKE256 XOF, and the
// scheme's seed bytes drive the standardized ML-KEM key generation.
func MakeKyberPair(seed []byte) KyberKeyPair {
	rng := DeterministicRNG(Keccak256(seed))
	pub, priv := kemScheme.DeriveKeyPair(readN(rng, kemScheme.SeedSize()))
	return KyberKeyPair{Public: pub, Private: priv}
}

// MakeMLDSAPair deterministically derives a ML-DSA-65 keypair from seed,
// mirroring MakeKyberPair.
func MakeMLDSAPair(seed []byte) MLDSAKeyPair {
	rng := DeterministicRNG(Keccak256(seed))
	pub, priv := sigScheme.DeriveKey(readN(rng, sigScheme.SeedSize()))
	return MLDSAKeyPair{Public: pub, Private: priv}
}

// PublicBytes returns the canonical binary encoding of the public key.
func (kp KyberKeyPair) PublicBytes() []byte { return mustMarshal(kp.Public) }

// PublicBytes returns the canonical binary encoding of the public key.
func (kp MLDSAKeyPair) PublicBytes() []byte { return mustMarshal(kp.Public) }

// Sign returns a ML-DSA-65 signature over message.
func (kp MLDSAKeyPair) Sign(message []byte) []byte {
	return sigScheme.Sign(kp.Private, message, nil)
}

// VerifyMLDSA reports whether sig is a valid ML-DSA-65 signature over
// message under pub.
func VerifyMLDSA(pub sign.PublicKey, message, sig []byte) bool {
	return sigScheme.Verify(pub, message, sig, nil)
}

// Encapsulate generates a fresh shared secret for pub and returns the
// ciphertext and shared secret.
func Encapsulate(pub kem.PublicKey) (ciphertext, sharedSecret []byte, err error) {
	return kemScheme.Encapsulate(pub)
}

// Decapsulate recovers the shared secret encapsulated in ciphertext for this
// keypair's private key.
func (kp KyberKeyPair) Decapsulate(ciphertext []byte) (sharedSecret []byte, err error) {
	return kemScheme.Decapsulate(kp.Private, ciphertext)
}

type binaryMarshaler interface{ MarshalBinary() ([]byte, error) }

// mustMarshal marshals a key whose MarshalBinary never fails for valid keys.
func mustMarshal(k binaryMarshaler) []byte {
	b, err := k.MarshalBinary()
	if err != nil {
		panic("crypto: marshalling key: " + err.Error())
	}
	return b
}
