package store

import (
	"encoding/binary"
	"fmt"

	"github.com/hashicorp/go-memdb"
)

// Badger key prefixes. Each block is stored under two keys in one transaction:
// the canonical block bytes and a small JSON metadata record (which also caches
// the index fields so the in-memory index can be rebuilt without decoding every
// block on startup).
const (
	blockPrefix = "b:"
	metaPrefix  = "m:"
)

// meta is the per-block metadata persisted alongside the block bytes. Author is
// not part of the block envelope (it is bound into the block ID but not stored),
// so it is supplied at ingest time (by the World guard, #31) and persisted here.
type meta struct {
	Audience   string `json:"a"`
	Type       string `json:"t"`
	Author     string `json:"au"`
	Timestamp  int64  `json:"ts"`
	ReceivedAt int64  `json:"r"`
	Blob       bool   `json:"b"` // block data was externalized to the blob store
}

func (m meta) entry(id string) *IndexEntry {
	return &IndexEntry{
		ID:         id,
		Audience:   m.Audience,
		Type:       m.Type,
		Author:     m.Author,
		Timestamp:  m.Timestamp,
		ReceivedAt: m.ReceivedAt,
	}
}

// IndexEntry is a row in the in-memory block index. Values returned from queries
// are shared, read-only views — callers must not mutate them.
type IndexEntry struct {
	ID         string // block ID, hex
	Audience   string // audience_code, hex
	Type       string // type_code, hex
	Author     string // author address, hex ("" if unknown)
	Timestamp  int64  // block timestamp (unix seconds)
	ReceivedAt int64  // when the node accepted the block (unix seconds)
}

const blocksTable = "blocks"

func schema() *memdb.DBSchema {
	return &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			blocksTable: {
				Name: blocksTable,
				Indexes: map[string]*memdb.IndexSchema{
					"id":        {Name: "id", Unique: true, Indexer: &memdb.StringFieldIndex{Field: "ID"}},
					"audience":  {Name: "audience", Indexer: &memdb.StringFieldIndex{Field: "Audience"}},
					"type":      {Name: "type", Indexer: &memdb.StringFieldIndex{Field: "Type"}},
					"author":    {Name: "author", AllowMissing: true, Indexer: &memdb.StringFieldIndex{Field: "Author"}},
					"timestamp": {Name: "timestamp", Indexer: timestampIndexer{}},
				},
			},
		},
	}
}

// timestampIndexer encodes the block timestamp into an order-preserving key so
// memdb LowerBound range scans (ByTimeRange) work correctly, including for
// negative values.
type timestampIndexer struct{}

func (timestampIndexer) FromObject(raw interface{}) (bool, []byte, error) {
	e, ok := raw.(*IndexEntry)
	if !ok {
		return false, nil, fmt.Errorf("timestampIndexer: unexpected type %T", raw)
	}
	return true, encodeInt64(e.Timestamp), nil
}

func (timestampIndexer) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("timestampIndexer: expected 1 arg, got %d", len(args))
	}
	v, ok := args[0].(int64)
	if !ok {
		return nil, fmt.Errorf("timestampIndexer: expected int64, got %T", args[0])
	}
	return encodeInt64(v), nil
}

// encodeInt64 maps an int64 to a big-endian byte key that sorts in numeric
// order (the sign bit is flipped so negatives precede positives).
func encodeInt64(v int64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(v)^(1<<63))
	return b[:]
}
