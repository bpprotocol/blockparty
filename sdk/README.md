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
| [`encryption`](./encryption) | ✅ implemented | #5 |
| [`audiences`](./audiences) | ✅ implemented | #6 |
| [`connections`](./connections) | ✅ implemented | #7 |
| [`blocktypes`](./blocktypes) | ✅ implemented | #8 |

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

### `encryption` — audience-scoped AEAD (#5)

Faithful to [`protocol/specs/encryption.md`](../protocol/specs/encryption.md):

- **`ContentKey`** — per-block key via HKDF-SHA256 from the audience secret, mixing in the block nonce.
- **`EncryptData` / `DecryptData`** — XChaCha20-Poly1305; `data = 24-byte nonce ‖ ciphertext+tag`; block metadata (`version‖type_code‖audience_code‖timestamp`) bound as **AAD**, so tampering it fails decryption.
- Agnostic to where the audience secret comes from (public audiences → #6, connections → #7). Inherently public block types skip this layer and store plaintext.

### `audiences` — public audiences & inbox rendezvous (#6)

Faithful to [`protocol/specs/audiences.md`](../protocol/specs/audiences.md):

- **Public audiences** `public-1 … public-16` (world-scoped): `PublicAudience` / `PublicAudienceSecret` derive code + 32-byte secret; only World-seed holders can derive the secret. Feeds the `encryption` AEAD layer.
- **Inbox audiences** (`…/inbox/<address>`): per-identity rendezvous; `InboxAudience` derives a code (no secret).
- **`Registry`** — the local code → audience map for resolving a received block's `audience_code` to its secret.

### `connections` — connect.* handshake, rotation & replay (#7)

Faithful to [`protocol/specs/connections.md`](../protocol/specs/connections.md):

- **Handshake:** `StartRequest` → `AcceptRequest` → `Complete` runs the two-message ML-KEM768 exchange (initiator ephemeral key + responder static key); both parties derive the **same** private audience secret/code.
- **Rotation:** `Rotate` / `ApplyRotate` advance to a new epoch with fresh KEM entropy (forward ratchet); `BuildClose` tears down.
- **`ReplayGuard`** — rejects replayed `(target, nonce)` pairs and stale timestamps (default ±300 s).
- Resolved a spec ambiguity: the rotation HKDF salt is the rotation ciphertext `ct3` (clarified in `connections.md`).

### `blocktypes` — core types, handlers, chunks, burn & ping (#8)

Faithful to [`protocol/specs/block-types.md`](../protocol/specs/block-types.md) and [`rpc.md`](../protocol/specs/rpc.md):

- **Type URNs + `Resolver`** — canonical type strings, a payload factory, and reverse mapping from a block's world-scoped `type_code` to its URN for dispatch.
- **`rpc.render` guard** — render is local-only, so decoding one received over a transport is refused (`ErrRenderNotTransportable`).
- **Chunk reassembly** — `ReassembleChunks` (and manifest wrappers) concatenate chunks in order and verify SHA-512.
- **Burn** — `VerifyBurn` (revealed private keys must re-derive the burned address) and `TrustState` to flag a burned identity's blocks as contested.
- **`core.ping`** — `BuildPing` / `HandlePing` / `ParsePingResponse`.

## Develop

```sh
cd sdk
go test ./...      # unit tests, KATs, determinism + golden vectors
go vet ./...
gofmt -l .         # should print nothing
```

Golden public-key vectors for a fixed seed are locked in `crypto/golden_test.go` so any change to the derivation is caught and other implementations can reproduce them.
