---
Title: BlockParty Protocol Whitepaper
Version: 0.1.0
Last Updated: 2026-06-28
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/whitepaper
---

# BlockParty Protocol

### A Post-Censorship Social Protocol for the Network-Agnostic Era

## Abstract

BlockParty is a decentralized, transport-agnostic social protocol designed for a post-censorship world. Prioritizing local-first ownership, audience-scoped encryption, and post-quantum security, it enables free communication and commerce across diverse environments—from resilient mesh networks to interplanetary links. BlockParty introduces a humane model of identity and trust that is both revocable and ephemeral, redefining permanence as a user choice rather than an infrastructural guarantee. This whitepaper outlines the core concepts, primitives, cryptographic foundations, and philosophical approach behind the protocol.

## Table of Contents

- [1. Introduction](#1-introduction)
  - [1.1 Problem Statement](#11-problem-statement)
  - [1.2 Related Protocols and Differences](#12-related-protocols-and-differences)
  - [1.3 Real-World Use Cases](#13-real-world-use-cases)
- [2. Design Principles](#2-design-principles)
- [3. Core Concepts](#3-core-concepts)
  - [Blocks](#blocks)
  - [Audiences](#audiences)
    - [Public Audiences](#public-audiences)
  - [Worlds](#worlds)
  - [Burnable Identity](#burnable-identity)
  - [Mirror-Based Persistence](#mirror-based-persistence)
- [4. Identity & Trust Model](#4-identity--trust-model)
  - [Key Derivation & Identity Recovery](#key-derivation--identity-recovery)
  - [Identity Lifecycle](#identity-lifecycle)
  - [Trust Philosophy](#trust-philosophy)
- [5. Block Type Primitives](#5-block-type-primitives)
- [6. Extension Mechanism](#6-extension-mechanism)
- [7. Protocol & Transport Layer](#7-protocol--transport-layer)
- [8. Security & Cryptography](#8-security--cryptography)
- [9. Resilience & Philosophy](#9-resilience--philosophy)
  - [Foundational Values](#foundational-values)
- [10. Future Directions](#10-future-directions)
- [11. Governance and Evolution](#11-governance-and-evolution)
- [12. Risks and Challenges](#12-risks-and-challenges)
- [13. Glossary](#13-glossary)
- [14. Appendix](#14-appendix)
- [15. Conclusion](#15-conclusion)
- [16. License & Contribution](#16-license--contribution)

## 1. Introduction

BlockParty is a decentralized, post-quantum secure social protocol designed to resist censorship, centralization, and platform dependency. It enables asynchronous communication, peer-to-peer commerce, and private identity management—even in high-latency, low-connectivity, or interplanetary environments.

This whitepaper presents the design, purpose, and primitives of BlockParty, which combines strong cryptographic guarantees with a humane, ephemeral, and world-scoped communication model.

## 1.1 Problem Statement

Modern digital communication is fragile. It relies heavily on centralized platforms, trusted intermediaries, and infrastructural stability—assumptions that fail under censorship, authoritarian control, corporate capture, infrastructure collapse, or hostile environments.

Existing social systems suffer from:

- **Platform dependency**: Speech rights are delegated to corporations or governments.
- **Network fragility**: Communication collapses under network partitions or high-latency conditions.
- **Identity permanence**: Traditional identities are permanent and traceable, even when circumstances change.
- **Cryptographic obsolescence**: Widely deployed cryptosystems are vulnerable to future quantum attacks.
- **Monolithic design**: Extending or adapting platforms requires their permission, not user innovation.

BlockParty is designed to address these failures: enabling human communication that persists, adapts, and protects itself in hostile or disconnected environments—without reliance on centralized authorities or future-fragile cryptography.

## 1.2 Related Protocols and Differences

While BlockParty shares some goals with existing decentralized protocols, it differs significantly in architecture, trust model, and design philosophy:

| Protocol                                                     | Similarities                                                                              | Key Differences                                                                                                                                                                                       |
| ------------------------------------------------------------ | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Nostr**                                                    | Both use cryptographic identities and propagate signed messages.                          | Nostr assumes all content is publicly relayable. BlockParty uses encrypted, audience-scoped communication by default and treats memory as optional rather than permanent.                             |
| **IPFS**                                                     | Both prioritize content-addressed storage and resilience against infrastructure failures. | IPFS focuses on permanent storage and availability. BlockParty focuses on ephemeral, optional persistence—content survives only if mirrored intentionally.                                            |
| **Blockchain-based Social Networks (e.g., Lens, Farcaster)** | Both envision user-controlled identity and data.                                          | Blockchain-based systems rely on global consensus, introducing scalability bottlenecks and public traceability. BlockParty is purely peer-scoped, permissionless, and avoids global ledgers entirely. |
| **Secure Scuttlebutt (SSB)**                                 | Both enable decentralized, peer-to-peer social graphs without centralized servers.        | SSB ties identity to an immutable feed. BlockParty identities are burnable, ephemeral, and audience-scoped, allowing selective revocation and post-truth states.                                      |
| **Dat/Hypercore Protocol**                                   | Both support peer-to-peer data synchronization.                                           | Dat/Hypercore emphasize versioned data sharing. BlockParty emphasizes asynchronous, audience-encrypted messaging that may or may not persist.                                                         |
| **ActivityPub (e.g., Mastodon)**                             | Both support decentralized interaction without centralized ownership.                     | ActivityPub still relies heavily on server intermediaries ("instances"). BlockParty operates directly peer-to-peer and minimizes infrastructure assumptions.                                          |

## 1.3 Real-World Use Cases

BlockParty is designed for diverse, resilient, and sometimes extreme environments where traditional communication platforms fail. Example scenarios include:

- **Interplanetary Messaging (Earth ↔ Mars):** BlockParty's asynchronous, delay-tolerant design makes it ideal for extremely high-latency links, such as Mars-Earth communications, where traditional networking models break down.

- **Censorship-Resistant Journalism:** In authoritarian regimes or regions with frequent internet blackouts, journalists and citizens can share encrypted, audience-scoped content over ad-hoc networks, portable storage, or mesh radios.

- **Offline Festivals and Remote Expeditions:** Communities at music festivals, research outposts, or remote expeditions can maintain peer-to-peer social communication using local transports like Bluetooth, NFC transfers, isolated WiFi hotspots, or USB drops, even without centralized infrastructure.

## 2. Design Principles

- **Local-first:** You own your keys, you store your blocks, you define your experience.
- **Audience-scoped privacy:** All blocks are encrypted based on derived audiences.
- **Post-quantum security:** Built on ML-KEM768 (formerly CRYSTALS-Kyber) and ML-DSA-65 (FIPS 204, formerly CRYSTALS-Dilithium) to anticipate the cryptographic future.
- **Burnable identity:** You can walk away. The protocol supports post-truth repudiation.
- **Transport-agnostic:** Blocks can be transferred via any medium—p2p, QR, even sneaker-net.
- **Extensible, not monolithic:** The core is small. Everything else is an optional extension.

## 3. Core Concepts

### Blocks

The atomic unit of all expression. Every action—post, reaction, request—is a cryptographically signed block.

By default, BlockParty blocks are opaque. Only structural metadata such as version number, block ID, and cryptographic signatures are directly interpretable without additional context. Other elements—such as type codes, audience codes, and data payloads—are world-specific derivations and remain unintelligible without proper keys and mappings.

A block contains the following structured components:

- **Version**: Protocol version the block conforms to.
- **Block ID**: Unique identifier derived from the block's contents.
- **Type Code**: A world-scoped hash representing the type of action or content.
- **Audience Code**: A world-scoped hash indicating the intended audience's cryptographic scope.
- **Timestamp**: Time of block creation.
- **Data Blob**: Encrypted payload meant for the designated audience.
- **World Signature**: ML-DSA-65 signature binding the block to its originating World.
- **Author Signature**: Signature by the author's root signing key, validating the author's endorsement of the world-signed block.

Without possession of world-specific type mappings or audience keys, blocks appear as opaque, structured artifacts. Interpretation and decryption are possible only within the correct cryptographic and world contexts.

### Audiences

Audience codes are deterministically derived using the scoped world's audience salt. Each audience defines a cryptographic context that governs which keys are used to encrypt and decrypt associated blocks. Access to content is determined by knowledge of the correct audience derivation, not enforced by the protocol itself. Without the right audience knowledge, blocks remain unintelligible.

#### Public Audiences

To facilitate easier peer discovery and shared spaces, BlockParty defines a finite set of well-known public audiences. These are deterministically derived using standardized identifiers:

```
bpprotocol.org/v1/audience/public-1
bpprotocol.org/v1/audience/public-2
bpprotocol.org/v1/audience/public-3
...etc.
```

**Key properties:**

- **Open Access:** Anyone knowing the public audience derivation can publish to and read from these spaces.
- **Discovery:** New clients may prioritize connecting to public audiences to bootstrap social interactions.
- **Extensibility:** New public audiences can be added by simple numeric extension.
- **World-scoped:** Public audiences are still derived through the World's audience salt, so `public-1` in one World is cryptographically unrelated to `public-1` in another. They are open *within* a World, not globally.
- **No Enforcement:** Participation is based on voluntary mirroring; there are no central servers or coordinators.

Public audiences act as distributed "town squares" without compromising the decentralized, censorship-resistant nature of the protocol. The reserved set (`public-1` … `public-16`), their derivation, and bootstrapping are specified in the [Audiences specification](https://bpprotocol.org/specs/audiences).

Audience codes are deterministically derived using the scoped world's audience salt. Each audience defines a cryptographic context that governs which keys are used to encrypt and decrypt associated blocks. Access to content is determined by knowledge of the correct audience derivation, not enforced by the protocol itself. Without the right audience knowledge, blocks remain unintelligible.

### Worlds

Worlds are deterministic derivations—self-contained universes generated from a known or guessed seed value. Anyone who knows the seed can independently derive the same World code, allowing interoperability without centralized coordination.

Within a World, type codes and audience codes are also deterministically derived using the world's unique salt. This scoping ensures that even identical type names or audience intents hash differently across Worlds, preserving their autonomy and isolation.

Access to a World is gated purely by knowledge: the protocol itself does not prevent outsiders from interacting. However, without knowledge of the world seed, an outsider cannot derive valid audience codes, cannot understand type codes, and cannot meaningfully address scoped blocks. Cryptographic obscurity—not permissioning—forms the practical shield. There are no ACLs, "accounts," or "groups" hardcoded in the protocol.

### Burnable Identity

Root key exposure renders all past actions contextually invalid within BlockParty. By intentionally revealing or leaking the root key, a user can sever the trust chain associated with their previous communications, signaling that all prior actions should no longer be considered authentic or attributable. This capability supports fresh starts, plausible deniability, and strategic repudiation when needed—all without relying on external revocation authorities.

### Mirror-Based Persistence

Content exists only as long as someone is mirroring it or it’s online. The internet is not forever, by design.

## 4. Identity & Trust Model

Each user possesses a root keypair that serves as the foundation for all other derived keys, including signing keys, encryption keys, and audience-specific keys. This hierarchy ensures cryptographic continuity while allowing flexibility and control over trust relationships.

### Key Derivation & Identity Recovery

Identities in BlockParty are derived deterministically from a user-provided passphrase and a world-specific salt, much like a cryptocurrency wallet. This allows users to recreate their identities anywhere, at any time, using only their phrase and the world information.

The process works as follows:

- Derive a world password from the passphrase and the World's wallet salt (so the same passphrase yields a different identity in each World).
- Derive a root keypair for post-quantum encryption (ML-KEM768) from a `mlkem`-domain-separated seed.
- Derive a root signature keypair for post-quantum signing (ML-DSA-65) from a `ml-dsa`-domain-separated seed.
- Generate the address by hashing **both** public keys into a shortened, content-binding address.

#### Pseudocode

```text
func OpenIdentity(world, walletOptions):
  worldPassword = HKDF(walletOptions.passphrase, world.walletSalt, "bpprotocol.org/v1/identity")

  kyberKey     = MakeKyberPair(HKDF(worldPassword, "", "mlkem"))
  mldsaKey = MakeMLDSAPair(HKDF(worldPassword, "", "ml-dsa"))

  return Identity{
    address: BytesToAddress(mldsaKey.public, kyberKey.public),
    rootKEM: kyberKey,
    rootSig: mldsaKey,
  }
end
```

This model ensures that identities are portable, regenerable, and revocable, while tying them securely to specific World contexts. The canonical algorithm — including `BytesToAddress`, which binds both public keys — is defined in the [Identity](https://bpprotocol.org/specs/identity) and [Derivations](https://bpprotocol.org/specs/derivations) specifications.

### Identity Lifecycle

- **Create:** An identity block is generated and published, announcing the identity address and optional metadata to an audience. Identities themselves exist independently of the block; the block simply makes the existence of the identity discoverable and contextual within a World.
- **Connect:** A two-message post-quantum handshake (ML-KEM768 key agreement) establishes a shared secret from which a private audience is derived, allowing scoped, confidential communication. See the [Connections specification](https://bpprotocol.org/specs/connections).
- **Rotate:** Users advance a connection to a new epoch with fresh key material (`connect.rotate`), severing stale or compromised relationships without abandoning their entire identity.
- **Burn:** Publishing an `identity.burn` block that reveals the identity's root private keys marks the entire identity lineage as contested, achieving a "post-truth" state and enabling repudiation of past activity. See the [Identity Burn RFC](https://bpprotocol.org/specs/identity-burn-rfc).

### Trust Philosophy

Trust within BlockParty is:

- **Situational:** Established per interaction or context, not globally assumed.
- **Scoped:** Limited to specific audiences or Worlds, rather than broad-based networks.
- **Revocable:** Easily severed by rotating keys or burning identities, ensuring that control always remains with the user.

## 5. Block Type Primitives

BlockParty's protocol is composed of a minimal set of signed block types. Each is identified by a canonical namespaced string (e.g. `bpprotocol.org/v1/types/content.post`) and hashed per World to generate a type code.

| Purpose             | Block Types                                                                                                                                                                                                                                                         |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Identity & Trust    | `bpprotocol.org/v1/types/identity`, `bpprotocol.org/v1/types/identity.burn`                                                                                                                                                                                         |
| Connection          | `bpprotocol.org/v1/types/connect.request`, `bpprotocol.org/v1/types/connect.response`, `bpprotocol.org/v1/types/connect.identity`, `bpprotocol.org/v1/types/connect.rotate`, `bpprotocol.org/v1/types/connect.close`                                                |
| Social Content      | `bpprotocol.org/v1/types/content.post`, `bpprotocol.org/v1/types/content.reaction`, `bpprotocol.org/v1/types/content.tag.request`, `bpprotocol.org/v1/types/content.tag.accept`, `bpprotocol.org/v1/types/content.gallery`, `bpprotocol.org/v1/types/content.quote` |
| Chunked Content     | `bpprotocol.org/v1/types/content.chunked.manifest`, `bpprotocol.org/v1/types/content.chunked.block`                                                                                                                                                                 |
| Transport & Storage | `bpprotocol.org/v1/types/chunk.manifest`, `bpprotocol.org/v1/types/chunk.block`                                                                                                                                                                                     |
| RPC & Dynamic UX    | `bpprotocol.org/v1/types/rpc.request`, `bpprotocol.org/v1/types/rpc.response`, `bpprotocol.org/v1/types/rpc.render`                                                                                                                                                 |

These types cover all core social, communicative, and structural functions of the protocol.

## 6. Extension Mechanism

BlockParty Protocol is intentionally minimal at its core, enabling rich extensibility by developers, implementers, and communities. Every block type is defined as a namespaced string, hashed per World. Core types follow the `bpprotocol.org/v1/types/` pattern.

Third parties are encouraged to define their own extensions using their own domain namespaces to ensure global uniqueness and interoperability. This extensibility model allows clients to add new functionality without altering the core protocol.

Examples of extensions include:

- **Commerce:** `commercehub.io/v1/types/invoice`, `commercehub.io/v1/types/product.catalog`
- **Games:** `gamespace.dev/v1/types/chess.move`, `gamespace.dev/v1/types/lobby.invite`
- **Scheduling:** `calendar.app/v1/types/invite`, `calendar.app/v1/types/decline`

Additionally:

- New block types can be introduced freely.
- New `content.post` MIME header content-types can be defined and selectively supported.
- New RPC methods can be created and invoked using `rpc.call` blocks, without requiring any core protocol modification.
- Custom MIME headers can be added alongside standard ones for specialized handling.

Clients may selectively support, ignore, or extend these features. They may also create client-specific extensions, allowing rich, divergent ecosystems to evolve independently while maintaining interoperability through clear namespace conventions.

## 7. Protocol & Transport Layer

BlockParty is transport-agnostic. It defines the structure, signing, and encryption of blocks, but does not enforce or manage transport mechanisms. Transport is entirely out-of-band: moving a block is simply a matter of serializing it into bytes and transmitting it via any medium the user chooses. Reliability, retries, and connection establishment are left to the discretion of clients or external systems.

Blocks can be transmitted and stored across a wide range of methods, including but not limited to:

- Libp2p/DHT
- Bluetooth
- QR codes
- USB drives
- Mars-to-Earth delay-tolerant links

Storage and transmission are fundamentally treated the same—serialization of blocks for resilience and portability. Client implementations are free to select, combine, or innovate new transport methods, reinforcing the core ethos: "nothing can stop the signal."

## 8. Security & Cryptography

BlockParty integrates cryptography that anticipates both current and post-quantum threat models:

Each block's authenticity is layered through two ML-DSA-65 signatures:

- **World Signature**: A ML-DSA-65 signature, produced by the World's signing key, validating that the block belongs to a specific World. It signs over the block ID, version, type code, audience code, timestamp, and encrypted data blob. Any client can verify this signature to confirm world membership without decrypting the block.
- **Author Signature**: A signature by the author's root ML-DSA-65 signing key, endorsing the previously generated World Signature. This authenticates authorship while preserving the World context.

Further cryptographic protections include:

- **ML-KEM768 (formerly CRYSTALS-Kyber)** for post-quantum key agreement, establishing the shared secret from which audience encryption keys are derived.
- **ML-DSA-65** for post-quantum signatures — author, World, and co-signatures alike.
- **Per-block AEAD encryption** using XChaCha20-Poly1305, keyed via HKDF-SHA256 from the audience secret, with block metadata bound as additional authenticated data, ensuring each block's payload is accessible only to the intended audience and that metadata is tamper-evident. See the [Block Encryption specification](https://bpprotocol.org/specs/encryption).
- **World signature + Author signature** are layered onto each block, validating both global (World) context and individual authorship.
- **Optional co-signatures** let additional identities endorse or witness a block. The core treats each as an independent, per-signer endorsement; threshold, quorum, and multi-party-consensus schemes are built on top of this primitive as extensions, not assumed by the core.

Transport security is external to the protocol; BlockParty focuses strictly on block-level integrity and confidentiality. Block validity is subordinate to trust state. If an identity enters post-truth state, previously valid blocks may be disregarded.

## 9. Resilience & Philosophy

BlockParty is built to endure where other systems fail — in disconnected, adversarial, and collapsing environments:

- Without continuous internet connectivity
- Amidst censorship regimes and hostile infrastructure
- Across planetary distances, such as between Mars and Earth
- On isolated, airgapped systems

BlockParty is delay-tolerant by design. It carries not just messages, but memory itself — portable, sovereign, and cryptographically verifiable.  
Where traditional networks fracture under stress, BlockParty persists through any transport: sneaker-net, radio relays, delay-tolerant networks, or physical exchange.

"Nothing can stop the signal" is not just an aspiration. It is an architectural principle.

Even if all infrastructures fail, BlockParty ensures that identity, memory, and association survive — because they are owned by individuals, not intermediaries.

---

### Foundational Values

BlockParty is more than a protocol.  
It is a new architecture for voluntary civilization.

We believe:

- **Privacy is a right, not a privilege.** Consent defines exposure.
- **Identity is portable and revocable.** It is not assigned by states, platforms, or registries.
- **Friendship and communication are acts of mutual consent.** They are encrypted, deniable, and voluntary.
- **Memory is sacred.** It should be earned by trust, not extracted by surveillance.
- **Forgetting is as essential as remembering.** Permanence is a user choice, not an infrastructural decree.
- **Association is sovereignty.** Worlds form, fork, and evolve by consent — not by enforced consensus.

BlockParty does not build platforms to capture.  
It enables worlds to live.

It is an architecture for post-censorship, post-fragility communication — where memory, trust, and freedom are carried by individuals, not administered by systems.

For a deeper exploration of the principles that shape BlockParty, see the [BlockParty Manifesto](https://bpprotocol.org/manifesto) and the [Foundations of BlockParty Anthology](https://bpprotocol.org/foundations).


## 10. Future Directions

BlockParty’s minimalist core leaves room for organic, community-driven innovation. Future expansions may deepen functionality while preserving the protocol’s foundational values of decentralization, ephemeralism, and user sovereignty. Key directions include:

### Dynamic User Experiences
- **Dynamic components via `rpc.render`**: Enabling clients to generate local, ephemeral content experiences without permanent block creation or server calls. A `rpc.render` is a **local-only** instruction — never serialized, signed, or transmitted — that a client turns into a `content.post`-shaped view. Supports more interactive, personalized interfaces without sacrificing resilience.

### Portable Services and Social Graphs
- **Portable services**: Developing shared inboxes, friend graphs, and lightweight reputation systems that function across Worlds and transports without central servers.

### Semantic Discovery and Contextual Trends
- **Search and trend analysis**: Empowering Worlds to index, surface, and recommend content based on semantic tagging extensions, allowing for organic discovery without sacrificing audience-scoped privacy.

### Standards for Decentralized Identity Verification
- **Client-driven identity standards**: Explore standardized, decentralized methods for identity verification across Worlds and clients. Possible approaches include leveraging DNS records for identity attestations, shared public key infrastructures, or Web-of-Trust style signatures—enhancing user confidence while preserving BlockParty’s decentralized and voluntary principles.

### Ecosystem Markets and Layered Services
- **Content marketplaces and service layers**: Supporting voluntary economic ecosystems through peer-to-peer commerce, reputation overlays, and cooperative mirroring without central payment processors or intermediaries.

These directions are intended as invitations, not prescriptions. Communities and implementers are encouraged to propose, fork, or extend as needed—ensuring that BlockParty remains dynamic, diverse, and fundamentally human-centered.


## 11. Governance and Evolution

BlockParty Protocol is released into the public domain under CC0.  
There is no central authority, owner, or mandatory governance structure.  
The protocol is free to be adopted, extended, forked, or evolved by any community, developer, or userbase.

Future iterations, extensions, or standards built atop BlockParty may emerge organically through ecosystem collaboration, but BlockParty itself imposes no formal governance or upgrade mechanism. Protocol freedom means that different Worlds, audiences, or clients may evolve independently—preserving diversity and resilience even if consensus is not universal.

Collaboration may occur through platforms like GitHub discussions ([github.com/bpprotocol/blockparty-protocol](https://github.com/bpprotocol/blockparty-protocol)), community forums, or World-specific channels, though no formal process is mandated. Communities are encouraged to propose enhancements, discuss interoperability standards, and share best practices through whichever mediums best fit their needs.

The canonical version of the BlockParty Protocol is coordinated through the github.com/bpprotocol/blockparty-protocol repository.
While independent forks and alternative specifications are welcome under the CC0 license, proposals for changes to the canonical protocol—such as new block types, cryptographic migrations, or standard extensions—should begin through this repository to foster community visibility and voluntary coordination.

For instance, a community might propose a new block type like `bpprotocol.org/v1/types/content.poll` via GitHub, discuss its adoption in open forums, and implement it within specific Worlds or clients based on voluntary consensus.

Conflicts may naturally arise—such as divergent extensions, competing World policies, or differing client behaviors. BlockParty embraces informal conflict resolution: communities are encouraged to coordinate through open dialogue where possible, or fork and evolve independently when necessary.

As noted in the [Risks and Challenges](#12-risks-and-challenges) section, independent evolution may lead to fragmentation. However, BlockParty treats diversity as a strength: resilience emerges from a multiplicity of approaches, not enforced uniformity. Diversity ensures no single point of failure, allowing BlockParty to thrive even if some Worlds or clients diverge.

Cryptographic migrations, as discussed under [Cryptographic Evolution](#12-risks-and-challenges), may similarly be proposed as specifications. Communities can coordinate via open channels to ensure smooth, opt-in transitions to updated cryptographic standards without requiring centralized control.

## 12. Risks and Challenges

While BlockParty is designed for resilience and autonomy, certain risks and challenges must be acknowledged:

### Cryptographic Evolution

BlockParty relies on post-quantum cryptographic standards such as ML-KEM768 and ML-DSA-65. Although these algorithms have been vetted through rigorous processes like NIST PQC, the future evolution of cryptanalysis could expose vulnerabilities.
BlockParty mitigates this risk through discrete block versioning: each block records its protocol version explicitly, allowing future updates to cryptographic primitives without retroactively invalidating historical data.
Ongoing monitoring, community consensus, and orderly migration plans will be essential to maintaining security over decades.

### Discovery vs Privacy Tension

BlockParty emphasizes audience-scoped privacy by default. Broader discovery—for example, finding identities or content within shared public audiences—is driven by client-level strategies rather than enforced by the protocol itself.
Clients may implement their own methods for locating blocks scoped to known public audiences. While this enables organic discoverability, it introduces potential trade-offs between user exposure, metadata correlation, and usability.
Careful ecosystem design, informed client defaults, and user education will be necessary to balance openness and privacy without compromising the protocol’s foundational cryptographic guarantees.

### Storage and Persistence Fragility

Block persistence depends on voluntary mirroring and self-storage. In hostile or low-resource environments, content availability may degrade if users or mirrors do not actively retain and relay blocks. This risk is intentional (favoring ephemeralism over enforced permanence), but may surprise users accustomed to traditional infrastructure.  
While individuals can persist their own content, it is also anticipated that ecosystem participants—including major infrastructure providers—may offer managed mirroring services to enhance reliability without reintroducing central control.

### User Trust Mismanagement

BlockParty empowers users with self-sovereign identity and trust models, but this decentralization places full responsibility on individuals. Users who fail to rotate compromised keys, misunderstand burn procedures, or mishandle private keys may face unrecoverable loss of relationships or content access.
This risk is behavioral rather than technical: cultural adoption of best practices, coupled with thoughtful client tooling—such as guided key rotation, burn initiation flows, and backup education—can mitigate much of the gap. Nonetheless, the "burn" concept is novel, and widespread user understanding will require deliberate onboarding strategies.

### Social Adoption Challenges

BlockParty diverges significantly from existing social platforms by design: ephemeral presence, burnable identity, and decentralized discovery differ from mainstream expectations of persistent profiles and centralized feeds. Educating users and designing intuitive client experiences will be critical to broader adoption.

### Transport Diversity Risks

BlockParty is transport-agnostic by design. Blocks are independently encrypted and signed before transmission, ensuring their security and tamper-evidence regardless of the transport medium.
While transport mechanisms (e.g., libp2p, NFC transfers, isolated WiFi) may vary in reliability, they do not compromise the security or integrity of the blocks themselves. If a block is corrupted or modified during transport, recipients will detect invalid signatures and reject the block automatically.
The primary risks associated with transport diversity relate to delivery reliability and latency, not data trustworthiness.

## 13. Glossary

| Term                  | Definition                                                                                   |
| --------------------- | -------------------------------------------------------------------------------------------- |
| **Block**             | A cryptographically signed unit of expression containing metadata and an encrypted payload.  |
| **World**             | A deterministic, salt-derived universe of type codes, audience codes, and identity scoping.  |
| **Audience**          | A cryptographic scoping mechanism for controlling who can decrypt and interpret content.     |
| **Mirror Node**       | Any device or participant that stores and redistributes blocks.                              |
| **Burnable Identity** | An identity model where root key exposure invalidates prior trust and allows repudiation.    |
| **Public Audience**   | A shared audience scope derived from public identifiers, enabling open communication spaces. |

## 14. Appendix

### Core Technical Details

- **Type Hashing Algorithm**\
  Block type URNs (e.g., `bpprotocol.org/v1/types/content.post`) are hashed per World using a cryptographic hash function (e.g., Keccak-256) combined with the World's type salt. This generates a unique world-scoped type code used in block metadata.

- **Canonical Block Type List**\
  BlockParty defines a minimal set of core block types covering identity, social content, connections, chunked data, and dynamic RPC interactions.\
  Full details and JSON schema examples are provided in the [Block Types and Type URNs Specification](https://bpprotocol.org/specs/block-types).

- **Example Identity Derivation Pseudocode**\
  See Section 4 for pseudocode demonstrating deterministic identity recovery based on user passphrase and World salt.

- **MIME Header Conventions**\
  Content blocks (such as `content.post` and `content.chunked.manifest`) use flat key-value headers, namespaced via domain URIs.

  - **Required header**: `bpprotocol.org/v1/content-type`
  - Clients must safely ignore unknown headers.\
    Full standards are provided in the [MIME Header Conventions Specification](https://bpprotocol.org/specs/mime-header-conventions).

- **Block Encryption**\
  Audience-scoped payloads use ML-KEM768 key agreement → HKDF-SHA256 → XChaCha20-Poly1305 AEAD, with block metadata bound as additional authenticated data. Audience secrets are established per-World for public audiences and per-connection for private ones.\
  Full details are provided in the [Block Encryption](https://bpprotocol.org/specs/encryption), [Audiences](https://bpprotocol.org/specs/audiences), and [Connections](https://bpprotocol.org/specs/connections) specifications.

### Recommended Reading

- [BlockParty Glossary](https://bpprotocol.org/specs/glossary) — Full evolving glossary of protocol terms.

- [BlockParty Manifesto](https://bpprotocol.org/manifesto) — Foundational values and guiding philosophy.

- [Block Types and Type URNs](https://bpprotocol.org/specs/block-types) — Canonical block definitions and extension guidance.

- [MIME Header Conventions](https://bpprotocol.org/specs/mime-header-conventions) — Standards for content metadata headers.

- [BlockParty Protocol Specifications (Draft)](https://github.com/bpprotocol/protocol) — Evolving canonical source for all specifications.

- [RFC 4838: Delay-Tolerant Network Architecture](https://datatracker.ietf.org/doc/html/rfc4838) — Foundational principles for store-and-forward, delay-tolerant communications.

- [NIST PQC Finalists: ML-KEM (Kyber)](https://csrc.nist.gov/Projects/post-quantum-cryptography/selected-algorithms-2022) — ML-KEM768 (formerly CRYSTALS-Kyber) for post-quantum key encapsulation.

- [NIST PQC Finalists: ML-DSA-65](https://csrc.nist.gov/Projects/post-quantum-cryptography/selected-algorithms-2022) — ML-DSA-65 signatures for post-quantum digital signature schemes.
- [Nostr: Nostr Protocol Specification](https://github.com/nostr-protocol/nostr) — Cryptographic relay protocol for decentralized social messaging.
- [IPFS: Libp2p Specification](https://docs.libp2p.io/) — Modular network stack for peer-to-peer applications, underlying IPFS.

## 15. Conclusion

BlockParty is a social protocol built for a world where communication must be resilient, private, ephemeral, and voluntary. It is an antidote to the fragile, surveillance-driven platforms that dominate the current internet. By rooting trust in cryptography, scoping communication to audiences rather than networks, and allowing users to revoke or abandon identities at will, BlockParty redefines how digital presence can work.

Our goal is not simply to build technology—it is to build humane infrastructure: systems that empower, respect, and eventually forget. We invite developers, communities, and anyone who values freedom to extend, remix, and carry BlockParty forward.

Nothing can stop the signal. But now, with BlockParty, you can choose how long it echoes.

## 16. License & Contribution

BlockParty Protocol is dedicated to the public domain under [CC0](https://creativecommons.org/publicdomain/zero/1.0/).

You are free to:

- Fork, extend, and mirror the protocol
- Use or ignore it for your own purposes
- Contribute back, or not

> Make protocols, not platforms.
> Make speech safe again.
> Make permanence optional.

---

This whitepaper will evolve as the protocol matures. The canonical version lives at [bpprotocol.org](https://bpprotocol.org). Contributions welcome.

