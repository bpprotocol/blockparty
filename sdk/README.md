# BlockParty SDK (Go)

Go reference implementation of the [BlockParty Protocol](../protocol). Tracking epic: **#1**.

Module path: `github.com/bpprotocol/blockparty/sdk`

> Module layout (rooting at `sdk/` vs a top-level module) is provisional and revisited in #9.

## Packages

| Package | Status | Issue |
|---------|--------|-------|
| [`crypto`](./crypto) | ✅ implemented | #2 |

### `crypto` — primitives & deterministic key generation (#2)

Faithful to [`protocol/specs/derivations.md`](../protocol/specs/derivations.md):

- **Hashing / KDF:** `Keccak256`, `Sha256`, `HMACSHA256`, `HKDFSHA256`.
- **Deterministic RNG:** `DeterministicRNG` — a SHAKE256 XOF that makes key generation reproducible across clients.
- **Post-quantum keys:** `MakeKyberPair` (ML-KEM768) and `MakeDilithiumPair` (Dilithium3), derived deterministically from a seed; plus `Sign`/`VerifyDilithium` and `Encapsulate`/`Decapsulate`.

Backed by [Cloudflare CIRCL](https://github.com/cloudflare/circl). The KEM uses FIPS-203 ML-KEM768; signatures use round-3 Dilithium3 (matching the spec's "Dilithium" wording). Migrating signatures to ML-DSA (FIPS 204) is future work.

## Develop

```sh
cd sdk
go test ./...      # unit tests, KATs, determinism + golden vectors
go vet ./...
gofmt -l .         # should print nothing
```

Golden public-key vectors for a fixed seed are locked in `crypto/golden_test.go` so any change to the derivation is caught and other implementations can reproduce them.
