---
Title: MIME Header Conventions for BlockParty Content Types
Version: 0.1.0
Last Updated: 2025-05-01
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/mime-header-conventions
---

# MIME Header Conventions for BlockParty Content Types

This document defines the conventions for MIME-style headers used in `content.post` and `content.chunked.manifest` block types. These headers are used to describe the semantic nature, encoding, and intended handling of content bodies.

---

## ✦ Header Format

Headers are expressed as a flat key-value map (string keys and string values). Keys are **case-insensitive**, but it is recommended that clients preserve the original casing for display and interoperability.

Header keys SHOULD use a **URI-style namespace format**, especially for extension-defined or non-standard fields. The convention is:

```
<namespace_domain>/<version>/<header-key>
```

For example:
```json
{
  "headers": {
    "bpprotocol.org/v1/content-type": "text/plain",
    "bpprotocol.org/v1/language": "en",
    "studio.example/v1/display-mode": "carousel"
  },
  "body": "Hello, world."
}
```

---

## ✦ Required Header Fields

| Header Key | Description |
|------------|-------------|
| `bpprotocol.org/v1/content-type` | The MIME type of the body. Examples: `text/plain`, `text/markdown`, `image/jpeg`, `application/json`. **Required**. |

---

## ✦ Common Optional Headers

| Header Key | Description |
|------------|-------------|
| `bpprotocol.org/v1/language` | ISO 639-1 language code (e.g. `en`, `fr`, `es`). Helps with multilingual rendering and fallback. |
| `bpprotocol.org/v1/encoding` | Encoding of the body if not UTF-8. e.g. `base64`, `gzip`. Use only when body is not a direct text or binary. |
| `bpprotocol.org/v1/summary` | A short (<=280 characters) plaintext summary of the post. Can be used for previewing or indexing. |
| `bpprotocol.org/v1/tags` | Comma-separated list of tags. These are informal and optional. Use semantic tagging extensions for rich tagging. |
| `bpprotocol.org/v1/visibility` | Optional hint for client rendering (e.g. `public`, `friends-only`, `ephemeral`). Non-enforced. |
| `bpprotocol.org/v1/title` | Short title for the content body. Can be used as a heading in preview UIs. |
| `bpprotocol.org/v1/created-at` | ISO 8601 timestamp indicating when the content was originally created (if different from block timestamp). |

---

## ✦ Conventions for `content.chunked.manifest`

The `headers` section of a `content.chunked.manifest` block behaves the same as `content.post`, and follows all the same conventions. Clients should honor headers before attempting to interpret raw chunk data.

```json
{
  "headers": {
    "bpprotocol.org/v1/content-type": "video/mp4",
    "bpprotocol.org/v1/language": "en",
    "bpprotocol.org/v1/title": "Highlights from the hike"
  },
  "chunks": ["chunk1", "chunk2"],
  "sha512": "..."
}
```

---

## ✦ Unknown Headers

Clients must safely ignore any unrecognized header keys. Applications and extensions SHOULD use their own domain-based namespaces (e.g. `studio.acme/v1/overlay-mode`) to avoid collisions.

---

## ✦ Notes on Internationalization

Clients may use the `bpprotocol.org/v1/language` header to prioritize content or fallback translations. Future extensions may define multilingual overlays or translation mappings.

---

This document is versioned. Future versions may introduce header registries, content negotiation mechanisms, or validation schemas for standard header formats.

