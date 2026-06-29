// Package store is the node's local persistence layer (#30): a durable BadgerDB
// block store, an in-memory hashicorp/go-memdb index over the hot fields
// (audience_code, type_code, author, timestamp), and a filesystem blob store for
// oversized payloads. Badger is the source of truth; the memdb index is derived
// and rebuilt from Badger on startup.
package store

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/go-memdb"
	"google.golang.org/protobuf/proto"

	"github.com/bpprotocol/blockparty/implementations/go/block"
	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
)

// ErrNotFound is returned when a block is not present.
var ErrNotFound = errors.New("store: block not found")

// defaultBlobThreshold is the block data size above which the payload is
// externalized to the filesystem blob store.
const defaultBlobThreshold = 1 << 20 // 1 MiB

// Record is a stored block together with the metadata the node tracks for it.
type Record struct {
	Block      *blockpb.Block
	Author     string // hex author address; "" if not yet resolved (#31)
	ReceivedAt int64  // unix seconds the node accepted the block
}

// Store is the node's block persistence layer. It is safe for concurrent use.
type Store struct {
	db            *badger.DB
	mem           *memdb.MemDB
	blob          *blobStore
	blobThreshold int
	log           *slog.Logger
}

// Open opens (creating if necessary) the store rooted at dataDir, laying out
// Badger under <dataDir>/badger and blobs under <dataDir>/blobs, then rebuilds
// the in-memory index from Badger.
func Open(dataDir string, log *slog.Logger) (*Store, error) {
	if dataDir == "" {
		return nil, errors.New("store: empty data dir")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(filepath.Join(dataDir, "badger")).WithLogger(badgerLogger{log})
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("store: open badger: %w", err)
	}

	mem, err := memdb.NewMemDB(schema())
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: build index: %w", err)
	}

	blob, err := newBlobStore(filepath.Join(dataDir, "blobs"))
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	s := &Store{db: db, mem: mem, blob: blob, blobThreshold: defaultBlobThreshold, log: log}
	if err := s.rebuildIndex(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: rebuild index: %w", err)
	}
	return s, nil
}

// Close releases the store.
func (s *Store) Close() error { return s.db.Close() }

// rebuildIndex repopulates the memdb index from the persisted metadata records.
func (s *Store) rebuildIndex() error {
	txn := s.mem.Txn(true)
	defer txn.Abort()

	err := s.db.View(func(btx *badger.Txn) error {
		it := btx.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(metaPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			id := strings.TrimPrefix(string(item.Key()), metaPrefix)
			var m meta
			if err := item.Value(func(v []byte) error { return json.Unmarshal(v, &m) }); err != nil {
				return fmt.Errorf("decode meta for %s: %w", id, err)
			}
			if err := txn.Insert(blocksTable, m.entry(id)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	txn.Commit()
	return nil
}

// Put stores (or replaces) a block and indexes it. Oversized data is offloaded
// to the blob store. Badger is written first (durable, source of truth), then
// the in-memory index — which is in any case rebuildable from Badger.
func (s *Store) Put(rec *Record) error {
	if rec == nil || rec.Block == nil {
		return errors.New("store: nil record")
	}
	id := block.IDHex(rec.Block)
	if id == "" {
		return errors.New("store: block has no id")
	}

	toStore := rec.Block
	blobbed := false
	if len(rec.Block.Data) > s.blobThreshold {
		if err := s.blob.Put(id, rec.Block.Data); err != nil {
			return fmt.Errorf("store: blob put: %w", err)
		}
		clone := proto.Clone(rec.Block).(*blockpb.Block)
		clone.Data = nil
		toStore = clone
		blobbed = true
	}

	blockBytes, err := block.Encode(toStore)
	if err != nil {
		if blobbed {
			_ = s.blob.Delete(id)
		}
		return fmt.Errorf("store: encode block: %w", err)
	}

	m := meta{
		Audience:   hex.EncodeToString(rec.Block.AudienceCode),
		Type:       hex.EncodeToString(rec.Block.TypeCode),
		Author:     rec.Author,
		Timestamp:  rec.Block.Timestamp,
		ReceivedAt: rec.ReceivedAt,
		Blob:       blobbed,
	}
	metaBytes, err := json.Marshal(m)
	if err != nil {
		if blobbed {
			_ = s.blob.Delete(id)
		}
		return fmt.Errorf("store: encode meta: %w", err)
	}

	if err := s.db.Update(func(btx *badger.Txn) error {
		if err := btx.Set([]byte(blockPrefix+id), blockBytes); err != nil {
			return err
		}
		return btx.Set([]byte(metaPrefix+id), metaBytes)
	}); err != nil {
		if blobbed {
			_ = s.blob.Delete(id)
		}
		return fmt.Errorf("store: badger write: %w", err)
	}

	txn := s.mem.Txn(true)
	if err := txn.Insert(blocksTable, m.entry(id)); err != nil {
		txn.Abort()
		return fmt.Errorf("store: index insert: %w", err)
	}
	txn.Commit()
	return nil
}

// Get returns the block (with any externalized data reattached) for an ID, or
// ErrNotFound.
func (s *Store) Get(id string) (*Record, error) {
	var (
		blockBytes []byte
		m          meta
		found      bool
	)
	err := s.db.View(func(btx *badger.Txn) error {
		bi, err := btx.Get([]byte(blockPrefix + id))
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if blockBytes, err = bi.ValueCopy(nil); err != nil {
			return err
		}
		mi, err := btx.Get([]byte(metaPrefix + id))
		if err != nil {
			return err
		}
		if err := mi.Value(func(v []byte) error { return json.Unmarshal(v, &m) }); err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("store: get %s: %w", id, err)
	}
	if !found {
		return nil, ErrNotFound
	}

	b, err := block.Decode(blockBytes)
	if err != nil {
		return nil, fmt.Errorf("store: decode block %s: %w", id, err)
	}
	if m.Blob {
		data, err := s.blob.Get(id)
		if err != nil {
			return nil, fmt.Errorf("store: blob get %s: %w", id, err)
		}
		b.Data = data
	}
	return &Record{Block: b, Author: m.Author, ReceivedAt: m.ReceivedAt}, nil
}

// Has reports whether a block with the given ID is stored.
func (s *Store) Has(id string) (bool, error) {
	txn := s.mem.Txn(false)
	defer txn.Abort()
	raw, err := txn.First(blocksTable, "id", id)
	if err != nil {
		return false, err
	}
	return raw != nil, nil
}

// Delete removes a block, its metadata, any externalized blob, and its index
// row. Absence is not an error (pruning/GC hook).
func (s *Store) Delete(id string) error {
	if err := s.db.Update(func(btx *badger.Txn) error {
		if err := btx.Delete([]byte(blockPrefix + id)); err != nil {
			return err
		}
		return btx.Delete([]byte(metaPrefix + id))
	}); err != nil {
		return fmt.Errorf("store: delete %s: %w", id, err)
	}
	if err := s.blob.Delete(id); err != nil {
		return fmt.Errorf("store: delete blob %s: %w", id, err)
	}

	txn := s.mem.Txn(true)
	raw, err := txn.First(blocksTable, "id", id)
	if err != nil {
		txn.Abort()
		return err
	}
	if raw != nil {
		if err := txn.Delete(blocksTable, raw); err != nil {
			txn.Abort()
			return err
		}
	}
	txn.Commit()
	return nil
}

// ByAudience returns index entries for a hex audience_code.
func (s *Store) ByAudience(audienceHex string) ([]*IndexEntry, error) {
	return s.queryEqual("audience", audienceHex)
}

// ByType returns index entries for a hex type_code.
func (s *Store) ByType(typeHex string) ([]*IndexEntry, error) {
	return s.queryEqual("type", typeHex)
}

// ByAuthor returns index entries for a hex author address.
func (s *Store) ByAuthor(authorHex string) ([]*IndexEntry, error) {
	return s.queryEqual("author", authorHex)
}

// ByTimeRange returns index entries whose timestamp falls in [from, to]
// inclusive, in ascending timestamp order.
func (s *Store) ByTimeRange(from, to int64) ([]*IndexEntry, error) {
	txn := s.mem.Txn(false)
	defer txn.Abort()
	it, err := txn.LowerBound(blocksTable, "timestamp", from)
	if err != nil {
		return nil, err
	}
	var out []*IndexEntry
	for obj := it.Next(); obj != nil; obj = it.Next() {
		e := obj.(*IndexEntry)
		if e.Timestamp > to {
			break
		}
		out = append(out, e)
	}
	return out, nil
}

// Count returns the number of indexed blocks.
func (s *Store) Count() (int, error) {
	txn := s.mem.Txn(false)
	defer txn.Abort()
	it, err := txn.Get(blocksTable, "id")
	if err != nil {
		return 0, err
	}
	n := 0
	for obj := it.Next(); obj != nil; obj = it.Next() {
		n++
	}
	return n, nil
}

func (s *Store) queryEqual(index, arg string) ([]*IndexEntry, error) {
	txn := s.mem.Txn(false)
	defer txn.Abort()
	it, err := txn.Get(blocksTable, index, arg)
	if err != nil {
		return nil, err
	}
	var out []*IndexEntry
	for obj := it.Next(); obj != nil; obj = it.Next() {
		out = append(out, obj.(*IndexEntry))
	}
	return out, nil
}

// badgerLogger adapts Badger's logger to slog, routing its chatty Info output to
// debug level.
type badgerLogger struct{ log *slog.Logger }

func (b badgerLogger) Errorf(f string, a ...interface{}) {
	b.log.Error(strings.TrimSpace(fmt.Sprintf(f, a...)))
}
func (b badgerLogger) Warningf(f string, a ...interface{}) {
	b.log.Warn(strings.TrimSpace(fmt.Sprintf(f, a...)))
}
func (b badgerLogger) Infof(f string, a ...interface{}) {
	b.log.Debug(strings.TrimSpace(fmt.Sprintf(f, a...)))
}
func (b badgerLogger) Debugf(f string, a ...interface{}) {
	b.log.Debug(strings.TrimSpace(fmt.Sprintf(f, a...)))
}
