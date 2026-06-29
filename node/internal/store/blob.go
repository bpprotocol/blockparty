package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// blobStore keeps large payloads on the filesystem, keyed by block ID and
// sharded by the first two hex characters of the ID. It exists so very large
// blocks / chunk.* payloads don't bloat the key/value store; the block envelope
// stays in Badger while its oversized data lives here, referenced by ID.
type blobStore struct {
	dir string
}

func newBlobStore(dir string) (*blobStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("blob: create dir: %w", err)
	}
	return &blobStore{dir: dir}, nil
}

func (b *blobStore) path(id string) (string, error) {
	if len(id) < 2 {
		return "", fmt.Errorf("blob: id too short: %q", id)
	}
	return filepath.Join(b.dir, id[:2], id), nil
}

// Put writes data atomically (temp file + rename) under the block ID.
func (b *blobStore) Put(id string, data []byte) error {
	p, err := b.path(id)
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+id+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil {
		_ = os.Remove(tmpName)
		return werr
	}
	if cerr != nil {
		_ = os.Remove(tmpName)
		return cerr
	}
	return os.Rename(tmpName, p)
}

// Get reads the blob for an ID. Returns ErrNotFound if absent.
func (b *blobStore) Get(id string) ([]byte, error) {
	p, err := b.path(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return data, err
}

// Has reports whether a blob exists for an ID.
func (b *blobStore) Has(id string) (bool, error) {
	p, err := b.path(id)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Delete removes the blob for an ID; absence is not an error.
func (b *blobStore) Delete(id string) error {
	p, err := b.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
