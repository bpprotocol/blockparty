package block

import (
	"bytes"
	"testing"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
	"github.com/bpprotocol/blockparty/sdk/derive"
	"github.com/bpprotocol/blockparty/sdk/identity"
)

const (
	testWorldPhrase = "bpprotocol.org/v1/test-world"
	testType        = "bpprotocol.org/v1/types/content.post"
	testAudience    = "bpprotocol.org/v1/audience/public-1"
	testTimestamp   = int64(1700000000)
	cosignType      = "bpprotocol.org/v1/cosign.endorse"
)

type fixture struct {
	world    derive.World
	author   identity.Identity
	cosigner identity.Identity
	typeCode derive.Code
	audCode  derive.Code
}

func newFixture() fixture {
	w := derive.OpenWorld(testWorldPhrase)
	return fixture{
		world:    w,
		author:   identity.OpenIdentity(w, "author-pass"),
		cosigner: identity.OpenIdentity(w, "cosigner-pass"),
		typeCode: derive.GetTypeCode(w, testType),
		audCode:  derive.GetAudienceCode(w, testAudience),
	}
}

func (f fixture) signed(data []byte) *blockpb.Block {
	b := New(f.author.Address, f.typeCode, f.audCode, testTimestamp, data)
	Sign(b, f.world, f.author.MLDSA)
	return b
}

func TestSignAndVerify(t *testing.T) {
	f := newFixture()
	b := f.signed([]byte("hello"))
	if err := Verify(b, f.world, f.author.MLDSA.Public); err != nil {
		t.Fatalf("Verify on a freshly signed block: %v", err)
	}
}

func TestNewSetsContentBindingID(t *testing.T) {
	f := newFixture()
	a := New(f.author.Address, f.typeCode, f.audCode, testTimestamp, []byte("hello"))
	b := New(f.author.Address, f.typeCode, f.audCode, testTimestamp, []byte("hello!"))
	if bytes.Equal(a.Id, b.Id) {
		t.Fatal("block ID did not change with payload")
	}
	if IDHex(a) == "" || len(a.Id) != 32 {
		t.Fatalf("unexpected ID: hex=%q len=%d", IDHex(a), len(a.Id))
	}
}

func TestVerifyMissingSignatures(t *testing.T) {
	f := newFixture()
	b := New(f.author.Address, f.typeCode, f.audCode, testTimestamp, []byte("hi"))
	if err := Verify(b, f.world, f.author.MLDSA.Public); err != ErrMissingSignatures {
		t.Fatalf("Verify unsigned = %v, want ErrMissingSignatures", err)
	}
}

func TestVerifyWrongKeys(t *testing.T) {
	f := newFixture()
	b := f.signed([]byte("hello"))

	stranger := identity.OpenIdentity(f.world, "stranger")
	if err := Verify(b, f.world, stranger.MLDSA.Public); err != ErrAuthorSignature {
		t.Fatalf("Verify with wrong author key = %v, want ErrAuthorSignature", err)
	}
	otherWorld := derive.OpenWorld("a-different-world")
	if err := Verify(b, otherWorld, f.author.MLDSA.Public); err != ErrWorldSignature {
		t.Fatalf("Verify with wrong world = %v, want ErrWorldSignature", err)
	}
}

func TestTamperingAnyFieldFailsVerification(t *testing.T) {
	f := newFixture()
	cases := []struct {
		name   string
		mutate func(*blockpb.Block)
	}{
		{"version", func(b *blockpb.Block) { b.Version++ }},
		{"id", func(b *blockpb.Block) { b.Id[0] ^= 0xff }},
		{"type_code", func(b *blockpb.Block) { b.TypeCode[0] ^= 0xff }},
		{"audience_code", func(b *blockpb.Block) { b.AudienceCode[0] ^= 0xff }},
		{"timestamp", func(b *blockpb.Block) { b.Timestamp++ }},
		{"data", func(b *blockpb.Block) { b.Data[0] ^= 0xff }},
		{"world_sig", func(b *blockpb.Block) { b.Sigs.World[0] ^= 0xff }},
		{"author_sig", func(b *blockpb.Block) { b.Sigs.Author[0] ^= 0xff }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := f.signed([]byte("hello"))
			c.mutate(b)
			if err := Verify(b, f.world, f.author.MLDSA.Public); err == nil {
				t.Fatalf("tampering %s still verified", c.name)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	f := newFixture()
	b := f.signed([]byte("hello"))
	AddCoSignature(b, cosignType, f.cosigner.MLDSA)

	enc, err := Encode(b)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	b2, err := Decode(enc)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	// Canonical encoding is stable.
	enc2, err := Encode(b2)
	if err != nil {
		t.Fatalf("re-Encode: %v", err)
	}
	if !bytes.Equal(enc, enc2) {
		t.Fatal("re-encoding a decoded block produced different bytes")
	}
	// Signatures survive the round-trip.
	if err := Verify(b2, f.world, f.author.MLDSA.Public); err != nil {
		t.Fatalf("Verify after round-trip: %v", err)
	}
	if cos := CoSignatures(b2); len(cos) != 1 || !VerifyCoSignature(b2, cos[0], f.cosigner.MLDSA.Public) {
		t.Fatal("co-signature did not survive the round-trip")
	}
}

func TestCoSignatureAddVerify(t *testing.T) {
	f := newFixture()
	b := f.signed([]byte("hello"))
	AddCoSignature(b, cosignType, f.cosigner.MLDSA)

	cos := CoSignatures(b)
	if len(cos) != 1 || cos[0].Type != cosignType {
		t.Fatalf("unexpected co-signatures: %+v", cos)
	}
	if !VerifyCoSignature(b, cos[0], f.cosigner.MLDSA.Public) {
		t.Fatal("valid co-signature failed to verify")
	}
	// Wrong signer key.
	if VerifyCoSignature(b, cos[0], f.author.MLDSA.Public) {
		t.Fatal("co-signature verified under the wrong signer key")
	}
	// The required signatures are unaffected by the presence of a co-signature.
	if err := Verify(b, f.world, f.author.MLDSA.Public); err != nil {
		t.Fatalf("Verify with a co-signature present: %v", err)
	}
}

func TestBadCoSignatureIgnored(t *testing.T) {
	f := newFixture()
	b := f.signed([]byte("hello"))
	// A garbage co-signature from nobody must not invalidate the block.
	b.Sigs.Extra = append(b.Sigs.Extra, &blockpb.ExtraSignature{
		Type: "evil.example/forge",
		Sig:  []byte("not a real signature"),
	})
	if err := Verify(b, f.world, f.author.MLDSA.Public); err != nil {
		t.Fatalf("a bogus co-signature invalidated the block: %v", err)
	}
	if VerifyCoSignature(b, b.Sigs.Extra[0], f.cosigner.MLDSA.Public) {
		t.Fatal("a bogus co-signature verified")
	}
}

func TestTamperedCoSignatureFailsButBlockValid(t *testing.T) {
	f := newFixture()
	b := f.signed([]byte("hello"))
	AddCoSignature(b, cosignType, f.cosigner.MLDSA)
	b.Sigs.Extra[0].Sig[0] ^= 0xff

	if VerifyCoSignature(b, b.Sigs.Extra[0], f.cosigner.MLDSA.Public) {
		t.Fatal("tampered co-signature verified")
	}
	if err := Verify(b, f.world, f.author.MLDSA.Public); err != nil {
		t.Fatalf("tampered co-signature invalidated the block: %v", err)
	}
}
