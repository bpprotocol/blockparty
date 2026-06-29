# BlockParty JS

Node.js / TypeScript reference implementation of the [BlockParty Protocol](../protocol). Tracking epic: **#10**.

This is a [pnpm](https://pnpm.io) workspace with a deliberate package boundary so the protocol logic is a **reusable, separately-publishable library**, not buried inside the CLI:

```
js/
├── packages/
│   ├── sdk/   → @blockparty/sdk   (reusable library — all protocol logic, published)
│   └── cli/   → @blockparty/cli   (thin reference node/CLI — depends on @blockparty/sdk)
```

- **`@blockparty/sdk`** holds every protocol primitive (crypto, derivations, identity, blocks, encryption, audiences, connections, block types). It is framework-agnostic and fully typed.
- **`@blockparty/cli`** is a thin consumer — argument parsing, transport, and wiring. It depends on the SDK via `workspace:*`.

External implementers reuse the library directly:

```sh
pnpm add @blockparty/sdk
```

```ts
import { PROTOCOL_VERSION } from "@blockparty/sdk";
```

## Develop

```sh
cd js
pnpm install
pnpm -r build     # build every package (sdk before cli)
pnpm -r test      # run unit tests (Vitest)
pnpm -r lint      # ESLint
pnpm -r typecheck # tsc --noEmit
```

## Modules

Each `@blockparty/sdk` module mirrors the Go reference in [`../sdk`](../sdk) and
is validated against the shared conformance vectors in
[`../sdk/vectors/vectors.json`](../sdk/vectors/vectors.json).

| Module                | Status         | Issue |
| --------------------- | -------------- | ----- |
| `crypto`              | ✅ implemented | #12   |
| `derive` / `identity` | ✅ implemented | #13   |
| `block`               | ✅ implemented | #14   |
| `encryption`          | ✅ implemented | #15   |
| `audiences`           | ✅ implemented | #16   |
| `connections`         | ✅ implemented | #17   |
| `blocktypes`          | ✅ implemented | #18   |

### `crypto` (#12)

Post-quantum primitives via [`@noble/post-quantum`](https://github.com/paulmillr/noble-post-quantum) and [`@noble/hashes`](https://github.com/paulmillr/noble-hashes):

- **Hashing / KDF:** `keccak256`, `sha256`, `sha512`, `hmacSha256`, `hkdfSha256`.
- **Deterministic RNG:** `deterministicRNG` (SHAKE256 XOF) for reproducible keygen.
- **Keys:** `makeKyberPair` (ML-KEM768), `makeMldsaPair` (ML-DSA-65); `signMldsa`/`verifyMldsa` (deterministic signing) and `encapsulate`/`decapsulate`.

Keygen is **byte-identical to the Go reference** — the crypto test asserts both PQ public-key fingerprints reproduce `vectors.json` exactly.

Cryptography (resolved in #20): **ML-DSA-65** (FIPS 204) for signatures and
**ML-KEM768** (FIPS 203) for the KEM.

### `derive` / `identity` (#13)

- **`derive`** — `openWorld`/`generateWorld` (signing key + wallet/type/audience salts), `getTypeCode`/`getAudienceCode` (16-byte codes), `bytesToAddress` (40-char hex binding both PQ keys), `getBlockID` (content-binding).
- **`identity`** — `openIdentity(world, passphrase)`: world-scoped `worldPassword` (HKDF over the wallet salt) + `ml-dsa`/`mlkem` domain separation.

All derivations reproduce `vectors.json` exactly (salts, codes, address, block ID, identity) — byte-identical to Go, including the empty-salt HKDF.

### `block` (#14)

The block envelope: `newBlock`/`signBlock`/`verifyBlock`, `addCoSignature`/`verifyCoSignature`, `encode`/`decode`. Wire types are generated from [`../protocol/proto/v1`](../protocol/proto/v1) with [protobuf-es](https://github.com/bufbuild/protobuf-es) into `src/blockpb` (regenerate via `pnpm --filter @blockparty/sdk gen:proto`; needs `protoc`). Signatures are over a domain-separated, length-prefixed preimage shared verbatim with Go.

**Full byte-level interop with Go:** the cross-impl test builds and signs a block and reproduces the Go reference's signed-block `idHex` **and** canonical-encoding fingerprint from `vectors.json` exactly — a Node block is byte-identical to a Go block, so each verifies the other's.

### `encryption` (#15)

Audience-scoped AEAD via [`@noble/ciphers`](https://github.com/paulmillr/noble-ciphers): `contentKey` (HKDF), `aad`, `encryptData`/`decryptData`/`encryptForBlock` using XChaCha20-Poly1305 (`data = 24-byte nonce ‖ ciphertext+tag`, block metadata bound as AAD).

**Cross-impl ciphertext interop both ways:** the test reproduces the Go-sealed ciphertext from `vectors.json` byte-for-byte (Node → Go) **and** decrypts that Go ciphertext back to the plaintext (Go → Node).

### `audiences` (#16)

- **Public audiences** `public-1 … public-16` (world-scoped): `publicAudience` / `publicAudienceSecret` derive code + 32-byte secret (matching `vectors.json`); only World-seed holders can derive the secret.
- **Inbox audiences** (`…/inbox/<address>`): per-identity rendezvous code with no secret.
- **`Registry`** — local `code → audience` map for resolving a received block's `audience_code`.

### `connections` (#17)

The `connect.*` handshake: `startRequest` → `acceptRequest` → `complete` (two-message ML-KEM768 exchange), `Connection.rotate`/`applyRotate` (forward ratchet), `buildClose`, and a `ReplayGuard`. The per-epoch derivations (`connectionSecret0`, `audienceCode`, `rotatedSecret`) reproduce `vectors.json` exactly; tests prove two parties derive the **same** private audience and exchange an encrypted block over it.

### `blocktypes` (#18)

Type-URN constants + a `Resolver` mapping `type_code → URN`, `decodePayload` (with the **`rpc.render` transport guard**), chunk reassembly (SHA-512), `identity.burn` (`buildBurn`/`verifyBurn` + `TrustState`), and `core.ping`. The generated payload types/schemas and protobuf-es helpers (`create`/`toBinary`/`fromBinary`) are re-exported from the SDK root.

`verifyBurn` re-derives the address from the revealed private keys via `getPublicKey` — and a Node burn's revealed keys are **byte-identical to Go's** (the test checks against the shared fixture), so burns interoperate.

The SDK is now complete: crypto → derivations → identity → blocks → encryption → audiences → connections → block types, every layer cross-impl-verified against the Go reference.
