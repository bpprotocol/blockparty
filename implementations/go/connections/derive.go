package connections

import (
	"strconv"

	"github.com/bpprotocol/blockparty/implementations/go/crypto"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
)

// connectInfo / audienceInfo are the per-epoch HKDF info strings.
func connectInfo(epoch uint64) string {
	return "bpprotocol.org/v1/connect:epoch=" + strconv.FormatUint(epoch, 10)
}

func audienceInfo(epoch uint64) string {
	return "bpprotocol.org/v1/connect-audience:epoch=" + strconv.FormatUint(epoch, 10)
}

// connectionSecret0 derives the epoch-0 connection secret from the two KEM
// shared secrets and the two handshake nonces. Both parties hold ss1, ss2,
// nonce, and nonceResp, so both derive the same secret.
func connectionSecret0(ss1, ss2, nonce, nonceResp []byte) []byte {
	ikm := concat(ss1, ss2)
	salt := concat(nonce, nonceResp)
	return crypto.HKDFSHA256(ikm, salt, []byte(connectInfo(0)), 32)
}

// rotatedSecret derives the next epoch's connection secret by ratcheting the
// previous secret with fresh KEM entropy ss3, salted by the rotation
// ciphertext ct3 (see connections.md §Rotation).
func rotatedSecret(prev, ss3, ct3 []byte, epoch uint64) []byte {
	return crypto.HKDFSHA256(concat(prev, ss3), ct3, []byte(connectInfo(epoch)), 32)
}

// audienceSecret derives the private audience secret for an epoch from the
// connection secret. This is the secret fed to the AEAD layer.
func audienceSecret(connSecret []byte, epoch uint64) []byte {
	return crypto.HKDFSHA256(connSecret, []byte{}, []byte(audienceInfo(epoch)), 32)
}

// audienceCode derives the private audience code for an epoch. It is derived
// from the secret, so it is unlinkable to either participant's address.
func audienceCode(connSecret []byte, epoch uint64) derive.Code {
	return derive.Code(crypto.Keccak256(audienceSecret(connSecret, epoch))[:derive.CodeBytes])
}

func concat(a, b []byte) []byte {
	out := make([]byte, 0, len(a)+len(b))
	out = append(out, a...)
	return append(out, b...)
}
