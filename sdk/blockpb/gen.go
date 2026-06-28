// Package blockpb contains the protobuf-generated types for BlockParty blocks
// and block-type payloads, generated from protocol/proto/v1.
//
// Regenerate with `go generate ./blockpb` (requires protoc + protoc-gen-go):
//
//go:generate protoc --proto_path=../../protocol/proto/v1 --go_out=. --go_opt=paths=source_relative block.proto block_types.proto
package blockpb
