---
Title: Block Structure and Semantics
Version: 0.1.0
Last Updated: 2026-06-28
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/block
---

# Block Structure and Semantics

This document defines the canonical structure of a **BlockParty block**, including how blocks are encoded, signed, hashed, and interpreted by clients. It serves as the core schema reference for all BlockParty-compatible implementations.

---

## 📦 What is a Block?

A **block** is the atomic unit of communication in BlockParty. Every message, post, identity announcement, reaction, or connection request is represented as a block.

Blocks are:
- **Signed** (by both the author and World)
- **Optionally encrypted** (audience-scoped)
- **Opaque by default** (unless the viewer has decryption context)
- **Transport-agnostic** (can be stored, transmitted, or mirrored freely)

---

## 🧱 Block Schema Overview

Each block consists of the following fields:

| Field           | Type     | Description |
|-----------------|----------|-------------|
| `version`       | uint32   | Numeric protocol version (e.g. `1`). Matches the `version` field in [block.proto](../proto/v1/block.proto). |
| `id`            | Bytes    | Deterministically derived block ID — see [Derivations](./derivations.md) `GetBlockID` |
| `type_code`     | Bytes    | Hashed type identifier (world-scoped) |
| `audience_code` | Bytes    | Hashed audience identifier (world-scoped) |
| `timestamp`     | Integer  | Unix timestamp in seconds |
| `data`          | Bytes    | Encrypted or plaintext payload — see [Block Encryption](./encryption.md) |
| `sigs.world`    | Bytes    | Dilithium signature from the World key |
| `sigs.author`   | Bytes    | Dilithium signature from the author's root signing key |
| `sigs.extra[]`  | ExtraSignature | Optional co-signatures (see §"Co-Signatures") |

The wire field names follow [block.proto](../proto/v1/block.proto): the three signatures live in a `Signatures` bundle (`world`, `author`, `extra`).

---

## 🔐 Signing and Verification

Each block carries **two required Dilithium signatures**:

1. **World Signature (`sigs.world`)**  
   - Validates that the block is scoped to a particular World  
   - Produced by `World.SigningKey` (a Dilithium keypair — see [Derivations](./derivations.md) §"World Key + Salts")  
   - Signed over: `version`, `id`, `type_code`, `audience_code`, `timestamp`, `data`  
   - Any holder of the World seed can verify world membership without decrypting `data`.

2. **Author Signature (`sigs.author`)**  
   - Validates that the author endorses this block  
   - Produced by the author's `dilithiumKey` (see [Identity](./identity.md))  
   - Signed over: the full content including `sigs.world`

Clients **must verify** both signatures before trusting a block.

### Co-Signatures

The optional `sigs.extra[]` array carries **co-signatures** — additional Dilithium signatures by other identities over the canonical block bytes (including `sigs.author`). Each `ExtraSignature` has:

- `type` — a namespaced string giving the co-signature's meaning (e.g. `bpprotocol.org/v1/cosign.endorse`, `bpprotocol.org/v1/cosign.witness`).
- `sig` — the Dilithium signature bytes.

Verification rules:

- Each co-signature is verified **independently** against its signer's public key; an unverifiable co-signature is simply ignored, never invalidating the block.
- The core protocol assigns **no consensus semantics** to co-signatures — they are per-signer endorsements, not Byzantine agreement. Threshold, quorum, or multi-party-consensus schemes are **extension-layer** constructs built on top of this primitive (see [Extensions](./extensions/index.md), e.g. chainlets), not part of the core.

---

## 🧮 ID and Code Derivation

All three are defined canonically in [Derivations](./derivations.md); this spec does not redefine them:

- **Block ID** — `GetBlockID(version, timestamp, audience_code, address, type_code, data)`. It binds the payload (`Keccak256(data)`), so it is content-binding: any change to `data` changes the `id`.
- **Type Code** — `GetTypeCode(world, canonicalType)`: Keccak-256 over the type URN salted by the World's `TypeSalt`.
- **Audience Code** — `GetAudienceCode(world, audienceID)`: Keccak-256 over the audience identifier salted by the World's `AudienceSalt`.

Deterministic derivation ensures interoperability without central coordination.

---

## 🔒 Encryption and Privacy

The `data` field is **audience-scoped encrypted**. The full scheme is defined in [Block Encryption](./encryption.md): an ML-KEM768 key agreement establishes an audience secret, HKDF-SHA256 derives a per-block content key, and XChaCha20-Poly1305 (AEAD) encrypts the payload with `version‖type_code‖audience_code‖timestamp` bound as additional authenticated data.

- If the recipient cannot derive the audience secret, they cannot decrypt the payload, and the block appears as an opaque stub.
- Metadata such as `timestamp`, `type_code`, and `audience_code` are visible but not meaningful without type/audience mappings; because they are bound as AEAD AAD, tampering with them causes decryption to fail.
- A block type MAY define a [plaintext block](./encryption.md) where the payload is inherently public (e.g. `identity`, `identity.burn`).

Payloads are serialized (Protobuf, or JSON where a block type defines it) before encryption.

---

## 🛠️ Client Behavior Notes

- If a block’s `type_code` is unrecognized, clients may ignore it or offer a fallback display.
- If a block cannot be decrypted (due to unknown audience), it should appear as an opaque stub.
- Clients **must verify** both signatures before trusting the contents.
- Clients may cache decoded type/audience mappings for rendering or routing.

---

## 📥 Transport and Serialization

Blocks are serialized as binary blobs using a consistent encoding (e.g. Protobuf or CBOR, to be standardized).  
The protocol does not define a transport layer — blocks may be transmitted via:

- P2P protocols (e.g. libp2p)
- Filesystems
- QR codes or NFC
- USB stick (sneakernet)
- Delay-tolerant relays (e.g. Mars-Earth)

**⚠️ Canonical Encoding Requirement:**
All BlockParty blocks **must be serialized as Protocol Buffers** for transmission or storage.
This ensures consistent wire compatibility and interoperability across clients. The Protobuf schema is defined in [block.proto](../proto/v1/block.proto).

---

## 🔗 Related Specifications

- [Block Types and Type Identifiers](./block-types.md)
- [Derivations](./derivations.md)
- [Block Encryption](./encryption.md)
- [Audiences](./audiences.md)
- [Connections](./connections.md)
- [MIME Header Conventions](./mime-header-conventions.md)
- [Whitepaper](../whitepaper.md)

---

## 📄 Status

This document defines version `0.1.0` of the BlockParty block specification.  
It is subject to revision as implementations are developed and interoperability tested.  
Contributions welcome at [github.com/bpprotocol/blockparty-protocol](https://github.com/bpprotocol/blockparty-protocol).

