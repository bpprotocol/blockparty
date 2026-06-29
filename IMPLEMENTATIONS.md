# BlockParty Implementations

BlockParty has multiple reference implementations. They are kept honest — and interoperable — by reproducing the same [conformance vectors](./conformance/vectors.json) (see [`conformance/README.md`](./conformance/README.md) for the contract).

> **Layout:** each implementation lives under `implementations/<language>/` (decided in #21, consolidated in #24).

## Index

| Implementation | Language | Path | Package / module | Status | Conformance |
|----------------|----------|------|------------------|--------|-------------|
| Go reference | Go | [`implementations/go/`](./implementations/go) | `github.com/bpprotocol/blockparty/implementations/go` (CLI: `implementations/go/cmd/bp`) | Complete | ✅ generates & reproduces `conformance/vectors.json` |
| TypeScript reference | TypeScript / Node | [`implementations/typescript/`](./implementations/typescript) | `@blockparty/sdk` + `@blockparty/cli` | Complete | ✅ reproduces `conformance/vectors.json` |

Both cover the full protocol stack: crypto → derivations → identity → blocks → encryption → audiences → connections → block types, plus a filesystem-transport CLI.

## Per-implementation README convention

Each implementation's README should state:

- **Install / build / test** commands.
- The **conformance vectors** path (`conformance/vectors.json`) and how the implementation validates against it.
- **Status** and a mapping of its modules to the [protocol specs](./protocol/specs).
- For a publishable library: how external applications consume it.

## Adding a new implementation

1. Create a directory for it under `implementations/<language>/`.
2. Implement the protocol layers, mirroring an existing reference (Go or TypeScript) and the [specs](./protocol/specs). Generate wire types from [`protocol/proto/v1`](./protocol/proto/v1).
3. **Conform:** reproduce every value in [`conformance/vectors.json`](./conformance/vectors.json) from the documented fixed inputs (a deep-equal "conformance" test is the strongest form), and pass the cross-impl fixtures listed in [`conformance/README.md`](./conformance/README.md) (signed-block encoding, AEAD ciphertext both ways, burn key fingerprints, connection derivations).
4. Add a README following the convention above.
5. Add a CI workflow (mirror `.github/workflows/go.yml` / `typescript.yml`).
6. Add a row to the index in this file.

If a value cannot be reproduced, the divergence is a bug in one implementation or a spec ambiguity to resolve upstream in [`protocol/specs`](./protocol/specs).
