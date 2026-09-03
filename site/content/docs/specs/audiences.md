---
title: Audiences
version: 0.1.0
updated: 2026-06-28
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/audiences
---

# Audiences

An **audience** is the cryptographic scope that decides who can decrypt and interpret a block. BlockParty does not enforce access with servers, accounts, or ACLs — access is governed entirely by who can derive an audience's key. This document defines the audience model, the well-known **public audiences**, and how audience codes and keys are derived.

See also: [Derivations](./derivations.md) (`GetAudienceCode`), [Block Encryption](./encryption.md) (audience secret → content key), and [Connections](./connections.md) (private audiences).

---

## 🧭 Audience Code vs Audience Key

Two distinct values are derived for every audience; do not conflate them:

| Value | Visible in block? | Purpose | Derivation |
|-------|-------------------|---------|------------|
| **Audience code** (`audience_code`) | Yes (block metadata) | Addressing/routing label; lets clients group and filter blocks | `GetAudienceCode(world, audienceID)` — [Derivations](./derivations.md) |
| **Audience secret / key** | No (never transmitted) | Decrypts the `data` payload | [Block Encryption](./encryption.md) |

Knowing an `audience_code` lets a client *recognize* blocks for an audience; it does **not** grant the ability to decrypt them. Only the audience secret does that.

---

## 🌐 Audience Kinds

### Private Audiences
Established between connected identities through an ML-KEM768 handshake. The audience secret is the connection's shared secret. Membership is whoever completed the handshake (1:1, or a small group seeded from pairwise connections). Fully specified in [Connections](./connections.md).

### Public Audiences
A finite, well-known set of audiences that any participant in a World can derive, enabling open "town square" spaces and client bootstrapping. Public audiences are **confidential to the World** (an outsider without the World seed still cannot derive them) but **open within it** (any World member can read and publish).

---

## 📣 Public Audience Identifiers

Public audiences use standardized identifiers under the `bpprotocol.org/v1/audience/` namespace, distinguished by a numeric suffix:

```
bpprotocol.org/v1/audience/public-1
bpprotocol.org/v1/audience/public-2
bpprotocol.org/v1/audience/public-3
...
bpprotocol.org/v1/audience/public-16
```

- **Reserved set:** `public-1` through `public-16` are reserved as the canonical well-known public audiences. Clients SHOULD bootstrap discovery by subscribing to `public-1` (the default "lobby") first.
- **Extensible:** higher numbers (`public-17`, `public-18`, …) are valid and reserved for future standardization; clients may use them by convention but interoperability is only guaranteed for the reserved set.
- **World-scoped:** the resulting `audience_code` and key are still derived through the World's `AudienceSalt`, so `public-1` in World A is cryptographically unrelated to `public-1` in World B. There is no global public audience — only per-World ones.

---

## 🧮 Deriving a Public Audience

```ts
// Addressing label (visible in block metadata)
audienceID  = "bpprotocol.org/v1/audience/public-1"
audienceCode = GetAudienceCode(world, audienceID)        // see Derivations

// Decryption key (never transmitted)
audienceSecret = PublicAudienceSecret(world, audienceID) // see Encryption
```

Both derivations require only the World context and the public identifier string. Any World member can therefore publish to and read from a public audience without coordination, while the content remains opaque to anyone outside the World.

---

## 🔎 Discovery & Bootstrapping

The protocol does not mandate a discovery mechanism, but the public audiences make organic bootstrapping possible:

1. A new client that knows a World seed derives `public-1`'s code and key.
2. It listens on whatever transports it has (libp2p topic, mirror, local exchange) for blocks bearing that `audience_code`.
3. From `identity` and `client.rallypoint` blocks seen there ([Client Extension](./extensions/client.md)), it discovers peers and can initiate private [connections](./connections.md).

Because participation is voluntary mirroring rather than server-enforced membership, the **Discovery vs Privacy** trade-off described in the [Whitepaper](../whitepaper.md) §12 applies: broad publication to a public audience increases discoverability at the cost of metadata exposure within the World. Clients SHOULD make this trade-off explicit to users.

---

## 🔗 Related Specifications

- [Derivations](./derivations.md)
- [Block Encryption](./encryption.md)
- [Connections](./connections.md)
- [Client Extension](./extensions/client.md)
