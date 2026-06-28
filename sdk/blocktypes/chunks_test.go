package blocktypes

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
)

func TestReassembleChunks(t *testing.T) {
	parts := map[string][]byte{
		"c1": []byte("Hello, "),
		"c2": []byte("world"),
		"c3": []byte("!"),
	}
	order := []string{"c1", "c2", "c3"}
	full := []byte("Hello, world!")
	sum := sha512.Sum512(full)

	got, err := ReassembleChunks(order, parts, hex.EncodeToString(sum[:]))
	if err != nil {
		t.Fatalf("ReassembleChunks: %v", err)
	}
	if !bytes.Equal(got, full) {
		t.Fatalf("reassembled %q, want %q", got, full)
	}
}

func TestReassembleBadChecksum(t *testing.T) {
	parts := map[string][]byte{"c1": []byte("data")}
	if _, err := ReassembleChunks([]string{"c1"}, parts, "00"); !errors.Is(err, ErrChecksum) {
		t.Fatalf("bad checksum err = %v, want ErrChecksum", err)
	}
}

func TestReassembleMissingChunk(t *testing.T) {
	if _, err := ReassembleChunks([]string{"missing"}, map[string][]byte{}, "x"); !errors.Is(err, ErrMissingChunk) {
		t.Fatalf("missing chunk err = %v, want ErrMissingChunk", err)
	}
}

func TestReassembleViaManifest(t *testing.T) {
	parts := map[string][]byte{"a": []byte("foo"), "b": []byte("bar")}
	full := []byte("foobar")
	sum := sha512.Sum512(full)
	m := &blockpb.ChunkManifest{Chunks: []string{"a", "b"}, Sha512: hex.EncodeToString(sum[:])}
	got, err := ReassembleChunkManifest(m, parts)
	if err != nil {
		t.Fatalf("ReassembleChunkManifest: %v", err)
	}
	if !bytes.Equal(got, full) {
		t.Fatalf("got %q, want %q", got, full)
	}

	cm := &blockpb.ContentChunkedManifest{Chunks: []string{"a", "b"}, Sha512: hex.EncodeToString(sum[:])}
	if _, err := ReassembleContentChunked(cm, parts); err != nil {
		t.Fatalf("ReassembleContentChunked: %v", err)
	}
}
