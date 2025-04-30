---
Title: Block Structure and Semantics
Version: 0.1.0
Last Updated: 2025-05-01
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
| `version`       | String   | Protocol version identifier (e.g. `"1.0.0"`) |
| `id`            | String   | Deterministically derived block ID (hash of entire block content excluding signatures) |
| `type_code`     | Bytes    | Hashed type identifier (world-scoped) |
| `audience_code` | Bytes    | Hashed audience identifier (world-scoped) |
| `timestamp`     | Integer  | Unix timestamp in seconds |
| `data`          | Bytes    | Encrypted or plaintext payload (e.g. JSON) |
| `world_sig`     | Bytes    | Post-quantum signature from World key |
| `author_sig`    | Bytes    | Signature from author's root signing key |

---

## 🔐 Signing and Verification

Each block carries **two signatures**:

1. **World Signature (`world_sig`)**  
   - Validates that the block is scoped to a particular World  
   - Signed over: `version`, `id`, `type_code`, `audience_code`, `timestamp`, `data`

2. **Author Signature (`author_sig`)**  
   - Validates that the author endorses this block  
   - Signed over: the full content including `world_sig`

**Note:** Some block types (e.g. identity burn, RPC) may use additional co-signatures or proofs in future extensions.

---

## 🧮 ID and Code Derivation

- **Block ID** is computed as the hash (e.g. Keccak-256) of all block fields *except* the `id` and signatures.  
- **Type Code** is the hash of the block’s canonical type string, salted by the World’s type salt.  
- **Audience Code** is the hash of the intended audience string, salted by the World’s audience salt.

Deterministic derivation ensures interoperability without central coordination.

---

## 🔒 Encryption and Privacy

The `data` field is **audience-scoped encrypted** using a symmetric or derived key:

- If the recipient lacks the audience key, they cannot decrypt the payload.
- Metadata such as `timestamp`, `type_code`, and `audience_code` are visible, but not meaningful without type mappings or keys.

Payloads are typically JSON-encoded before encryption, unless the block type defines a binary format.

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
This ensures consistent wire compatibility and interoperability across clients. The Protobuf schema is defined in [block.proto](./proto/blockparty/v1/block.proto)

---

## 🔗 Related Specifications

- [Block Types and Type Identifiers](./block-types.md)
- [Derivations](./derivations.md)
- [MIME Header Conventions](./mime-header-conventions.md)
- [Whitepaper](../docs/whitepaper.md)

---

## 📄 Status

This document defines version `0.1.0` of the BlockParty block specification.  
It is subject to revision as implementations are developed and interoperability tested.  
Contributions welcome at [github.com/bpprotocol/blockparty-protocol](https://github.com/bpprotocol/blockparty-protocol).

