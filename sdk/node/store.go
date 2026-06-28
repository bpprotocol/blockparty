package node

import (
	"os"
	"path/filepath"

	"github.com/bpprotocol/blockparty/sdk/block"
	"github.com/bpprotocol/blockparty/sdk/blockpb"
)

// BlockExt is the file extension for serialized blocks on the filesystem
// transport.
const BlockExt = ".block"

// FileStore is a filesystem ("sneakernet") transport: blocks are written as
// individual files named by their hex ID and read back by scanning a directory.
// It is the simplest transport that demonstrates BlockParty's "moving a block
// is just moving bytes" model.
type FileStore struct {
	Dir string
}

// Write serializes a block and writes it as <id>.block in the store directory,
// creating the directory if needed. It returns the file path.
func (s FileStore) Write(b *blockpb.Block) (string, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", err
	}
	data, err := block.Encode(b)
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.Dir, block.IDHex(b)+BlockExt)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// ReadAll reads and decodes every *.block file in the store directory. A
// missing directory yields no blocks (not an error).
func (s FileStore) ReadAll() ([]*blockpb.Block, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var blocks []*blockpb.Block
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != BlockExt {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.Dir, e.Name()))
		if err != nil {
			return nil, err
		}
		b, err := block.Decode(data)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, b)
	}
	return blocks, nil
}
