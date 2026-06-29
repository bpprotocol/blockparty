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
| blocks                | ⬜             | #14   |
| encryption            | ⬜             | #15   |
| audiences             | ⬜             | #16   |
| connections           | ⬜             | #17   |
| block types           | ⬜             | #18   |

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
