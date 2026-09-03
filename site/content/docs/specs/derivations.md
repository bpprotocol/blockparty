---
title: Deterministic Derivations
version: 0.1.0
updated: 2026-06-28
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/derivations
---

# Deterministic Derivations

This document defines how identifiers and cryptographic primitives in BlockParty are **deterministically derived** from user input and world context. These derivations ensure interoperability across clients and avoid reliance on central registries. BlockParty uses deterministic functions to compute unique, reproducible identifiers — such as block IDs, type codes, audience hashes, and identity addresses — without relying on central coordination. This document defines the canonical algorithms and inputs for those derivations.

All examples assume:
- Cryptographic primitives: **Keccak-256**, **SHA-256**, **HMAC-SHA256**, and **HKDF-SHA256**
- Post-quantum keypairs: **ML-KEM768** (KEM) and **ML-DSA-65** (signatures)
- Encoding as UTF-8 and byte-safe inputs
- All hash outputs are hex-encoded unless otherwise specified

> 🔐 **Post-quantum by default.** Every key BlockParty derives is post-quantum (ML-KEM768 or ML-DSA-65). The protocol does **not** use classical elliptic-curve cryptography anywhere in its core. Earlier drafts derived World and audience material from classical ECC and hashed identifiers with MD5; both have been removed in favor of Keccak-256 and the post-quantum stack. See [Block Encryption](./encryption.md) for the symmetric/AEAD layer.

⚠️ **Normalization Rule:**
All string inputs to hash or key derivation functions must be normalized to UTF-8, with no trailing slashes or whitespace. Identifiers are treated as case-sensitive unless otherwise stated.

---

## 🧂 Protocol Constants

| Constant | Value | Purpose |
|----------|-------|---------|
| `GLOBAL_SALT` | `"bpprotocol.org/v1/global"` | Fixed domain-separation salt for opening Worlds from a seed phrase. Hard-coded into every conforming client; it is **not** a secret — it only namespaces BlockParty derivations away from collisions with other systems that reuse the same seed phrases. |
| `CODE_BYTES` | `16` | Truncated width (in bytes) of a `type_code` or `audience_code`. |

---

## 📛 Type Code

Derives a type hash scoped to a World.

Function: `GetTypeCode(world, canonicalType)`

**Pseudocode:**
```ts
function GetTypeCode(world, canonicalType):
  seed = HMAC_SHA256(key=world.TypeSalt, message=canonicalType)
  return hex(Keccak256(seed)[0:CODE_BYTES])   // 16-byte, world-scoped code
```

- `canonicalType` = e.g. `bpprotocol.org/v1/types/content.post`
- The World's `TypeSalt` ensures the same identifier hashes differently across Worlds.
- The code is a non-secret identifier; Keccak-256 (collision-resistant) replaces the MD5 used in earlier drafts.

---

## 👥 Audience Code

Derives a deterministic audience identifier scoped to a World.

Function: `GetAudienceCode(world, audienceID)`

**Pseudocode:**
```ts
function GetAudienceCode(world, audienceID):
  seed = HMAC_SHA256(key=world.AudienceSalt, message=audienceID)
  return hex(Keccak256(seed)[0:CODE_BYTES])   // 16-byte, world-scoped code
```

- `audienceID` is any agreed string. Public audiences use the well-known identifiers `bpprotocol.org/v1/audience/public-N`; private audiences use a connection-derived identifier (see [Connections](./connections.md)).
- The audience **code** is only an addressing label. The **encryption key** for an audience is derived separately — see [Block Encryption](./encryption.md).
- Clients maintain a local map of `audience_code → audienceID/key` for audiences they participate in.

---

## 🌐 World Key + Salts

Worlds are cryptographic domains with isolated type and audience scopes. A World is defined by a **ML-DSA-65** signing keypair (its `world_sig` authority) and three derived salts.

Function: `OpenWorld(seedPhrase)`

**Pseudocode:**
```ts
function OpenWorld(seedPhrase):
  worldSeed = HMAC_SHA256(key=GLOBAL_SALT, message=seedPhrase)
  signingKey = MakeMLDSAPair(worldSeed)   // World key — signs world_sig
  return GenerateWorld(signingKey, worldSeed)

function GenerateWorld(signingKey, worldSeed):
  return World(
    SigningKey:   signingKey,                          // ML-DSA-65 keypair
    WalletSalt:   HMAC_SHA256(worldSeed, "wallets"),   // scopes identity derivation
    TypeSalt:     HMAC_SHA256(worldSeed, "types"),     // scopes type codes
    AudienceSalt: HMAC_SHA256(worldSeed, "channels")   // scopes audience codes
  )
```

- Anyone who knows `seedPhrase` derives the identical World — including the same `SigningKey`. World membership is therefore gated purely by knowledge of the seed, not by protocol-enforced permissions (see the [Whitepaper](../whitepaper.md) §3).
- `World.SigningKey.private` produces the **World Signature** (`world_sig`) on every block scoped to the World; `World.SigningKey.public` lets any holder of the seed verify it. The World key is ML-DSA-65 so that `world_sig` is a genuine post-quantum signature, consistent with [Block Structure](./block.md) §"Signing and Verification".
- `WalletSalt` is consumed by identity derivation (see [Identity](./identity.md)); `TypeSalt` and `AudienceSalt` by the code derivations above.

---

## 🆔 Identity Address

Function: `BytesToAddress(mldsaPub, kyberPub)`

**Pseudocode:**
```ts
function BytesToAddress(mldsaPub, kyberPub):
  digest = Keccak256("v1" || mldsaPub || kyberPub)
  return hex(digest[-20:])   // last 20 bytes → 40-char hex address
```

- Binds **both** post-quantum public keys (signature + KEM), so an address commits to the whole identity, not just one key.
- The `"v1"` domain prefix allows the address scheme to evolve without colliding with future versions.
- Truncating to the last 20 bytes yields a 40-character hex address (same width as Ethereum). This is the single canonical address derivation; [Identity](./identity.md) and the [Whitepaper](../whitepaper.md) reference it rather than redefining it.

---

## 🧾 Block ID

Function: `GetBlockID(version, timestamp, audienceCode, identityAddress, typeCode, data)`

**Pseudocode:**
```ts
function GetBlockID(version, ts, audienceCode, address, typeCode, data):
  dataHash = hex(Keccak256(data))
  message  = "block.v" + version + ":" + ts + "/" + address + "/" + typeCode + "/" + dataHash
  seed     = HMAC_SHA256(key=audienceCode, message=message)
  return hex(SHA256(seed))
```

- The `version` input is the block's numeric `version` field (e.g. `1`), namespacing the ID via the `block.v1:` prefix.
- Including `audienceCode`, `identityAddress`, `timestamp`, `typeCode`, **and `Keccak256(data)`** makes the ID unique per actor, audience, type, moment, and payload — and **content-binding**: two blocks that differ only in their `data` payload receive different IDs. This is the canonical block-ID algorithm; [Block Structure](./block.md) references it.
- Clients compute this ID before signing and compare it to the encoded block to verify integrity.

---

## 🔐 Key Derivation (Identity, World, Audience)

Deterministic key derivation generates the same keypair for a given seed. Each function takes a `seed` (a Keccak-256 digest or a structured, normalized string) and returns a consistent keypair. BlockParty uses **only** post-quantum schemes.

---

### 🛡️ `MakeKyberPair(seed)`
- Produces a post-quantum **ML-KEM768** (Kyber) KEM keypair — used for encryption / key agreement.
- Requires a deterministic RNG so the same seed always yields the same keypair.

**Pseudocode:**
```ts
function MakeKyberPair(seed):
    rng = DeterministicRNG(Keccak256(seed))   // SHAKE256 XOF seeded by Keccak256(seed)
    (pubKey, privKey) = MLKEM768.KeyGen(rng)
    return { pubKey, privKey }
```

> 📦 ML-KEM768 is a lattice-based KEM. It encapsulates a shared secret used to key the AEAD layer ([Block Encryption](./encryption.md)); it is not a signature scheme.

---

### ✍️ `MakeMLDSAPair(seed)`
- Produces a post-quantum **ML-DSA-65** signature keypair.
- Also requires a deterministic RNG.

**Pseudocode:**
```ts
function MakeMLDSAPair(seed):
    rng = DeterministicRNG(Keccak256(seed))   // SHAKE256 XOF seeded by Keccak256(seed)
    (pubKey, privKey) = MLDSA.KeyGen(rng)
    return { pubKey, privKey }
```

> ✒️ ML-DSA-65 produces the `author_sig`, the `world_sig` (via `World.SigningKey`), and any co-signatures (see [Block Structure](./block.md)).

---

### 🎲 `DeterministicRNG(seed)`

`MakeKyberPair` and `MakeMLDSAPair` require a deterministic byte stream so that key generation is reproducible from a seed. BlockParty specifies a **SHAKE256** extendable-output function (XOF) absorbing the 32-byte `seed`; the KEM/signature `KeyGen` routine reads as many bytes as it needs from the XOF. Any conforming implementation MUST use SHAKE256 here so that keypairs derived from the same seed are byte-identical across clients.

---

## Summary of Primitives

| Use | Primitive |
|-----|-----------|
| Type / audience codes, address, payload hash | Keccak-256 |
| Block ID, salts, domain separation | SHA-256 / HMAC-SHA256 |
| Deterministic key generation RNG | SHAKE256 XOF |
| Signatures (author, world, co-sign) | ML-DSA-65 |
| Key agreement / encryption | ML-KEM768 |
| Symmetric AEAD over `data` | XChaCha20-Poly1305 (see [encryption.md](./encryption.md)) |

This document defines deterministic derivation logic to ensure protocol consistency, verifiability, and extensibility. Future updates may include cross-language implementation vectors.
