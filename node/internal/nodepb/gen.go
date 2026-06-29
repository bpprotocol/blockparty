package nodepb

// Regenerate the Connect service from the protobuf definition. Requires protoc,
// protoc-gen-go, and protoc-gen-connect-go on PATH.
//go:generate protoc --proto_path=../../proto/v1 --go_out=. --go_opt=paths=source_relative --connect-go_out=. --connect-go_opt=paths=source_relative node.proto
