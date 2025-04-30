# BlockParty Protocol Specification

> 📡 Core protocol, specification, and cryptographic foundations for the BlockParty decentralized social layer.

This directory contains the **canonical reference** for the BlockParty Protocol, including:

- 📘 The [Whitepaper](./whitepaper.md) — a high-level overview of design goals and protocol architecture
- 📑 The [Specs directory](./specs/) — detailed technical specifications for block structure, types, derivations, and encryption
- 📦 [Protobuf definitions](./proto/) — wire format for block serialization across languages

---

## ✦ Structure

```
protocol/
├── whitepaper.md       # Core protocol overview and philosophy
├── specs/              # Specification documents (block format, types, derivations, etc.)
├── proto/              # Protobuf definitions for block encoding
├── VERSION             # Protocol version file (semantic)
```

---

## ✦ Key Specifications

- 🔐 [Block Structure](./specs/block.md) — Core schema for all blocks
- 🧠 [Deterministic Derivations](./specs/derivations.md) — How identities, audiences, types, and block IDs are computed
- 🧱 [Block Types](./specs/block-types.md) — Canonical domain-scoped block types and their structures
- 📎 [MIME Header Conventions](./specs/mime-header-conventions.md) — For typed `content.post` and `chunked` messages

All documents in this directory are versioned and follow semver via the `VERSION` file.

---

## ✦ Encoding

BlockParty messages are encoded as **Protobuf binary payloads**, with schemas defined under [`proto/`](./proto/). These formats are language-agnostic and designed for delay-tolerant transport across any medium — from p2p gossip to paper-printed QR codes.

---

## ✦ Status

This protocol is in active development. Contributions are welcome.  
See [../CHANGELOG.md](../CHANGELOG.md) for version history and upcoming changes.

> Canonical spec lives in this repo. Discussions and improvements via issues or pull requests.