import { defineNuxtConfig } from 'nuxt/config'

// Nuxt 3 as a static single-page app (no SSR) rendered inside the Electron
// shell. `nuxt generate` emits a static bundle to .output/public that the
// Electron main process loads via loadFile in production.
export default defineNuxtConfig({
  ssr: false,
  // Global design tokens + UI primitives (Friendkit-inspired social styling).
  css: ['~/assets/css/main.css'],
  devtools: { enabled: false },
  // Work around a Nuxt 3.21 SPA-dev regression: with `ssr: false` the
  // vite-node-server plugin resolves the *client* server as if it were the SSR
  // one and throws "No entry found in rollupOptions.input", so `nuxt dev` never
  // starts. Enabling the Vite Environment API gives the dev server a real `ssr`
  // environment with the expected entry, which takes the same code path cleanly.
  experimental: { viteEnvironmentApi: true },
  // Assets are referenced from the root: in production the Electron shell
  // serves .output/public from a custom `app://bundle` scheme (see
  // electron/main.ts), so absolute paths resolve against that origin. Loading
  // the same bundle over file:// would not work — Nuxt's entry is an ES module,
  // which browsers refuse to fetch cross-origin from a file URL.
  app: {
    head: {
      title: 'BlockParty',
    },
  },
  // No `nitro.preset` override needed: with `ssr: false`, `nuxt generate`
  // already emits the static .output/public bundle the Electron shell loads.
  compatibilityDate: '2024-11-01',
})
