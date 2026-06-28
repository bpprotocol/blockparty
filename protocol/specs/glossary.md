---
Title: BlockParty Glossary
Version: 0.1.0
Last Updated: 2026-06-28
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/glossary
---

# BlockParty Glossary

A reference of core terms used throughout the BlockParty Protocol and ecosystem.

---

| Term | Definition |
|:-----|:-----------|
| **Block** | The atomic unit of signed, encrypted expression in BlockParty. Every action—post, reaction, request—is a cryptographically signed block. |
| **World** | A deterministic, salt-derived namespace that scopes type codes, audience codes, and identity operations. Worlds are gated by knowledge, not protocol-enforced permissions. |
| **Audience** | A cryptographic scoping mechanism that defines who can decrypt and interpret a block. Each audience has a public **audience code** (an addressing label) and a secret **audience key** (decrypts the payload). Derived per World. |
| **Public Audience** | A special class of audience with a well-known derivation (`bpprotocol.org/v1/audience/public-N`), enabling open discovery and public communication within a World. |
| **Private Audience** | An audience whose secret is established by an ML-KEM768 connection handshake between identities; its code is derived from the shared secret and is unlinkable to the participants. |
| **Inbox Audience** | A per-identity rendezvous audience (`bpprotocol.org/v1/audience/inbox/<address>`) any World member can derive, used to deliver connection-handshake blocks. |
| **AEAD** | Authenticated Encryption with Associated Data. BlockParty encrypts block payloads with XChaCha20-Poly1305, binding block metadata as associated data so tampering is detectable. |
| **KEM** | Key Encapsulation Mechanism. BlockParty uses ML-KEM768 (Kyber) to establish shared secrets for audience encryption and connections. |
| **Co-Signature** | An optional additional Dilithium signature by another identity over a block (`sigs.extra[]`), expressing endorsement or witnessing. Verified independently; consensus schemes are extension-layer. |
| **Mirror Node** | Any participant or device that voluntarily stores and rebroadcasts blocks. Mirrors are essential for persistence but are not mandatory for network operation. |
| **Burnable Identity** | A user identity model where revealing the root private key invalidates all previous trust, allowing repudiation of past activity. |
| **Root Key** | The fundamental signing and encryption keypair from which all other keys and trust relationships are derived. |
| **Identity Block** | A block that announces an identity address and associated metadata to a World, enabling discovery. Not required for identity existence. |
| **Connection** | A peer relationship established through private key exchanges, enabling scoped audience communication between identities. |
| **Rotation** | The process of regenerating audience keys or rotating connections to revoke access or sever stale relationships without burning the root identity. |
| **Post-Truth State** | The condition after an identity's root private keys have been publicly revealed (burned), rendering all communications by that identity — past and future — contestable, since anyone can now forge them. |
| **Block ID** | A unique identifier derived from the canonical contents of a block. Used to link, reference, and validate blocks. |
| **Type Code** | A world-scoped hash derived from a block type URN. Defines the functional intent of a block. |
| **Type URN** | A canonical namespaced string (e.g., `bpprotocol.org/v1/types/content.post`) describing a block’s semantic purpose before hashing into a type code. |
| **MIME Headers** | Key-value metadata headers used in `content.post` and similar block types to describe payloads semantically (e.g., content type, language, tags). |
| **RPC Request** | A block type representing a structured remote procedure call request to another participant. Not tied to any particular transport. |
| **RPC Render** | A block type used for local-only, dynamic content rendering, allowing ephemeral client-driven UI generation without block creation. |
| **Transport Agnosticism** | BlockParty's model of treating transport and storage mediums as external to the protocol—blocks can move over any medium that can deliver bytes. |
| **Sneakernet** | Physical transport of blocks via hard drives, USB sticks, or any non-networked medium. A key survival strategy for disconnected environments. |
| **Delay-Tolerant Networking (DTN)** | Communication model where delivery can be delayed for extended periods; supported natively by BlockParty's block structure and independence from real-time networking assumptions. |
| **Canonical Specification** | The evolving primary version of the BlockParty Protocol, coordinated through the [blockparty-protocol GitHub repository](https://github.com/bpprotocol/blockparty-protocol). |
| **Fragmentation Risk** | The possibility that Worlds or clients may evolve independently, leading to divergence—accepted as a feature of resilience, not a flaw. |

---

This glossary will evolve alongside the protocol specifications. Contributions welcome.

