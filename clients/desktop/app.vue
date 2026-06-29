<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import type { BootstrapRequest } from './electron/bridge'
import { useNode } from './composables/useNode'
import OnboardingView from './components/OnboardingView.vue'
import NodeDashboard from './components/NodeDashboard.vue'

const { status, lifecycle, error, busy, view, refresh, bootstrap } = useNode()
const bootstrapError = ref<string | null>(null)
let timer: ReturnType<typeof setInterval> | undefined

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
  <main class="app">
    <header>
      <h1>BlockParty</h1>
      <span class="conn" :class="view === 'ready' || view === 'onboarding' ? 'up' : 'down'">
        {{ view === 'ready' || view === 'onboarding' ? 'Connected' : 'Disconnected' }}
      </span>
    </header>

    <NodeDashboard v-if="view === 'ready' && status" :status="status" />

    <OnboardingView v-else-if="view === 'onboarding'" :busy="busy" @submit="onBootstrap" />

    <section v-else class="disconnected">
      <p v-if="view === 'no-bridge'">Running outside Electron — no node bridge.</p>
      <p v-else>Connecting to the local node…</p>
      <p v-if="error" class="err">{{ error }}</p>
      <button v-if="view !== 'no-bridge'" @click="refresh">Retry</button>
    </section>

    <p v-if="bootstrapError" class="err">{{ bootstrapError }}</p>

    <footer v-if="lifecycle" class="lifecycle">
      node process: {{ lifecycle.mode }} · {{ lifecycle.state }}
      <span v-if="lifecycle.restarts > 0">· {{ lifecycle.restarts }} restart(s)</span>
      · {{ lifecycle.endpoint }}
    </footer>
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
.conn.down {
  background: #cf222e;
  color: white;
}
.err {
  color: #cf222e;
  font-family: ui-monospace, monospace;
  font-size: 0.9rem;
}
button {
  margin-top: 0.5rem;
  padding: 0.4rem 0.9rem;
}
.lifecycle {
  margin-top: 2rem;
  font-family: ui-monospace, monospace;
  font-size: 0.8rem;
  opacity: 0.55;
}
</style>
