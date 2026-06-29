package crypto

import (
	"bytes"
	"testing"
)

func TestDeterministicRNGReproducible(t *testing.T) {
	seed := []byte("bpprotocol.org/v1/test-seed")
	a := readN(DeterministicRNG(seed), 128)
	b := readN(DeterministicRNG(seed), 128)
	if !bytes.Equal(a, b) {
		t.Fatal("same seed produced different streams")
	}
}

func TestDeterministicRNGSeedSensitive(t *testing.T) {
	a := readN(DeterministicRNG([]byte("seed-a")), 64)
	b := readN(DeterministicRNG([]byte("seed-b")), 64)
	if bytes.Equal(a, b) {
		t.Fatal("different seeds produced identical streams")
	}
}

func TestDeterministicRNGStreamIsContinuous(t *testing.T) {
	// Reading 64 then 64 must equal reading 128 at once (it is one XOF stream).
	seed := []byte("continuity")
	r := DeterministicRNG(seed)
	first := readN(r, 64)
	second := readN(r, 64)
	whole := readN(DeterministicRNG(seed), 128)
	if !bytes.Equal(append(first, second...), whole) {
		t.Fatal("split reads do not match a single read of the XOF stream")
	}
}
