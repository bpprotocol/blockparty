# @blockparty/sdk

Reusable TypeScript library for the [BlockParty Protocol](https://github.com/bpprotocol/blockparty) — post-quantum crypto, deterministic derivations, identity, blocks, audience-scoped encryption, connections, and block types.

It is one of two reference implementations (the other is in Go); both are validated against the same [conformance vectors](https://github.com/bpprotocol/blockparty/blob/master/sdk/vectors/vectors.json), so they interoperate byte-for-byte.

- **Post-quantum:** ML-DSA-65 (FIPS 204) signatures, ML-KEM768 (FIPS 203) KEM, via [@noble](https://github.com/paulmillr).
- **Runtime-agnostic:** no Node-only APIs in the core (the filesystem transport lives in the CLI).

## Install

```sh
pnpm add @blockparty/sdk
```

## Usage

```ts
import {
  openWorld,
  openIdentity,
  publicAudience,
  getTypeCode,
  newBlock,
  signBlock,
  verifyBlock,
  encryptForBlock,
  decryptData,
  create,
  toBinary,
  ContentPostSchema,
  TYPE_CONTENT_POST,
  utf8ToBytes,
} from "@blockparty/sdk";

// Open a World and derive an identity from a passphrase.
const world = openWorld("festival-2026");
const alice = openIdentity(world, "correct horse battery staple");

// Encrypt a post to a public audience, then sign it.
const pub = publicAudience(world, 1);
const typeCode = getTypeCode(world, TYPE_CONTENT_POST);
const payload = toBinary(
  ContentPostSchema,
  create(ContentPostSchema, { body: utf8ToBytes("hello") }),
);
const data = encryptForBlock(
  pub.secret!,
  1,
  typeCode,
  pub.code,
  Math.floor(Date.now() / 1000),
  payload,
);

const block = newBlock(alice.address, typeCode, pub.code, Math.floor(Date.now() / 1000), data);
signBlock(block, world, alice.mldsa);

// A recipient who knows the World verifies and decrypts.
verifyBlock(block, world, alice.mldsa.publicKey); // true
const plaintext = decryptData(pub.secret!, block);
```

See the [connections](https://github.com/bpprotocol/blockparty/blob/master/protocol/specs/connections.md) and other specs for the private handshake, rotation, burn, and RPC flows.

## License

CC0-1.0 (public domain).
