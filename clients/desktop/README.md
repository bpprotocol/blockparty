# BlockParty desktop client

Cross-platform desktop client for BlockParty: an **Electron** shell wrapping a
**Nuxt 3 SPA** renderer. It is a thin GUI over the local [node server](../../node)
— it does not speak libp2p or manage storage itself; it talks to the node's API.
Tracking epic: **[#26](https://github.com/bpprotocol/blockparty/issues/26)**.

> **Status:** scaffold ([#40](https://github.com/bpprotocol/blockparty/issues/40)). Secure Electron shell, Nuxt SPA renderer, and the dev/build pipeline are in place. The node API client (#41), node lifecycle (#42), onboarding (#43), feed (#44), and the rest follow.

## Architecture

- **`electron/main.ts`** — the Electron main process: window lifecycle and (later) managing the bundled `bpnode`. It loads the Nuxt dev server in development and the generated static SPA in production.
- **`electron/preload.ts`** — the **only** bridge between renderer and main. It exposes a small, typed `window.bpDesktop` API via `contextBridge`; the renderer never touches Node, `fs`, or the main process directly.
- **`app.vue` / Nuxt** — the renderer (SPA, `ssr: false`).

### Secure defaults (#40)

The `BrowserWindow` is created with `contextIsolation: true`, `nodeIntegration: false`, `sandbox: true`, and `webSecurity: true`; external links open in the user's browser, not in-app. All privileged capability is added behind the typed preload bridge.

## Develop

```sh
pnpm install
pnpm dev            # nuxt dev server + electron window (requires a display)
```

```sh
pnpm build:renderer # nuxt generate → .output/public (static SPA)
pnpm build:main     # tsc → dist-electron (main + preload, CommonJS)
pnpm build          # both, then electron-builder → release/ (mac/win/linux)
pnpm typecheck      # vue-tsc (renderer) + tsc (electron)
pnpm lint           # eslint (flat config: js + ts + vue)
pnpm format         # prettier
```

## Layout

```
clients/desktop/
├── electron/          # main + preload (compiled by tsc → dist-electron/)
├── app.vue            # renderer root (Nuxt SPA)
├── types/window.d.ts  # renderer-side type for the preload bridge
├── nuxt.config.ts     # ssr: false static SPA
├── electron-builder.yml
└── eslint.config.mjs
```
