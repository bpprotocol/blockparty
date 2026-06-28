# BlockParty SDK (Go)

Go reference implementation of the [BlockParty Protocol](../protocol). Tracking epic: **#1**.

Module path: `github.com/bpprotocol/blockparty/sdk`

> Module layout (rooting at `sdk/` vs a top-level module) is provisional and revisited in #9.

## Packages

| Package | Status | Issue |
|---------|--------|-------|
| [`crypto`](./crypto) | ✅ implemented | #2 |
| [`derive`](./derive) | ✅ implemented | #3 |
| [`identity`](./identity) | ✅ implemented | #3 |
| [`blockpb`](./blockpb) | ✅ generated | #4 |
| [`block`](./block) | ✅ implemented | #4 |

### `crypto` — primitives & deterministic key generation (#2)

Faithful to [`protocol/specs/derivations.md`](../protocol/specs/derivations.md):

- **Hashing / KDF:** `Keccak256`, `Sha256`, `HMACSHA256`, `HKDFSHA256`.
- **Deterministic RNG:** `DeterministicRNG` — a SHAKE256 XOF that makes key generation reproducible across clients.
- **Post-quantum keys:** `MakeKyberPair` (ML-KEM768) and `MakeDilithiumPair` (Dilithium3), derived deterministically from a seed; plus `Sign`/`VerifyDilithium` and `Encapsulate`/`Decapsulate`.

Backed by [Cloudflare CIRCL](https://github.com/cloudflare/circl). The KEM uses FIPS-203 ML-KEM768; signatures use round-3 Dilithium3 (matching the spec's "Dilithium" wording). Migrating signatures to ML-DSA (FIPS 204) is future work.

### `derive` — worlds, codes, address & block ID (#3)

Faithful to [`protocol/specs/derivations.md`](../protocol/specs/derivations.md):

- **Worlds:** `OpenWorld` / `GenerateWorld` → Dilithium signing key + wallet/type/audience salts (`GlobalSalt = "bpprotocol.org/v1/global"`).
- **Codes:** `GetTypeCode` / `GetAudienceCode` → 16-byte, world-scoped `Code` (Keccak-256).
- **Address:** `BytesToAddress` → 40-char hex binding both PQ public keys.
- **Block ID:** `GetBlockID` → content-binding identifier (folds in `Keccak256(data)`).

String identifiers are normalized (trim surrounding whitespace + trailing slashes) before hashing.

### `identity` — identity derivation (#3)

Faithful to [`protocol/specs/identity.md`](../protocol/specs/identity.md): `OpenIdentity(world, passphrase)` derives an `Identity` (Dilithium + Kyber keys + address) via a world-scoped `worldPassword` (HKDF over `WalletSalt`) and `mlkem`/`dilithium` domain separation. The same passphrase yields a distinct identity per World.

### `blockpb` — generated protobuf types (#4)

Go types generated from [`protocol/proto/v1`](../protocol/proto/v1) (`Block`, `Signatures`, `ExtraSignature`, and all block-type payloads). Regenerate with `go generate ./blockpb` (needs `protoc` + `protoc-gen-go`).

### `block` — envelope, signing & co-signatures (#4)

Faithful to [`protocol/specs/block.md`](../protocol/specs/block.md):

- **Lifecycle:** `New` (derives the content-binding ID) → `Sign` → `AddCoSignature`* → `Encode`; `Decode` → `Verify`.
- **Two-layer signing:** `world_sig` (World key) then `author_sig` (author key, over the fields + `world_sig`).
- **Co-signatures** (`sigs.extra`): each an independent Dilithium signature over the fields + `world_sig` + `author_sig`; an invalid co-signature never invalidates the block.
- Signatures are over a domain-separated, length-prefixed **preimage** of the signed fields (not the wire bytes), so they're independent of encoder ordering. Wire format is deterministic Protobuf.

## Develop

```sh
cd sdk
go test ./...      # unit tests, KATs, determinism + golden vectors
go vet ./...
gofmt -l .         # should print nothing
```

Golden public-key vectors for a fixed seed are locked in `crypto/golden_test.go` so any change to the derivation is caught and other implementations can reproduce them.
