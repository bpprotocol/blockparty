package encryption

import (
	"crypto/rand"
	"encoding/binary"
	"errors"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
	"github.com/bpprotocol/blockparty/sdk/crypto"
)

// NonceSize is the XChaCha20-Poly1305 nonce length prefixed to the data field.
const NonceSize = chacha20poly1305.NonceSizeX // 24

// aeadInfo is the HKDF info prefix for content-key derivation; the block nonce
// is appended to it.
const aeadInfo = "bpprotocol.org/v1/aead"

var (
	// ErrShortData is returned when a data field is too short to contain a nonce.
	ErrShortData = errors.New("encryption: data shorter than nonce")
	// ErrDecrypt is returned when authentication fails: wrong audience secret,
	// tampered ciphertext, or tampered AAD-bound metadata.
	ErrDecrypt = errors.New("encryption: authentication failed")
)

// ContentKey derives the 32-byte per-block content key from the audience
// secret, the audience code, and the block nonce (see encryption.md).
func ContentKey(audienceSecret, audienceCode, blockNonce []byte) []byte {
	info := append([]byte(aeadInfo), blockNonce...)
	return crypto.HKDFSHA256(audienceSecret, audienceCode, info, 32)
}

// AAD builds the canonical additional authenticated data bound to a block's
// ciphertext: the 4-byte big-endian version, the raw type and audience codes,
// then the 8-byte big-endian timestamp.
func AAD(version uint32, typeCode, audienceCode []byte, timestamp int64) []byte {
	out := make([]byte, 0, 4+len(typeCode)+len(audienceCode)+8)
	out = binary.BigEndian.AppendUint32(out, version)
	out = append(out, typeCode...)
	out = append(out, audienceCode...)
	out = binary.BigEndian.AppendUint64(out, uint64(timestamp))
	return out
}

// EncryptData encrypts plaintext for an audience and returns the data field:
// a fresh 24-byte nonce followed by the AEAD ciphertext (which includes the
// Poly1305 tag).
func EncryptData(audienceSecret, audienceCode, plaintext, aad []byte) ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return sealWithNonce(audienceSecret, audienceCode, nonce, plaintext, aad)
}

// EncryptForBlock is a convenience that builds the AAD from a block's metadata
// and encrypts plaintext into a data field. The caller then constructs the
// block with the returned data (so the block ID binds the ciphertext).
func EncryptForBlock(audienceSecret []byte, version uint32, typeCode, audienceCode []byte, timestamp int64, plaintext []byte) ([]byte, error) {
	return EncryptData(audienceSecret, audienceCode, plaintext, AAD(version, typeCode, audienceCode, timestamp))
}

// sealWithNonce performs the AEAD seal with a caller-supplied nonce. It is the
// deterministic core of EncryptData, used directly only by tests/vectors.
func sealWithNonce(audienceSecret, audienceCode, nonce, plaintext, aad []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(ContentKey(audienceSecret, audienceCode, nonce))
	if err != nil {
		return nil, err
	}
	// Prefix the nonce, then append the sealed ciphertext after it.
	out := make([]byte, NonceSize, NonceSize+len(plaintext)+aead.Overhead())
	copy(out, nonce)
	return aead.Seal(out, nonce, plaintext, aad), nil
}

// DecryptData decrypts a block's data field using the audience secret. It
// reconstructs the AAD from the block's metadata, so any tampering with
// version, type_code, audience_code, or timestamp causes authentication to
// fail. A caller who cannot derive the audience secret cannot decrypt, and the
// block remains an opaque stub.
func DecryptData(audienceSecret []byte, b *blockpb.Block) ([]byte, error) {
	if len(b.Data) < NonceSize {
		return nil, ErrShortData
	}
	nonce, ciphertext := b.Data[:NonceSize], b.Data[NonceSize:]
	aead, err := chacha20poly1305.NewX(ContentKey(audienceSecret, b.AudienceCode, nonce))
	if err != nil {
		return nil, err
	}
	aad := AAD(b.Version, b.TypeCode, b.AudienceCode, b.Timestamp)
	plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}
