# BlockParty JS

Node.js / TypeScript reference implementation of the [BlockParty Protocol](../protocol). Tracking epic: **#10**.

This is a [pnpm](https://pnpm.io) workspace with a deliberate package boundary so the protocol logic is a **reusable, separately-publishable library**, not buried inside the CLI:

```
js/
├── packages/
│   ├── sdk/   → @blockparty/sdk   (reusable library — all protocol logic, published)
│   └── cli/   → @blockparty/cli   (thin reference node/CLI — depends on @blockparty/sdk)
```

- **`@blockparty/sdk`** holds every protocol primitive (crypto, derivations, identity, blocks, encryption, audiences, connections, block types). It is framework-agnostic and fully typed.
- **`@blockparty/cli`** is a thin consumer — argument parsing, transport, and wiring. It depends on the SDK via `workspace:*`.

External implementers reuse the library directly:

```sh
pnpm add @blockparty/sdk
```

```ts
import { PROTOCOL_VERSION } from "@blockparty/sdk";
```

## Develop

```sh
cd js
pnpm install
pnpm -r build     # build every package (sdk before cli)
pnpm -r test      # run unit tests (Vitest)
pnpm -r lint      # ESLint
pnpm -r typecheck # tsc --noEmit
```

## Status

Scaffold only (#11). The protocol modules land in #12–#18, each mirroring the
Go reference in [`../sdk`](../sdk) and validated against the shared conformance
vectors in [`../sdk/vectors/vectors.json`](../sdk/vectors/vectors.json).

Cryptography target (resolved in #20): **ML-DSA-65** (FIPS 204) for signatures
and **ML-KEM768** (FIPS 203) for the KEM.
