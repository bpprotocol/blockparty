module github.com/bpprotocol/blockparty/node

go 1.25.4

require github.com/bpprotocol/blockparty/implementations/go v0.0.0

require (
	github.com/cloudflare/circl v1.6.4 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

replace github.com/bpprotocol/blockparty/implementations/go => ../implementations/go
