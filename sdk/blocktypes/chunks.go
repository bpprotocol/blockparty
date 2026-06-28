package blocktypes

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
)

var (
	// ErrMissingChunk is returned when a manifest references a chunk that is not
	// available for reassembly.
	ErrMissingChunk = errors.New("blocktypes: missing chunk")
	// ErrChecksum is returned when reassembled bytes do not match the manifest's
	// SHA-512.
	ErrChecksum = errors.New("blocktypes: reassembled checksum mismatch")
)

// ReassembleChunks concatenates the chunk payloads named by chunkIDs (in order)
// and verifies the result against sha512Hex (the manifest's hex SHA-512).
// dataByID maps a chunk block ID to that chunk's payload bytes.
func ReassembleChunks(chunkIDs []string, dataByID map[string][]byte, sha512Hex string) ([]byte, error) {
	var buf []byte
	for _, id := range chunkIDs {
		d, ok := dataByID[id]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrMissingChunk, id)
		}
		buf = append(buf, d...)
	}
	sum := sha512.Sum512(buf)
	if hex.EncodeToString(sum[:]) != sha512Hex {
		return nil, ErrChecksum
	}
	return buf, nil
}

// ReassembleChunkManifest reassembles a transport-layer chunk.manifest.
func ReassembleChunkManifest(m *blockpb.ChunkManifest, dataByID map[string][]byte) ([]byte, error) {
	return ReassembleChunks(m.Chunks, dataByID, m.Sha512)
}

// ReassembleContentChunked reassembles an application-layer
// content.chunked.manifest (its headers are handled separately by the caller).
func ReassembleContentChunked(m *blockpb.ContentChunkedManifest, dataByID map[string][]byte) ([]byte, error) {
	return ReassembleChunks(m.Chunks, dataByID, m.Sha512)
}
