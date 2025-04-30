---
Title: Deterministic Derivations
Version: 0.1.0
Last Updated: 2025-05-02
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/derivations
---

# Deterministic Derivations

This document defines how identifiers and cryptographic primitives in BlockParty are **deterministically derived** from user input and world context. These derivations ensure interoperability across clients and avoid reliance on central registries. BlockParty uses deterministic functions to compute unique, reproducible identifiers — such as block IDs, type codes, audience hashes, and identity addresses — without relying on central coordination. This document defines the canonical algorithms and inputs for those derivations.

All examples assume:
- Cryptographic primitives (e.g. Keccak-256, SHA-256, MD5)
- Encoding as UTF-8 and byte-safe inputs
- All hash outputs are hex-encoded unless otherwise specified


⚠️ Normalization Rule:
All string inputs to hash or key derivation functions must be normalized to UTF-8, with no trailing slashes or whitespace. Identifiers are treated as case-sensitive unless otherwise stated.

---

## 📛 Type Code

Derives a type hash scoped to a World.

Function: `GetTypeCode(world, canonicalType)`

**Pseudocode:**
```ts
function GetTypeCode(world, canonicalType):
  typeSeed = HMAC_SHA256(key=world.TypeSalt, message=canonicalType)
  hash = MD5(typeSeed)
  return hex(hash)
```

- canonicalType = e.g. `bpprotocol.org/v1/content.post`
- World-specific hash ensures uniqueness across Worlds

---

## 👥 Audience Code

Derives a deterministic audience ID scoped to a World.

Function: `GetAudienceCode(world, audienceID)`

**Pseudocode:**
```ts
function GetAudienceCode(world, audienceID):
  kp = MakeECCPair(seed=HMAC_SHA256(key=world.AudienceSalt, message=audienceID))
  return hex(MD5(kp.publicKey))
```

- Used for encrypting content to a shared audience
- Clients must maintain a map of known audiences

---

## 🌐 World Key + Salts

Worlds are cryptographic domains with isolated type and audience scopes.

Function: `OpenWorld(seedPhase)`

**Pseudocode:**
```ts
function OpenWorld(seedPhase):
  rootKeyPair = MakeECCPair(seed=HMAC_SHA256(key=GLOBAL_SALT, message=seedPhase))
  return GenerateWorld(rootKeyPair.publicKey, rootKeyPair.privateKey)

function GenerateWorld(identity, genesisKey):
  return World(
    Identity: identity,
    Genesis: genesisKey,
    WalletSalt: HMAC_SHA256(genesisKey, "wallets"),
    TypeSalt: HMAC_SHA256(genesisKey, "types"),
    AudienceSalt: HMAC_SHA256(genesisKey, "channels")
  )
```


---

## 🆔 Identity Address

Function: BytesToAddress(pubKey)

**Pseudocode:**
```ts
function BytesToAddress(pubKey):
  keccak = Keccak256(pubKey)
  return hex(keccak[-20:]) // last 20 bytes of hash
```

- Same convention as Ethereum
- Results in 40-character hex address

---

## 🧾 Block ID

Function: `GetBlockID(version, timestamp, audienceCode, identityAddress, typeCode)`

**Pseudocode:**
```ts
function GetBlockID(version, ts, audienceCode, address, typeCode):
  message = "block.v" + version + ":" + ts + "/" + address + "/" + typeCode
  seed = HMAC_SHA256(key=audienceCode, message=message)
  return hex(SHA256(seed))
```

- The `version` input corresponds to the block’s numeric `version` field (e.g. `1`), enabling version-aware ID derivation via the `block.v1:` namespace prefix.
- Including `audienceCode`, `identityAddress`, `timestamp`, and `typeCode` ensures uniqueness per actor, audience, type, and moment in time.
- Clients may compute this ID before signing the block and compare it to the final encoded block to verify integrity.
```

---

## 🔐 Key Derivation (Account, Audience, World)

Deterministic key derivation is used to generate the same public/private key pair for a given seed. The method used depends on the cryptographic scheme. Each function below takes a `seed` (e.g., a Keccak256 digest or structured string) and returns a consistent keypair.

---

### 🧩 `MakeECCPair(seed)`
- Uses a heuristic approach to derive an ECC private key from the seed.
- Fast, lightweight, and compatible with traditional crypto ecosystems.
- Useful for identities, world salts, or namespaces.

**Pseudocode::**
```ts
function MakeECCPair(seed):
    privKey = ECC.PrivateKeyFromBytes(Keccak256(seed))
    pubKey = privKey.toPublicKey()
    return { pubKey, privKey }
```

> ⚠️ Note: Ensure the hash output is a valid scalar within the curve's field. Use curve-specific clamping for curves like Ed25519.

---

### 🛡️ `MakeKyberPair(seed)`
- Produces a post-quantum keypair (KEM) using Kyber.
- Requires deterministic RNG based on seed.

**Pseudocode::**
```ts
function MakeKyberPair(seed):
    rng = DeterministicRNG(seed)  // Seeded with e.g. Keccak256(seed)
    (pubKey, privKey) = Kyber.KeyGen(rng)
    return { pubKey, privKey }
```

> 📦 Kyber is a lattice-based KEM. Typically used for encrypting shared secrets, not signing.

---

### ✍️ `MakeDilithiumPair(seed)`
- Produces a post-quantum signature keypair using Dilithium.
- Also requires deterministic RNG from seed.

**Pseudocode::**
```ts
function MakeDilithiumPair(seed):
    rng = DeterministicRNG(seed)
    (pubKey, privKey) = Dilithium.KeyGen(rng)
    return { pubKey, privKey }
```

> ✒️ Dilithium is suited for deterministic signature schemes — useful if you ever want post-quantum identity assertions.

---

This document defines deterministic derivation logic to ensure protocol consistency, verifiability, and extensibility. Future updates may include cross-language implementation examples.