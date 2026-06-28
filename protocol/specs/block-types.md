---
Title: Block Types and Type Identifiers
Version: 0.1.0
Last Updated: 2026-06-28
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/block-types
---

# Block Types and Type Identifiers

> This document defines the canonical block types used by the BlockParty Protocol, including their structure, semantics, and usage guidance for implementers.

BlockParty uses a set of core block types to represent every action or piece of data exchanged over the protocol. These types are defined by **canonical domain-scoped identifiers** (e.g. `bpprotocol.org/v1/types/content.post`), which are hashed per-World to generate a type code stored in block metadata.

This document lists the canonical block types defined under the `bpprotocol.org/v1/types/` namespace, along with schema examples for reference.

---

## 📦 Content & Expression

### `content.post`  
**Type:** `bpprotocol.org/v1/types/content.post`

A general-purpose post. Includes MIME-style headers and body content.

```json
{
  "headers": {
    "bpprotocol.org/v1/content-type": "text/plain",
    "bpprotocol.org/v1/language": "en"
  },
  "body": "Hello, world."
}
```

### `content.reaction`
**Type:** `bpprotocol.org/v1/types/content.reaction`

A lightweight reaction to another block.
```json
{
  "target": "<block_id>",
  "reaction": "👍"
}
```

### `content.tag.request`
**Type:** `bpprotocol.org/v1/types/content.tag.request`

A request to tag another identity in a block.
```json
{
  "target": "<block_id>",
  "identity": "<identity_address>",
  "label": "mention"
}
```

### `content.tag.accept`
**Type:** `bpprotocol.org/v1/types/content.tag.accept`

Acknowledgement of a tag request.
```json
{
  "request": "<tag_request_block_id>"
}
```

### `content.chunked.manifest`
**Type:** `bpprotocol.org/v1/types/content.chunked.manifest`

A manifest for progressively loadable streamable content (application layer; see §"Chunked Data"). `chunks` is an ordered list of `content.chunked.block` block IDs.
```json
{
  "headers": {
    "bpprotocol.org/v1/content-type": "video/mp4"
  },
  "chunks": ["<chunk_block_id_1>", "<chunk_block_id_2>"],
  "sha512": "<digest of reassembled bytes>"
}
```

### `content.chunked.block`
**Type:** `bpprotocol.org/v1/types/content.chunked.block`

A single streamable content chunk.
```json
{
  "data": "<binary_data>"
}
```

### `content.gallery`  
**Type:** `bpprotocol.org/v1/types/content.gallery`

Groups a list of existing blocks into an ordered visual or thematic collection. Often used for photo galleries, curated series, or multi-block storytelling.

```json
{
  "title": "Spring Hike 2025",
  "description": "A collection of photos from our day in the hills.",
  "blocks": [
    "block-id-1",
    "block-id-2",
    "block-id-3"
  ]
}
```

- `title` *(optional)*: A human-readable gallery title
- `description` *(optional)*: Describes the gallery context or intent
- `blocks`: An ordered array of block IDs to display

**Client guidance:** Render blocks sequentially as a gallery. Degrade gracefully if referenced blocks are unavailable or encrypted.

### `content.quote`  
**Type:** `bpprotocol.org/v1/types/content.quote`

Creates a quoted reference to another block, optionally accompanied by commentary. Enables resharing, highlighting, or contextual dialogue around content while respecting audience boundaries.

```json
{
  "quote": "block-id-original",
  "commentary": {
    "headers": {
      "Content-Type": "text/plain"
    },
    "body": "This hits hard today."
  }
}
```

- `quote`: The ID of the block being quoted
- `commentary` *(optional)*: A content.post-like structure (headers/body) offering thoughts, context, or critique

**Client behavior:**  
- Attempt to display the quoted block if it's accessible to the current user  
- If not accessible, display a placeholder message (e.g. *"You do not have access to this content."*)  
- Always render the commentary if present

**Use Cases:**  
- Social resharing with context
- Quoting a private post without disclosing its content
- Annotating or critiquing content with full attribution

**Privacy Note:**  
The quoted block is not duplicated or mirrored unless already visible. This block acts as a contextual reference, not a copy.

---

## 🤝 Connections & Relationships

These blocks form the post-quantum connection handshake. The full protocol — key agreement, private-audience derivation, replay protection, and rotation — is specified in [Connections](./connections.md); the schemas below are the on-the-wire payloads.

### `connect.request`
**Type:** `bpprotocol.org/v1/types/connect.request`

Initiates a private connection. Sent to the target's inbox audience. `init_kem_pub` is the initiator's ephemeral ML-KEM768 public key; `kem_ciphertext` is encapsulated to the target's static `kyberKey.pub`.
```json
{
  "target": "<identity_address>",
  "init_kem_pub": "<base64 ephemeral ML-KEM768 public key>",
  "kem_ciphertext": "<base64 ML-KEM768 ciphertext>",
  "nonce": "<base64 random(32)>"
}
```

### `connect.response`
**Type:** `bpprotocol.org/v1/types/connect.response`

Completes the handshake. `kem_ciphertext` is encapsulated to the initiator's `init_kem_pub`; both parties then derive the shared connection secret and private audience.
```json
{
  "request": "<connect_request_block_id>",
  "kem_ciphertext": "<base64 ML-KEM768 ciphertext>",
  "nonce_response": "<base64 random(32)>"
}
```

### `connect.identity`
**Type:** `bpprotocol.org/v1/types/connect.identity`

Reveals private metadata, **encrypted to the newly derived private audience**. `keys` are any additional public keys shared within the connection.
```json
{
  "name": "Alice",
  "keys": ["<additional_pubkey>", "..."]
}
```

### `connect.rotate`
**Type:** `bpprotocol.org/v1/types/connect.rotate`

Advances the connection to a new epoch with fresh KEM entropy (forward ratchet). `new_channel` is the hex audience code of the next epoch; `kem_ciphertext` is the fresh contribution. See [Connections](./connections.md) §"Rotation".
```json
{
  "new_channel": "<hex audience_code of next epoch>",
  "kem_ciphertext": "<base64 ML-KEM768 ciphertext>"
}
```

### `connect.close`
**Type:** `bpprotocol.org/v1/types/connect.close`

Notifies the peer of relationship closure. `reason` ∈ `rotated_keys` | `closed` | `compromised` | `expired` | `other`.
```json
{
  "target": "<identity_address>",
  "reason": "rotated_keys"
}
```

---

## 🧬 Identity & Lifecycle

### `identity`
**Type:** `bpprotocol.org/v1/types/identity`

Declares an identity and its public keys. Normally a [plaintext block](./encryption.md) on a [public audience](./audiences.md). `public_keys` are the identity's `dilithiumKey.pub` and `kyberKey.pub` (the material peers need to verify authorship and open [connections](./connections.md)).
```json
{
  "name": "Alice",
  "public_keys": ["<dilithium_pub>", "<kyber_pub>"]
}
```

### `identity.burn`
**Type:** `bpprotocol.org/v1/types/identity.burn`

Burns the identity by **revealing its root private keys**, entering the post-truth state. Verification and client behavior are specified in the [Identity Burn RFC](./identity-burn-rfc.md) §5. Normally a plaintext block on a public audience so it is world-readable.
```json
{
  "identity": "<identity_address>",
  "revealed_dilithium": "<base64 dilithiumKey.private>",
  "revealed_kyber": "<base64 kyberKey.private>",
  "burn_notice": "voluntary"
}
```
- `burn_notice` ∈ `voluntary` | `compromised` | `rotated` | `other`.

---

## 🔗 Chunked Data

BlockParty defines **two** chunking systems with distinct roles — choose by layer:

| System | Purpose | Carries MIME headers? |
|--------|---------|-----------------------|
| `chunk.*` | **Transport/storage layer** — fragment any serialized block or blob that is too large for a single transport unit (QR frame, BLE MTU, etc.), to be reassembled verbatim. Opaque bytes. | No |
| `content.chunked.*` | **Application layer** — a *streamable content item* (large video, audio) with MIME headers, progressively loadable as a post. | Yes (manifest) |

### Reassembly (both systems)

In both, the manifest's `chunks` array is an **ordered list of the block IDs** of the corresponding chunk blocks. To reconstruct:

1. Resolve each block ID in `chunks` order.
2. Concatenate each chunk block's `data` payload in that order.
3. Verify `SHA-512` of the concatenated bytes equals the manifest's `sha512`; reject on mismatch.

Per-chunk integrity is provided by the chunk block IDs themselves (content-binding, per [Derivations](./derivations.md) `GetBlockID`). A missing chunk leaves the content incomplete; clients may retry retrieval or surface a partial/unavailable state.

### `chunk.manifest`
**Type:** `bpprotocol.org/v1/types/chunk.manifest`

Describes a reassemblable blob (transport/storage layer).
```json
{
  "chunks": ["<chunk_block_id_1>", "<chunk_block_id_2>"],
  "sha512": "<digest of reassembled bytes>"
}
```

### `chunk.block`
**Type:** `bpprotocol.org/v1/types/chunk.block`

Contains one fragment of binary payload data.
```json
{
  "data": "<binary_data>"
}
```
---

## 🔌 RPC Interaction

### `rpc.request`  
**Type:** `bpprotocol.org/v1/types/rpc.request`

Represents a structured request for an RPC-style operation.  
Method names are expressed as **canonical domain-scoped identifiers** (e.g. `bpprotocol.org/v1/rpc/core.ping`) to ensure global uniqueness and client-specific handling.

```json
{
  "id": "rpc-1234",
  "method": "bpprotocol.org/v1/rpc/core.ping",
  "params": {
    "timestamp": 1714567890
  }
}
```

**Client behavior:**  
- If the RPC method is **unsupported**, the client may safely ignore the request.  
- Execution of the RPC may be:
  - **User-triggered** (e.g. clicking a CTA rendered from the block), or
  - **Automatic**, based on the client’s configuration or logic.  
- No global enforcement exists — clients are expected to handle supported methods gracefully and ignore those they do not recognize.


### `rpc.response`
**Type:** `bpprotocol.org/v1/types/rpc.response`

A response to a previous `rpc.request`, using the same ID for correlation.
```json
{
  "id": "rpc-1234",
  "result": {
    "pong": true,
    "latency": 42
  },
  "error": null
}
```

### `rpc.render`  
**Type:** `bpprotocol.org/v1/types/rpc.render`

A dynamic, **local-only** render instruction. It reuses the block *shape* for uniformity but is never an on-the-wire block: a client constructs it internally (or derives it from local state) and, if it recognizes the `method`, generates a `content.post`-compatible structure and renders it **as if it were a post**. No response block is created — this is a one-way, ephemeral local render.

> ⚠️ A `rpc.render` is **never serialized, signed, encrypted, or transmitted**. It carries no `world_sig`/`author_sig` because it never leaves the client. If a client receives something claiming to be a `rpc.render` over a transport, it MUST discard it. To send instructions to a peer, use [`rpc.request`](#rpc-request) instead.

```json
{
  "id": "render-001",
  "method": "bpprotocol.org/v1/methods/render.time.greeting",
  "params": {
    "timezone": "America/New_York"
  }
}
```

- `id` *(optional)*: Correlation or invocation ID  
- `method`: A **canonical domain-scoped identifier** specifying the rendering logic. There is **no global method registry** — render methods are defined by whoever implements them (under their own namespace) and clients support whichever they choose; unknown methods are ignored.
- `params`: Arbitrary key-value data passed to the render handler

**Client behavior:**  
- If the method is **unsupported**, the client may safely ignore the block.  
- Rendering may occur:
  - **Automatically**, based on client logic or heuristics  
  - **User-triggered**, such as when a user interacts with a visible UI element derived from the block  
- If supported, the output must conform to the `content.post` schema and be rendered locally.  
- This block is **never published, signed, or shared** — it exists purely for ephemeral local transformation.

---

## 🔧 Usage & Resolution

Each block type is identified by a **canonical domain-scoped string** (e.g. `bpprotocol.org/v1/types/content.post`). These identifiers are hashed using a deterministic function scoped to a World’s type salt to produce the `type code` stored in the block’s metadata.

This approach enables:

- **World-specific semantics** — the same identifier may resolve differently in different Worlds
- **Extensibility** — third parties can define their own types using their own domain roots (e.g. `example.org/v1/types/custom.embed`)
- **Stable interop** — identifiers are portable and unambiguous across clients, even when block types are unknown or unsupported

**Client behavior:**  
Clients may cache, index, or label known type identifiers to enhance rendering, filtering, or interaction.  
If a block type is **unknown or unsupported**, clients are free to ignore it or present a fallback representation.

This document is versioned and will evolve alongside the protocol. Contributions welcome.

