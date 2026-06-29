package blocktypes

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
)

const testWorldPhrase = "bpprotocol.org/v1/test-world"

func TestPayloadRoundTripAllTypes(t *testing.T) {
	// Every core type must instantiate and round-trip through marshal/decode
	// (rpc.render is exercised separately since decoding it is refused).
	samples := map[string]proto.Message{
		TypeContentPost:            &blockpb.ContentPost{Headers: map[string]string{"bpprotocol.org/v1/content-type": "text/plain"}, Body: []byte("hi")},
		TypeContentReaction:        &blockpb.ContentReaction{Target: "blk", Reaction: "👍"},
		TypeContentTagRequest:      &blockpb.ContentTagRequest{Target: "blk", Identity: "addr", Label: "mention"},
		TypeContentTagAccept:       &blockpb.ContentTagAccept{Request: "req"},
		TypeContentChunkedManifest: &blockpb.ContentChunkedManifest{Chunks: []string{"a", "b"}, Sha512: "deadbeef"},
		TypeContentChunkedBlock:    &blockpb.ContentChunkedBlock{Data: []byte{1, 2, 3}},
		TypeContentGallery:         &blockpb.ContentGallery{Title: "g", Blocks: []string{"x"}},
		TypeContentQuote:           &blockpb.ContentQuote{Quote: "orig"},
		TypeConnectRequest:         &blockpb.ConnectRequest{Target: "addr", Nonce: []byte("n")},
		TypeConnectResponse:        &blockpb.ConnectResponse{Request: "id"},
		TypeConnectIdentity:        &blockpb.ConnectIdentity{Name: "Alice"},
		TypeConnectRotate:          &blockpb.ConnectRotate{NewChannel: "code"},
		TypeConnectClose:           &blockpb.ConnectClose{Target: "addr", Reason: "closed"},
		TypeIdentity:               &blockpb.IdentityDeclare{Name: "Alice", PublicKeys: []string{"k"}},
		TypeIdentityBurn:           &blockpb.IdentityBurn{Identity: "addr", BurnNotice: "voluntary"},
		TypeChunkManifest:          &blockpb.ChunkManifest{Chunks: []string{"a"}, Sha512: "x"},
		TypeChunkBlock:             &blockpb.ChunkBlock{Data: []byte{9}},
		TypeRpcRequest:             &blockpb.RpcRequest{Id: "1", Method: "m"},
		TypeRpcResponse:            &blockpb.RpcResponse{Id: "1"},
	}
	for urn, msg := range samples {
		data, err := MarshalPayload(msg)
		if err != nil {
			t.Fatalf("%s marshal: %v", urn, err)
		}
		got, err := DecodePayload(urn, data)
		if err != nil {
			t.Fatalf("%s decode: %v", urn, err)
		}
		if !proto.Equal(got, msg) {
			t.Errorf("%s round-trip mismatch", urn)
		}
	}
}

func TestResolverMapsTypeCodes(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	r := NewResolver(w)
	code := derive.GetTypeCode(w, TypeContentPost)
	urn, ok := r.TypeURN([]byte(code))
	if !ok || urn != TypeContentPost {
		t.Fatalf("resolver TypeURN = (%q, %v), want content.post", urn, ok)
	}
	if _, ok := r.TypeURN([]byte("not-a-real-code_")); ok {
		t.Error("resolver matched an unknown code")
	}
}

func TestResolverDecode(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	r := NewResolver(w)
	post := &blockpb.ContentPost{Body: []byte("hello")}
	data, _ := MarshalPayload(post)
	b := &blockpb.Block{TypeCode: []byte(derive.GetTypeCode(w, TypeContentPost))}

	urn, msg, err := r.Decode(b, data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if urn != TypeContentPost {
		t.Fatalf("urn = %q", urn)
	}
	if got, ok := msg.(*blockpb.ContentPost); !ok || string(got.Body) != "hello" {
		t.Fatalf("decoded payload wrong: %v", msg)
	}
}

func TestRenderRefusedOnDecode(t *testing.T) {
	// A render payload must never be accepted from a transport.
	data, _ := MarshalPayload(&blockpb.RpcRender{Id: "r", Method: "render.x"})
	if _, err := DecodePayload(TypeRpcRender, data); err != ErrRenderNotTransportable {
		t.Fatalf("DecodePayload(render) = %v, want ErrRenderNotTransportable", err)
	}

	w := derive.OpenWorld(testWorldPhrase)
	r := NewResolver(w)
	b := &blockpb.Block{TypeCode: []byte(derive.GetTypeCode(w, TypeRpcRender))}
	if _, _, err := r.Decode(b, data); err != ErrRenderNotTransportable {
		t.Fatalf("Resolver.Decode(render) = %v, want ErrRenderNotTransportable", err)
	}
}

func TestDecodeUnknownType(t *testing.T) {
	if _, err := DecodePayload("example.com/v1/types/unknown", nil); err != ErrUnknownType {
		t.Fatalf("DecodePayload(unknown) = %v, want ErrUnknownType", err)
	}
}
