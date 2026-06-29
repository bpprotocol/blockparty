import { defineNuxtConfig } from 'nuxt/config'

// Nuxt 3 as a static single-page app (no SSR) rendered inside the Electron
// shell. `nuxt generate` emits a static bundle to .output/public that the
// Electron main process loads via loadFile in production.
export default defineNuxtConfig({
  ssr: false,
  devtools: { enabled: false },
  // Relative base so the generated index.html loads its assets via file://.
  app: {
    baseURL: './',
    head: {
      title: 'BlockParty',
    },
  },
  // The renderer talks only to the local node over the preload bridge; no
  // external data fetching at build time.
  nitro: {
    preset: 'static',
  },
  compatibilityDate: '2024-11-01',
})
