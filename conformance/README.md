# BlockParty Conformance Vectors

`vectors.json` is the **cross-client conformance oracle** for the BlockParty Protocol: deterministic outputs of every core derivation from a fixed set of inputs. It is owned by the protocol, not by any one implementation — each reference implementation reproduces it exactly, which is what proves they interoperate byte-for-byte.

All byte values are lowercase hex; large keys are fingerprinted with Keccak-256.

## Fixture sections

| Section | What it locks |
|---------|---------------|
| `crypto` | ML-KEM768 and ML-DSA-65 public keys derived from a seed (fingerprinted). |
| `world` | A World's wallet/type/audience salts, signing-key fingerprint, and a type/audience code. |
| `identity` | An identity's address and key fingerprints (passphrase → keys, world-scoped). |
| `blockID` | A content-binding block ID. |
| `audiences` | A public-audience secret and an inbox audience code. |
| `aead` | The content key, AAD, and a deterministic XChaCha20-Poly1305 ciphertext (`nonce ‖ ct`). |
| `block` | A fully-signed block's ID and the Keccak-256 of its canonical Protobuf encoding. |
| `connection` | The epoch-0 connection secret and the audience codes for epochs 0 and 1 (after a rotation), from fixed KEM secrets/nonces. |
| `burn` | An `identity.burn`: the address and the fingerprints of the revealed private keys. |

The fixed inputs (world seed, passphrase, key seed, timestamps, etc.) are constants in the generator (see below) and are echoed in the file where relevant (e.g. `crypto.seed`, `block.passphrase`).

## Generation

The Go reference implementation is the **generator**: [`sdk/vectors`](../sdk/vectors) computes the full set from the fixed inputs and writes this file.

```sh
cd sdk
go test ./vectors -update-vectors   # regenerate ../conformance/vectors.json
go test ./vectors                   # verify the committed file is up to date
```

Any other implementation reproduces the same values from the same fixed inputs and diffs against this file — see "how to conform" below.

## How to conform

A conforming implementation MUST:

1. **Reproduce every value in `vectors.json`** from the documented fixed inputs (a single "conformance" test that re-derives the whole structure and deep-equals this file is the strongest form).
2. **Interoperate on the cross-impl fixtures**, not just the scalar derivations:
   - `block.encodedKeccak` — build and sign the same block and reproduce its canonical encoding byte-for-byte (requires deterministic ML-DSA signing).
   - `aead.data` — reproduce the ciphertext (Node → Go) **and** decrypt it back to `aead.plaintext` (Go → Node).
   - `burn.{mldsaSkKeccak,kyberSkKeccak}` — the revealed private keys must be byte-identical, and `verifyBurn` must re-derive the address.
   - `connection.*` — the per-epoch derivations must match, and a live two-party handshake must agree.

If a value cannot be reproduced, the divergence is either a bug in one implementation or a genuine spec ambiguity to resolve in [`protocol/specs`](../protocol/specs).

See [`IMPLEMENTATIONS.md`](../IMPLEMENTATIONS.md) for the list of implementations and the checklist for adding a new one.
