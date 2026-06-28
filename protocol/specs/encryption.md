---
Title: Block Encryption
Version: 0.1.0
Last Updated: 2026-06-28
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/encryption
---

# Block Encryption

This document defines how a block's `data` payload is **encrypted to an audience**. The [Block Structure](./block.md) spec describes the block envelope and signatures; this spec defines the confidentiality layer that fills the `data` field. All key material is post-quantum (see [Derivations](./derivations.md)).

> 🔑 **Summary:** `ML-KEM768` establishes a 32-byte shared secret → `HKDF-SHA256` derives a per-block content key → `XChaCha20-Poly1305` encrypts `data` with the block's metadata bound as additional authenticated data (AAD).

---

## 🧩 The Audience Key

Every audience has an **audience secret** — a 32-byte value that all members of the audience can derive and non-members cannot. How the secret is established depends on the audience kind:

| Audience kind | How the audience secret is established |
|---------------|----------------------------------------|
| **Public** (`bpprotocol.org/v1/audience/public-N`) | Deterministically derived from the well-known identifier and the World — anyone who knows the World seed and the public identifier can derive it. See [Audiences](./audiences.md). Public audiences are *confidential to the World*, not to a private group. |
| **Private** (1:1 or small group) | Derived from an **ML-KEM768 key agreement** performed during the connection handshake. See [Connections](./connections.md). |

For a public audience, the audience secret is:

```ts
function PublicAudienceSecret(world, audienceID):
  return HKDF_SHA256(
    ikm  = world.AudienceSalt,
    salt = GetAudienceCode(world, audienceID),
    info = "bpprotocol.org/v1/audience-secret",
    len  = 32)
```

For a private audience, the audience secret is the connection's shared secret (the output of the KEM handshake), bound to the current channel epoch so that `connect.rotate` produces a fresh secret (see [Connections](./connections.md) §"Rotation").

---

## 🔐 Per-Block Content Key

A fresh content key is derived **per block** from the audience secret, so that compromise of one block's key does not reveal others and so the same audience secret can encrypt many blocks safely:

```ts
function ContentKey(audienceSecret, audienceCode, blockNonce):
  return HKDF_SHA256(
    ikm  = audienceSecret,
    salt = audienceCode,
    info = "bpprotocol.org/v1/aead" || blockNonce,
    len  = 32)
```

- `blockNonce` is the 24-byte random value also used as the XChaCha20-Poly1305 nonce (below). Binding it into the KDF gives a unique content key per block even under the same audience secret.

---

## 📦 Encrypting `data`

```ts
function EncryptData(audienceSecret, audienceCode, plaintext, aad):
  blockNonce = random(24)                              // XChaCha20-Poly1305 nonce
  key        = ContentKey(audienceSecret, audienceCode, blockNonce)
  ciphertext = XChaCha20Poly1305.Seal(key, blockNonce, plaintext, aad)
  return blockNonce || ciphertext                       // stored in the block's `data` field
```

- **Cipher:** XChaCha20-Poly1305 (AEAD). The 24-byte extended nonce permits random nonces without a counter, which suits delay-tolerant, multi-device, offline authorship where a global counter is impractical.
- **Layout of `data`:** the 24-byte `blockNonce`, then the AEAD ciphertext (which includes the 16-byte Poly1305 tag).
- **Plaintext:** the block-type payload, serialized per [Block Types](./block-types.md) (Protobuf, or JSON where a block type defines it), prior to encryption.

### Additional Authenticated Data (AAD)

The AEAD authenticates — but does not encrypt — the visible block metadata, so a tampered `type_code`, `audience_code`, `timestamp`, or `version` causes decryption to fail:

```ts
aad = version || type_code || audience_code || timestamp
```

This binds the ciphertext to the exact envelope it was authored for, complementing the `world_sig`/`author_sig` over the same fields.

---

## 🔓 Decryption

```ts
function DecryptData(audienceSecret, block):
  blockNonce, ciphertext = split(block.data, 24)
  key  = ContentKey(audienceSecret, block.audience_code, blockNonce)
  aad  = block.version || block.type_code || block.audience_code || block.timestamp
  return XChaCha20Poly1305.Open(key, blockNonce, ciphertext, aad)   // fails if AAD or tag mismatch
```

A recipient who cannot derive `audienceSecret` (wrong World, not a member of the private audience) cannot recover `key`, and the block remains an opaque stub — consistent with the "opaque by default" property in [Block Structure](./block.md).

---

## 🪪 Plaintext Blocks

A block MAY be left unencrypted when its payload is inherently public (for example, an `identity` announcement or an `identity.burn` that must be world-readable). Such blocks:

- set `audience_code` to the relevant public audience code, and
- place the serialized payload directly in `data` with no AEAD wrapper.

Whether a given block type is plaintext or encrypted is defined by that type in [Block Types](./block-types.md). Signatures (`world_sig`, `author_sig`) are always present regardless of encryption.

---

## 🔁 Forward Secrecy & Rotation

- The per-block content key derivation limits blast radius: leaking one content key does not expose other blocks under the same audience.
- BlockParty does **not** claim full forward secrecy at the identity level: an attacker who later obtains a private audience's long-lived `kyberKey` private key can re-derive past connection secrets unless the connection was rotated. `connect.rotate` ([Connections](./connections.md)) advances the channel epoch and establishes a fresh audience secret, bounding exposure to the epoch in which a key was compromised.
- Identity-level repudiation is handled separately by [identity burn](./identity-burn-rfc.md), which targets *authenticity*, not confidentiality.

---

## 🔗 Related Specifications

- [Block Structure](./block.md)
- [Derivations](./derivations.md)
- [Audiences](./audiences.md)
- [Connections](./connections.md)
