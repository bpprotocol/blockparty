<script setup lang="ts">
import { onMounted, ref } from 'vue'

// The renderer reaches privileged APIs only through the preload bridge
// (window.bpDesktop) — never Node directly. The node API client lands in #41.
const versions = ref<{ electron: string; chrome: string; node: string } | null>(null)
const bridgeAvailable = ref(false)

onMounted(() => {
  if (typeof window !== 'undefined' && window.bpDesktop) {
    bridgeAvailable.value = true
    versions.value = window.bpDesktop.versions()
  }
})
</script>

<template>
  <main class="app">
    <h1>BlockParty</h1>
    <p class="tagline">Desktop client</p>
    <p v-if="versions" class="versions">
      Electron {{ versions.electron }} · Chromium {{ versions.chrome }} · Node {{ versions.node }}
    </p>
    <p v-else class="versions">Preload bridge not detected (running outside Electron).</p>
    <p class="status">Renderer is up. Node API client, onboarding, and feed land in #41–#44.</p>
    <p class="security">
      contextIsolation ✓ · nodeIntegration ✗ · sandbox ✓ — bridge present: {{ bridgeAvailable }}
    </p>
  </main>
</template>

<style>
:root {
  color-scheme: light dark;
  font-family: system-ui, sans-serif;
}
body {
  margin: 0;
}
.app {
  max-width: 40rem;
  margin: 0 auto;
  padding: 3rem 1.5rem;
}
h1 {
  margin-bottom: 0.25rem;
}
.tagline {
  margin-top: 0;
  opacity: 0.7;
}
.versions {
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
}
.security {
  font-family: ui-monospace, monospace;
  font-size: 0.85rem;
  opacity: 0.6;
}
</style>
