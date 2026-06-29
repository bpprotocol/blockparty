package store

import (
	"bytes"
	"encoding/hex"
	"io"
	"log/slog"
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// mkBlock builds a minimal valid block. idByte makes the ID unique; the code
// strings become the audience_code/type_code bytes.
func mkBlock(idByte byte, audience, typ string, ts int64, data []byte) *blockpb.Block {
	return &blockpb.Block{
		Version:      1,
		Id:           bytes.Repeat([]byte{idByte}, 32),
		TypeCode:     []byte(typ),
		AudienceCode: []byte(audience),
		Timestamp:    ts,
		Data:         data,
	}
}

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir(), testLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestPutGetRoundTrip(t *testing.T) {
	s := openTemp(t)
	b := mkBlock(0x01, "aud1", "type1", 1000, []byte("hello world"))
	if err := s.Put(&Record{Block: b, Author: "abc123", ReceivedAt: 5}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	id := hex.EncodeToString(b.Id)
	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got.Block.Data, b.Data) {
		t.Errorf("data = %q, want %q", got.Block.Data, b.Data)
	}
	if got.Author != "abc123" {
		t.Errorf("author = %q, want abc123", got.Author)
	}
	if got.ReceivedAt != 5 {
		t.Errorf("receivedAt = %d, want 5", got.ReceivedAt)
	}

	if _, err := s.Get("deadbeef"); err != ErrNotFound {
		t.Errorf("Get(missing) err = %v, want ErrNotFound", err)
	}
}

func TestIndexQueries(t *testing.T) {
	s := openTemp(t)
	blocks := []*blockpb.Block{
		mkBlock(0x01, "audA", "post", 100, []byte("a")),
		mkBlock(0x02, "audA", "react", 200, []byte("b")),
		mkBlock(0x03, "audB", "post", 300, []byte("c")),
	}
	authors := []string{"alice", "alice", "bob"}
	for i, b := range blocks {
		if err := s.Put(&Record{Block: b, Author: authors[i], ReceivedAt: int64(i)}); err != nil {
			t.Fatalf("Put %d: %v", i, err)
		}
	}

	if got, _ := s.ByAudience(hex.EncodeToString([]byte("audA"))); len(got) != 2 {
		t.Errorf("ByAudience(audA) = %d, want 2", len(got))
	}
	if got, _ := s.ByType(hex.EncodeToString([]byte("post"))); len(got) != 2 {
		t.Errorf("ByType(post) = %d, want 2", len(got))
	}
	if got, _ := s.ByAuthor("alice"); len(got) != 2 {
		t.Errorf("ByAuthor(alice) = %d, want 2", len(got))
	}
	if got, _ := s.ByAuthor("bob"); len(got) != 1 {
		t.Errorf("ByAuthor(bob) = %d, want 1", len(got))
	}

	// Time range [150, 350] should catch the second and third blocks, in order.
	got, err := s.ByTimeRange(150, 350)
	if err != nil {
		t.Fatalf("ByTimeRange: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("ByTimeRange = %d entries, want 2", len(got))
	}
	if got[0].Timestamp != 200 || got[1].Timestamp != 300 {
		t.Errorf("ByTimeRange order = [%d, %d], want [200, 300]", got[0].Timestamp, got[1].Timestamp)
	}

	if n, _ := s.Count(); n != 3 {
		t.Errorf("Count = %d, want 3", n)
	}
}

func TestRestartRebuildsIndex(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir, testLogger())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	b := mkBlock(0x09, "audX", "post", 42, []byte("persist me"))
	if err := s.Put(&Record{Block: b, Author: "carol", ReceivedAt: 7}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	s.Close()

	// Reopen: the index must be rebuilt from Badger with no extra Puts.
	s2, err := Open(dir, testLogger())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()

	if n, _ := s2.Count(); n != 1 {
		t.Fatalf("after restart Count = %d, want 1", n)
	}
	if got, _ := s2.ByAuthor("carol"); len(got) != 1 {
		t.Errorf("after restart ByAuthor(carol) = %d, want 1", len(got))
	}
	rec, err := s2.Get(hex.EncodeToString(b.Id))
	if err != nil {
		t.Fatalf("after restart Get: %v", err)
	}
	if !bytes.Equal(rec.Block.Data, b.Data) {
		t.Errorf("after restart data = %q, want %q", rec.Block.Data, b.Data)
	}
}

func TestBlobExternalization(t *testing.T) {
	s := openTemp(t)
	s.blobThreshold = 16 // force externalization for the test

	big := bytes.Repeat([]byte("x"), 1024)
	b := mkBlock(0x0a, "audY", "chunk", 1, big)
	if err := s.Put(&Record{Block: b}); err != nil {
		t.Fatalf("Put: %v", err)
	}
	id := hex.EncodeToString(b.Id)

	// The payload must be on the blob store, not inline in Badger.
	if ok, _ := s.blob.Has(id); !ok {
		t.Error("expected payload to be externalized to the blob store")
	}
	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !bytes.Equal(got.Block.Data, big) {
		t.Errorf("reattached data mismatch: got %d bytes, want %d", len(got.Block.Data), len(big))
	}

	// Delete must remove the blob too.
	if err := s.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if ok, _ := s.blob.Has(id); ok {
		t.Error("expected blob removed after Delete")
	}
	if _, err := s.Get(id); err != ErrNotFound {
		t.Errorf("Get after Delete err = %v, want ErrNotFound", err)
	}
}
