---
title: Client Extension Block Types
version: 0.1.0
updated: 2025-05-01
status: Draft
license: CC0
canonical: https://bpprotocol.org/specs/extensions/client
---

# Client Extension Block Types

This document defines a set of optional, non-binding block types used by clients to coordinate presence, signal capabilities, and enable peer-to-peer rendezvous. These block types are **audience-scoped**, **ephemeral**, and fully **extensible**.

---

## `bpprotocol.org/v1/types/client.rallypoint`

Announces a rendezvous point for direct or relayed peer-to-peer connections.

```json
{
  "multiaddr": "/ip4/192.168.0.42/tcp/7777",
  "relay": "/dns4/relay.example.com/tcp/443/p2p-circuit",
  "protocols": [
    "bpprotocol.org/v1/relay/rpc",
    "bpprotocol.org/v1/filesync"
  ],
  "start_at": 1746381600,
  "expires_at": 1746388800
}
```

### Fields:
- `multiaddr`: Required. The public or local multiaddr at which the client may be contacted.
- `relay`: Optional. A multiaddr for a relay or gateway that can route traffic to the client.
- `protocols`: Optional. List of supported post-handshake application-layer protocols.
- `start_at`: Optional. UNIX timestamp (UTC) when the rallypoint becomes valid.
- `expires_at`: Optional. UNIX timestamp (UTC) when the rallypoint should be ignored.

---

## `bpprotocol.org/v1/types/client.status`

Signals ephemeral presence or contextual status for a peer.

```json
{
  "emoji": "📚",
  "message": "reading specs",
  "expires_at": 1746385200
}
```

### Fields:
- `emoji`: Optional. A short emoji indicator.
- `message`: Optional. Freeform status message.
- `expires_at`: Optional. UNIX timestamp (UTC) expiration.

Used to display lightweight presence in UIs such as chat, dashboards, or world lists.

---

## `bpprotocol.org/v1/types/client.info`

Publishes non-identity runtime metadata about the client and supported features.

```json
{
  "client": "BlockParty Reference Node",
  "version": "1.0.0-alpha",
  "locale": "en-US",
  "platform": "desktop",
  "supports": [
    "bpprotocol.org/v1/types/content.gallery",
    "bpprotocol.org/v1/types/rpc.render"
  ],
  "expires_at": 1746392400
}
```

### Fields:
- `client`: Optional. The name of the client software.
- `version`: Optional. The current version string.
- `locale`: Optional. IETF language tag.
- `platform`: Optional. Freeform string, e.g. "mobile", "web", "desktop".
- `supports`: Optional. List of known-supported block types or protocols.
- `expires_at`: Optional. Expiration as UNIX timestamp (UTC).

This block is not an identity declaration. It exists to assist in adaptive UX or peer coordination.

---

## Usage Notes

- All blocks are **audience-scoped** and may be safely mirrored without leaking sensitive data.
- All timestamps use **UNIX epoch format** in **UTC**.
- Clients may choose to ignore these blocks, cache them briefly, or use them to inform UX.

These blocks are optional and intended as working examples of how the protocol can be extended gracefully by client implementations.

