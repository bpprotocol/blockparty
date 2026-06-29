<p align="center">
  <img src="./assets/banner.svg" alt="BlockParty Protocol Banner" width="100%">
</p>

# BlockParty Monorepo

![Version](https://img.shields.io/badge/version-0.1.0-blueviolet)
![Status](https://img.shields.io/badge/status-draft-yellow)
[![License: CC0](https://img.shields.io/badge/license-CC0-ff69b4.svg)](https://creativecommons.org/publicdomain/zero/1.0/)
[![Made for Post-Network World](https://img.shields.io/badge/built_for-Post--Network%20World-black)](#)
[![Read the Whitepaper](https://img.shields.io/badge/whitepaper-available-brightgreen)](./protocol/whitepaper.md)

> **BlockParty is a post-censorship, post-platform, post-network protocol for speech that forgives.**

This monorepo contains the entire BlockParty ecosystem:
- The core **protocol** specifications and derivation logic
- The **reference implementations** (Go and TypeScript) — see [`IMPLEMENTATIONS.md`](./IMPLEMENTATIONS.md)
- The **blockparty.org site**, documentation, essays, and design philosophy

> Reference implementations are validated against the shared [`conformance/vectors.json`](./conformance/vectors.json), so they interoperate byte-for-byte. See [`IMPLEMENTATIONS.md`](./IMPLEMENTATIONS.md) for the index and how to add a new language.

BlockParty introduces new primitives like **burnable identities**, **audience-scoped encryption**, and **post-truth state transitions** — forming a delay-tolerant, cryptographically grounded foundation for social communication in adversarial or disconnected environments.

---

## ✦ Repository Structure

> 📦 Reference implementations live under `implementations/`, one directory per language (`implementations/go`, `implementations/typescript`). Each bundles a reusable SDK plus a CLI and is validated against the shared conformance vectors — see [`IMPLEMENTATIONS.md`](./IMPLEMENTATIONS.md).

```
blockparty/
├── implementations/  # Reference implementations, one dir per language (go, typescript)
├── conformance/      # Shared cross-client conformance vectors
├── protocol/         # Specs, whitepaper, and protobuf definitions
├── site/             # Source for bpprotocol.org (docs + essays)
├── foundations/      # Cultural and philosophical essays
├── assets/           # Logos, banners, shared branding
└── docs/             # Manifesto, roadmap, monorepo rationale
```

---

## ✦ Core Documents

- 📘 [Whitepaper](./protocol/whitepaper.md) — Protocol design and rationale
- 📑 [Specs](./protocol/specs/index.md) — Type definitions, derivation logic, block structure
- 🕊️ [Manifesto](./docs/manifesto.md) — Social and moral foundation
- 🧠 [Foundations](./foundations/) — Essays exploring design values, political context, and cultural resonance
- 📍 [Monorepo Rationale](./docs/monorepo.md) — Why everything lives here

---

## ✦ Key Concepts

- **Burnable Identities** — Revoke your root key to invalidate prior blocks by context, not deletion
- **Post-Truth State** — Blocks remain verifiable but lose trust once the root is burned
- **Audience-Scoped Encryption** — Encrypt to cryptographic audiences, not user accounts or servers
- **Delay-Tolerant Transmission** — Transport blocks over Bluetooth, QR codes, or Mars relays
- **Memory-by-Mirroring** — Nothing persists unless others choose to mirror it

---

## ✦ Versioning

Each subproject maintains its own `VERSION` file:
- `protocol/VERSION` — Current protocol spec version
- `client/VERSION` — Reference client version
- `node/VERSION` — Reference node version

Major protocol changes are documented in [CHANGELOG.md](./CHANGELOG.md).

---

## ✦ License

This project is released under the [CC0 Public Domain Dedication](https://creativecommons.org/publicdomain/zero/1.0/).
You may fork, adapt, reuse, or ignore it freely.

[![Public Domain](https://licensebuttons.net/p/mark/1.0/88x31.png)](http://questioncopyright.org/promise)

> ["Make art not law"](http://questioncopyright.org/make_art_not_law_interview) — Nina Paley

---

## ✦ Canonical Site

Rendered documentation and essays are published at:
👉 [**https://bpprotocol.org**](https://bpprotocol.org)

This GitHub repo remains the canonical source of truth for protocol evolution and implementation.

---

## ✦ Contributing

Pull requests are welcome for:
- Typo fixes and clarifications
- Proposed spec improvements or extensions
- New essays or philosophical commentary (in `foundations/`)

Major protocol extensions should be proposed via RFC-style documents under `protocol/specs/` and discussed in the open.

---

## ✦ Contact

Until formal governance is established, the protocol is maintained by its creator:
📩 [contact@bpprotocol.org](mailto:contact@bpprotocol.org)
