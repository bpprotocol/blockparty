---
Title: BlockParty Protocol Specs
Version: 0.1.0
Last Updated: 2025-04-24
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/
---

# 🧪 BlockParty Protocol Specs

This folder contains the formal specifications for the BlockParty Protocol. These documents define how nodes, clients, and worlds interact within the system.

## 📂 Spec Documents

- [`rpc.md`](./rpc.md) — Core RPC definitions (e.g., `core.ping`)
- [`derivations.md`](./derivations.md) — Deterministic address and entropy derivations (e.g., identity, audience, world)
- [`identity.md`](./identity.md) — Identity model and address formats
- [`identity-burn-rfc.md`](./identity-burn-rfc.md) — RFC: Burnable Identities and the Post-Truth State in Cryptographic Protocols
- [`block-types.md`](./block-types.md) — Block kinds and rules (e.g., profile updates, audience invites)
- [`extensions/`](./extensions/index.md) — Directory for optional or experimental extensions
- [`mime-header-conventions.md`](./mime-header-conventions.md) — Conventions for content headers in BlockParty messages

## 🔐 Key Themes

- Determinism and forward-compatibility
- Decentralized identity and addressing
- Delay-tolerant RPC and content dissemination

## 📌 Notes

- All specs are CC0-licensed.
- Versioning: Specs are considered stable once merged to `main`.

