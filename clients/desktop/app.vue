<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRuntimeConfig } from '#imports'
import type { BootstrapRequest } from './electron/bridge'
import { useNode } from './composables/useNode'
import OnboardingView from './components/OnboardingView.vue'
import NodeDashboard from './components/NodeDashboard.vue'
import ComposeView from './components/ComposeView.vue'
import FeedView from './components/FeedView.vue'
import ConnectionsView from './components/ConnectionsView.vue'
import IdentityView from './components/IdentityView.vue'

const { status, lifecycle, error, busy, view, refresh, bootstrap } = useNode()
const bootstrapError = ref<string | null>(null)
let timer: ReturnType<typeof setInterval> | undefined

// Background image lives in public/. Resolve it through the app's baseURL so the
// URL is correct both under `nuxt dev` (served at /) and in the packaged SPA,
// which loads over file:// with a relative baseURL ('./').
const baseURL = useRuntimeConfig().app.baseURL
const authBgStyle = computed(() => ({
  backgroundImage: `url("${baseURL}bg.png")`,
}))

async function onBootstrap(req: BootstrapRequest): Promise<void> {
  bootstrapError.value = null
  const r = await bootstrap(req)
  if (!r.ok) bootstrapError.value = r.error ?? 'Bootstrap failed.'
}

onMounted(() => {
  void refresh()
  // Poll while connecting/onboarding so the view advances once the node is ready
  // or a World is bootstrapped; the dashboard stays fresh too.
  timer = setInterval(() => void refresh(), 3000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <!-- Dashboard: shown once the node is ready and a World is loaded. -->
  <main v-if="view === 'ready' && status" class="app">
    <header>
      <h1>BlockParty</h1>
      <span class="conn up">Connected</span>
    </header>

    <NodeDashboard :status="status" />
    <ComposeView v-if="status.canAuthor" />
    <ConnectionsView v-if="status.canAuthor" />
    <IdentityView v-if="status.canAuthor" />
    <FeedView />

    <footer v-if="lifecycle" class="lifecycle">
      node process: {{ lifecycle.mode }} · {{ lifecycle.state }}
      <span v-if="lifecycle.restarts > 0">· {{ lifecycle.restarts }} restart(s)</span>
      · {{ lifecycle.endpoint }}
    </footer>
  </main>

  <!-- Login / setup: a centered card over the background image, shown until the
       dashboard is ready (connecting, onboarding, or disconnected states). -->
  <div v-else class="auth" :style="authBgStyle">
    <div class="auth-card">
      <h1 class="brand">BlockParty</h1>

      <OnboardingView v-if="view === 'onboarding'" :busy="busy" @submit="onBootstrap" />

      <section v-else class="status">
        <p v-if="view === 'no-bridge'">Running outside Electron — no node bridge.</p>
        <p v-else class="connecting">Connecting to the local node…</p>
        <p v-if="error" class="err">{{ error }}</p>
        <button v-if="view !== 'no-bridge'" class="primary" @click="refresh">Retry</button>
      </section>

      <p v-if="bootstrapError" class="err">{{ bootstrapError }}</p>

      <p v-if="lifecycle" class="lifecycle">
        node: {{ lifecycle.mode }} · {{ lifecycle.state }} · {{ lifecycle.endpoint }}
      </p>
    </div>
  </div>
</template>

<style>
:root {
  color-scheme: light dark;
  font-family: system-ui, sans-serif;
}
body {
  margin: 0;
}

/* Dashboard */
.app {
  max-width: 40rem;
  margin: 0 auto;
  padding: 2.5rem 1.5rem;
}
header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}
.conn {
  font-size: 0.8rem;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
}
.conn.up {
  background: #1a7f37;
  color: white;
}
.err {
  color: #cf222e;
  font-family: ui-monospace, monospace;
  font-size: 0.9rem;
}
.lifecycle {
  margin-top: 2rem;
  font-family: ui-monospace, monospace;
  font-size: 0.8rem;
  opacity: 0.55;
}

/* Login / setup screen */
.auth {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  box-sizing: border-box;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}
/* Scrim for legibility over the photo. */
.auth::before {
  content: '';
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
}
.auth-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 30rem;
  max-height: calc(100vh - 4rem);
  overflow-y: auto;
  box-sizing: border-box;
  padding: 2rem 1.75rem;
  border-radius: 14px;
  background: color-mix(in srgb, Canvas 90%, transparent);
  color: CanvasText;
  border: 1px solid color-mix(in srgb, CanvasText 12%, transparent);
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(6px);
}
.auth-card .brand {
  margin: 0 0 1.25rem;
  font-size: 1.6rem;
  letter-spacing: 0.02em;
}
.auth-card .status {
  display: grid;
  gap: 0.75rem;
  justify-items: start;
}
.auth-card .connecting {
  opacity: 0.8;
}
.auth-card .lifecycle {
  margin-top: 1.5rem;
}
.auth-card .primary {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  background: #1a7f37;
  color: white;
  cursor: pointer;
}
</style>
