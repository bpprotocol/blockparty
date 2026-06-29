package exchangepb

// Regenerate the exchange wire messages. Requires protoc + protoc-gen-go.
//go:generate protoc --proto_path=../../proto/v1 --go_out=. --go_opt=paths=source_relative exchange.proto
