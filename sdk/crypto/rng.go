package crypto

import (
	"fmt"
	"io"

	"golang.org/x/crypto/sha3"
)

// DeterministicRNG returns an io.Reader that produces a SHAKE256 extendable
// output stream absorbing seed directly. The same seed always yields the same
// byte stream on any platform, which is what makes key generation reproducible
// across clients.
//
// Per protocol/specs/derivations.md, callers that derive keys first reduce
// their input to a 32-byte value with Keccak256 and pass that as seed (see
// MakeKyberPair / MakeMLDSAPair).
func DeterministicRNG(seed []byte) io.Reader {
	sh := sha3.NewShake256()
	// ShakeHash.Write never returns an error.
	_, _ = sh.Write(seed)
	return sh
}

// readN reads exactly n bytes from r, panicking on failure. The XOF readers
// produced by DeterministicRNG never fail, so a failure here indicates a
// programming error.
func readN(r io.Reader, n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		panic(fmt.Sprintf("crypto: short read from deterministic RNG: %v", err))
	}
	return b
}
