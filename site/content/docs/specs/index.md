---
title: BlockParty Protocol Specs
version: 0.1.0
updated: 2026-06-28
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/
---

# 🧪 BlockParty Protocol Specs

This folder contains the formal specifications for the BlockParty Protocol. These documents define how nodes, clients, and worlds interact within the system.

## 📂 Spec Documents

- [`block.md`](./block.md) — Canonical block structure, signatures, and co-signatures
- [`block-types.md`](./block-types.md) — Canonical block types and their payload schemas
- [`derivations.md`](./derivations.md) — Deterministic identifier and key derivations (identity, audience, world, block ID)
- [`encryption.md`](./encryption.md) — Audience-scoped AEAD encryption of block payloads
- [`identity.md`](./identity.md) — Identity model, key derivation, and address format
- [`identity-burn-rfc.md`](./identity-burn-rfc.md) — RFC: Burnable Identities and the Post-Truth State
- [`connections.md`](./connections.md) — The `connect.*` post-quantum handshake and private audiences
- [`audiences.md`](./audiences.md) — Audience model and public-audience derivation
- [`rpc.md`](./rpc.md) — Core RPC definitions (e.g., `core.ping`)
- [`mime-header-conventions.md`](./mime-header-conventions.md) — Conventions for content headers in BlockParty messages
- [`glossary.md`](./glossary.md) — Reference glossary of protocol terms
- [`extensions/`](./extensions/index.md) — Directory for optional or experimental extensions

## 🔐 Key Themes

- Determinism and forward-compatibility
- Decentralized identity and addressing
- Delay-tolerant RPC and content dissemination

## 📌 Notes

- All specs are CC0-licensed.
- Versioning: Specs are considered stable once merged to `main`.

