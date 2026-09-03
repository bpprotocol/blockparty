---
title: BlockParty Protocol Extensions
version: 0.1.0
updated: 2025-05-01
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/extensions/
---

# BlockParty Protocol Extensions

The BlockParty Protocol is intentionally minimal at its core, enabling rich extension by developers, implementers, and communities.

This directory contains specifications and reference implementations for optional block types, protocol behaviors, and domain-specific models that extend the base protocol.

---

## ✦ Philosophy

BlockParty is designed to be:
- 🔌 **Extensible by design** — Anyone can define new block types
- 🌐 **World-scoped and isolated** — Types are resolved per World, avoiding global conflicts
- 🧠 **Composable, not monolithic** — New functionality builds atop the core, rather than bloating it

---

## ✦ Contribution Guidelines

To propose a new extension:
1. Define a namespace URN (e.g. `urn:com.examplecorp:blocktypes:product.offer`)
2. Include schema examples and semantic behavior
3. Provide rationale for its existence and audience scope
4. Add your spec under this directory (e.g. `specs/extensions/commerce.md`)

---

## ✦ Examples
- [`client.md`](./client.md) – Ephemeral presence, coordination, and discovery primitives for peer clients  
- [`dm.md`](./dm.md) – Secure, audience-scoped direct messaging with reply/thread support  
- `commerce.md` – Peer-to-peer storefronts and invoicing
- `games.md` – Turn-based state exchange (e.g. chess)
- `calendar.md` – Scheduling and time-based coordination

---

> The core protocol exists to enable expressive, humane communication.  
> Extensions exist to make it practical, powerful, and personal.

Contributions welcome.

