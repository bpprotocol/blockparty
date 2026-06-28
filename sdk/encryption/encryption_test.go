package encryption

import (
	"bytes"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/block"
	"github.com/bpprotocol/blockparty/sdk/blockpb"
	"github.com/bpprotocol/blockparty/sdk/crypto"
	"github.com/bpprotocol/blockparty/sdk/derive"
	"github.com/bpprotocol/blockparty/sdk/identity"
)

const (
	testWorldPhrase = "bpprotocol.org/v1/test-world"
	testType        = "bpprotocol.org/v1/types/content.post"
	testAudience    = "bpprotocol.org/v1/audience/public-1"
	testTimestamp   = int64(1700000000)
)

type env struct {
	secret   []byte
	typeCode []byte
	audCode  []byte
	world    derive.World
}

func newEnv() env {
	w := derive.OpenWorld(testWorldPhrase)
	return env{
		secret:   crypto.Keccak256([]byte("bpprotocol.org/v1/test-secret")),
		typeCode: []byte(derive.GetTypeCode(w, testType)),
		audCode:  []byte(derive.GetAudienceCode(w, testAudience)),
		world:    w,
	}
}

// encBlock builds a minimal block whose data is plaintext encrypted to the
// audience (no signatures; signature flow is exercised in the integration test).
func (e env) encBlock(t *testing.T, plaintext []byte) *blockpb.Block {
	t.Helper()
	data, err := EncryptForBlock(e.secret, 1, e.typeCode, e.audCode, testTimestamp, plaintext)
	if err != nil {
		t.Fatalf("EncryptForBlock: %v", err)
	}
	return &blockpb.Block{
		Version:      1,
		TypeCode:     e.typeCode,
		AudienceCode: e.audCode,
		Timestamp:    testTimestamp,
		Data:         data,
	}
}

func TestRoundTrip(t *testing.T) {
	e := newEnv()
	plaintext := []byte("nothing can stop the signal")
	b := e.encBlock(t, plaintext)

	if len(b.Data) <= NonceSize {
		t.Fatalf("data too short: %d", len(b.Data))
	}
	got, err := DecryptData(e.secret, b)
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("round-trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestWrongSecretFails(t *testing.T) {
	e := newEnv()
	b := e.encBlock(t, []byte("secret message"))
	wrong := crypto.Keccak256([]byte("not the right secret"))
	if _, err := DecryptData(wrong, b); err != ErrDecrypt {
		t.Fatalf("DecryptData with wrong secret = %v, want ErrDecrypt", err)
	}
}

func TestTamperingAADMetadataFailsDecryption(t *testing.T) {
	e := newEnv()
	cases := []struct {
		name   string
		mutate func(*blockpb.Block)
	}{
		{"version", func(b *blockpb.Block) { b.Version++ }},
		{"type_code", func(b *blockpb.Block) { b.TypeCode[0] ^= 0xff }},
		{"audience_code", func(b *blockpb.Block) { b.AudienceCode[0] ^= 0xff }},
		{"timestamp", func(b *blockpb.Block) { b.Timestamp++ }},
		{"ciphertext", func(b *blockpb.Block) { b.Data[len(b.Data)-1] ^= 0xff }},
		{"nonce", func(b *blockpb.Block) { b.Data[0] ^= 0xff }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := e.encBlock(t, []byte("hello"))
			c.mutate(b)
			if _, err := DecryptData(e.secret, b); err == nil {
				t.Fatalf("tampering %s still decrypted", c.name)
			}
		})
	}
}

func TestShortData(t *testing.T) {
	if _, err := DecryptData(make([]byte, 32), &blockpb.Block{Data: []byte("short")}); err != ErrShortData {
		t.Fatalf("DecryptData on short data = %v, want ErrShortData", err)
	}
}

func TestContentKeyDeterministicAndSensitive(t *testing.T) {
	secret := crypto.Keccak256([]byte("s"))
	code := []byte("0123456789abcdef")
	nonce := bytes.Repeat([]byte{1}, NonceSize)

	k := ContentKey(secret, code, nonce)
	if len(k) != 32 {
		t.Fatalf("content key length = %d, want 32", len(k))
	}
	if !bytes.Equal(k, ContentKey(secret, code, nonce)) {
		t.Fatal("ContentKey not deterministic")
	}
	// Distinct per nonce, code, and secret.
	if bytes.Equal(k, ContentKey(secret, code, bytes.Repeat([]byte{2}, NonceSize))) {
		t.Error("ContentKey not nonce-sensitive")
	}
	if bytes.Equal(k, ContentKey(secret, []byte("fedcba9876543210"), nonce)) {
		t.Error("ContentKey not audience-code-sensitive")
	}
	if bytes.Equal(k, ContentKey(crypto.Keccak256([]byte("s2")), code, nonce)) {
		t.Error("ContentKey not secret-sensitive")
	}
}

func TestNonceVariesPerEncryption(t *testing.T) {
	e := newEnv()
	a := e.encBlock(t, []byte("same"))
	b := e.encBlock(t, []byte("same"))
	if bytes.Equal(a.Data[:NonceSize], b.Data[:NonceSize]) {
		t.Fatal("two encryptions reused the same nonce")
	}
	if bytes.Equal(a.Data, b.Data) {
		t.Fatal("two encryptions of the same plaintext produced identical data")
	}
}

// TestSignedEncryptedBlock exercises the full flow: encrypt the payload, build
// and sign the block, verify signatures, then decrypt.
func TestSignedEncryptedBlock(t *testing.T) {
	e := newEnv()
	author := identity.OpenIdentity(e.world, "author-pass")
	plaintext := []byte("audience-scoped content")

	data, err := EncryptForBlock(e.secret, block.Version, e.typeCode, e.audCode, testTimestamp, plaintext)
	if err != nil {
		t.Fatalf("EncryptForBlock: %v", err)
	}
	b := block.New(author.Address, derive.Code(e.typeCode), derive.Code(e.audCode), testTimestamp, data)
	block.Sign(b, e.world, author.Dilithium)

	if err := block.Verify(b, e.world, author.Dilithium.Public); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	got, err := DecryptData(e.secret, b)
	if err != nil {
		t.Fatalf("DecryptData: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted %q, want %q", got, plaintext)
	}
}
