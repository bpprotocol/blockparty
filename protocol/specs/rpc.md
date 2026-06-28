---
Title: Core RPC Methods
Version: 0.1.0
Last Updated: 2026-06-28
Status: Draft
License: CC0
Canonical URL: https://bpprotocol.org/specs/rpc
---

# Core RPC Methods

BlockParty supports delay-tolerant, signed RPC-style communication using rallypoints and negotiated protocols. This document defines core-level RPC methods that clients may support by default, beginning with `core.ping`.

These methods are intended to be minimal, illustrative, and non-prescriptive. Implementations may support them to improve inter-client coordination and diagnostics, but they are not required for protocol compliance.

---

## 📡 `bpprotocol.org/v1/rpc/core.ping`

A round-trip health check and latency probe.

### 🔁 Purpose
- Determine if a peer is reachable and responsive
- Calculate round-trip latency
- Optionally compare system clocks

### 📤 Request Payload
```json
{
  "timestamp": 1746396000
}
```
- `timestamp`: UNIX epoch (UTC) of when the ping was sent

### 📥 Response Payload
```json
{
  "timestamp": 1746396000,
  "received_at": 1746396002,
  "server_time": 1746396003,
  "node": "BlockParty Reference Node 1.0.0"
}
```
- `timestamp`: Echoed original timestamp from request
- `received_at`: Time the ping was received (set by responder)
- `server_time`: Current time on responder’s clock (optional)
- `node`: Optional string describing the software responding

### 💡 Behavior Notes
- Useful for latency and drift calculation
- No authentication is required for this method
- It may be implemented by all nodes regardless of purpose

### Example Calculation
A client sends a ping at `1746396000` and receives a response at `1746396005`:
- Round-trip latency ≈ 5 seconds
- Clock skew ≈ `server_time - (timestamp + latency/2)`

---

## 📘 Future Notes
- Other core methods may be defined later but should be minimal and optional
- Complex APIs and streaming protocols should be negotiated via rallypoints using domain-specific `protocol` strings
- **`rpc.request` vs `rpc.render`:** an [`rpc.request`](./block-types.md#-rpc-interaction) is a real signed block sent to a peer (which may answer with `rpc.response`); an [`rpc.render`](./block-types.md) is a **local-only** instruction a client renders into a `content.post`-shaped view and never transmits. Render methods have no global registry — clients support whichever they implement.

> "Ping is not a feature. It's a signal that connection is still possible."

This document may be expanded as usage patterns and client capabilities evolve.