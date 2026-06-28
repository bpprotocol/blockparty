package blocktypes

import (
	"errors"

	"google.golang.org/protobuf/proto"

	"github.com/bpprotocol/blockparty/sdk/blockpb"
	"github.com/bpprotocol/blockparty/sdk/derive"
)

// Canonical block-type URNs under the bpprotocol.org/v1/types/ namespace.
const (
	TypeContentPost            = "bpprotocol.org/v1/types/content.post"
	TypeContentReaction        = "bpprotocol.org/v1/types/content.reaction"
	TypeContentTagRequest      = "bpprotocol.org/v1/types/content.tag.request"
	TypeContentTagAccept       = "bpprotocol.org/v1/types/content.tag.accept"
	TypeContentChunkedManifest = "bpprotocol.org/v1/types/content.chunked.manifest"
	TypeContentChunkedBlock    = "bpprotocol.org/v1/types/content.chunked.block"
	TypeContentGallery         = "bpprotocol.org/v1/types/content.gallery"
	TypeContentQuote           = "bpprotocol.org/v1/types/content.quote"

	TypeConnectRequest  = "bpprotocol.org/v1/types/connect.request"
	TypeConnectResponse = "bpprotocol.org/v1/types/connect.response"
	TypeConnectIdentity = "bpprotocol.org/v1/types/connect.identity"
	TypeConnectRotate   = "bpprotocol.org/v1/types/connect.rotate"
	TypeConnectClose    = "bpprotocol.org/v1/types/connect.close"

	TypeIdentity     = "bpprotocol.org/v1/types/identity"
	TypeIdentityBurn = "bpprotocol.org/v1/types/identity.burn"

	TypeChunkManifest = "bpprotocol.org/v1/types/chunk.manifest"
	TypeChunkBlock    = "bpprotocol.org/v1/types/chunk.block"

	TypeRpcRequest  = "bpprotocol.org/v1/types/rpc.request"
	TypeRpcResponse = "bpprotocol.org/v1/types/rpc.response"
	TypeRpcRender   = "bpprotocol.org/v1/types/rpc.render"
)

var (
	// ErrUnknownType is returned when a type code or URN is not a known core type.
	ErrUnknownType = errors.New("blocktypes: unknown block type")
	// ErrRenderNotTransportable is returned when an rpc.render block is decoded
	// from a transport; rpc.render is local-only and must never be transmitted.
	ErrRenderNotTransportable = errors.New("blocktypes: rpc.render is local-only and must not be transmitted")
)

// coreTypes lists every core block type with a constructor for its payload.
var coreTypes = []struct {
	URN string
	New func() proto.Message
}{
	{TypeContentPost, func() proto.Message { return &blockpb.ContentPost{} }},
	{TypeContentReaction, func() proto.Message { return &blockpb.ContentReaction{} }},
	{TypeContentTagRequest, func() proto.Message { return &blockpb.ContentTagRequest{} }},
	{TypeContentTagAccept, func() proto.Message { return &blockpb.ContentTagAccept{} }},
	{TypeContentChunkedManifest, func() proto.Message { return &blockpb.ContentChunkedManifest{} }},
	{TypeContentChunkedBlock, func() proto.Message { return &blockpb.ContentChunkedBlock{} }},
	{TypeContentGallery, func() proto.Message { return &blockpb.ContentGallery{} }},
	{TypeContentQuote, func() proto.Message { return &blockpb.ContentQuote{} }},
	{TypeConnectRequest, func() proto.Message { return &blockpb.ConnectRequest{} }},
	{TypeConnectResponse, func() proto.Message { return &blockpb.ConnectResponse{} }},
	{TypeConnectIdentity, func() proto.Message { return &blockpb.ConnectIdentity{} }},
	{TypeConnectRotate, func() proto.Message { return &blockpb.ConnectRotate{} }},
	{TypeConnectClose, func() proto.Message { return &blockpb.ConnectClose{} }},
	{TypeIdentity, func() proto.Message { return &blockpb.IdentityDeclare{} }},
	{TypeIdentityBurn, func() proto.Message { return &blockpb.IdentityBurn{} }},
	{TypeChunkManifest, func() proto.Message { return &blockpb.ChunkManifest{} }},
	{TypeChunkBlock, func() proto.Message { return &blockpb.ChunkBlock{} }},
	{TypeRpcRequest, func() proto.Message { return &blockpb.RpcRequest{} }},
	{TypeRpcResponse, func() proto.Message { return &blockpb.RpcResponse{} }},
	{TypeRpcRender, func() proto.Message { return &blockpb.RpcRender{} }},
}

var payloadFactory = func() map[string]func() proto.Message {
	m := make(map[string]func() proto.Message, len(coreTypes))
	for _, ct := range coreTypes {
		m[ct.URN] = ct.New
	}
	return m
}()

// MarshalPayload serializes a payload message to bytes for placement in a
// block's data field (before any encryption).
func MarshalPayload(m proto.Message) ([]byte, error) {
	return proto.MarshalOptions{Deterministic: true}.Marshal(m)
}

// DecodePayload instantiates and unmarshals the payload for a known type URN.
// It refuses rpc.render (ErrRenderNotTransportable) since render blocks are
// local-only and must never arrive over a transport.
func DecodePayload(typeURN string, plaintext []byte) (proto.Message, error) {
	if typeURN == TypeRpcRender {
		return nil, ErrRenderNotTransportable
	}
	factory, ok := payloadFactory[typeURN]
	if !ok {
		return nil, ErrUnknownType
	}
	m := factory()
	if err := proto.Unmarshal(plaintext, m); err != nil {
		return nil, err
	}
	return m, nil
}

// Resolver maps a World's type codes back to their canonical URNs, so an
// incoming block's type_code can be dispatched to the right payload type.
type Resolver struct {
	byCode map[string]string // code hex -> URN
}

// NewResolver precomputes the type codes for all core types in a World.
func NewResolver(w derive.World) *Resolver {
	r := &Resolver{byCode: make(map[string]string, len(coreTypes))}
	for _, ct := range coreTypes {
		r.byCode[derive.GetTypeCode(w, ct.URN).Hex()] = ct.URN
	}
	return r
}

// TypeURN returns the URN for a raw type code, and whether it is a known core type.
func (r *Resolver) TypeURN(code []byte) (string, bool) {
	urn, ok := r.byCode[derive.Code(code).Hex()]
	return urn, ok
}

// Decode resolves a block's type and unmarshals its (already-decrypted)
// payload, returning the URN and the message. It enforces the rpc.render guard.
func (r *Resolver) Decode(b *blockpb.Block, plaintext []byte) (string, proto.Message, error) {
	urn, ok := r.TypeURN(b.TypeCode)
	if !ok {
		return "", nil, ErrUnknownType
	}
	m, err := DecodePayload(urn, plaintext)
	return urn, m, err
}
