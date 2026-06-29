<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import type { LifecycleState, NodeStatus } from './electron/bridge'

const status = ref<NodeStatus | null>(null)
const lifecycle = ref<LifecycleState | null>(null)
const error = ref<string | null>(null)
const connected = ref(false)
const checking = ref(false)
let timer: ReturnType<typeof setInterval> | undefined

async function refresh(): Promise<void> {
  const bridge = typeof window !== 'undefined' ? window.bpDesktop : undefined
  if (!bridge) {
    connected.value = false
    error.value = 'Preload bridge unavailable (running outside Electron)'
    return
  }
  lifecycle.value = await bridge.lifecycle.getState()

  checking.value = true
  const r = await bridge.node.getStatus()
  checking.value = false
  if (r.ok) {
    status.value = r.value
    connected.value = true
    error.value = null
  } else {
    connected.value = false
    error.value = r.error
  }
}

onMounted(() => {
  void refresh()
  timer = setInterval(() => void refresh(), 3000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function short(s: string): string {
  return s ? `${s.slice(0, 12)}…` : '—'
}
</script>

<template>
  <main class="app">
    <header>
      <h1>BlockParty</h1>
      <span class="conn" :class="connected ? 'up' : 'down'">
        {{ connected ? 'Connected' : 'Disconnected' }}
      </span>
    </header>

    <section v-if="connected && status" class="status">
      <dl>
        <dt>Node</dt>
        <dd>v{{ status.version }} · {{ status.mode }} mode</dd>
        <dt>World</dt>
        <dd>{{ status.worldLoaded ? short(status.world) : 'none loaded' }}</dd>
        <dt>Identity</dt>
        <dd>{{ status.identity ? short(status.identity) : '—' }}</dd>
        <dt>Blocks</dt>
        <dd>{{ status.blockCount }}</dd>
        <dt>Can author</dt>
        <dd>{{ status.canAuthor ? 'yes' : 'no' }}</dd>
      </dl>
      <p v-if="!status.worldLoaded" class="hint">No World loaded — onboarding lands in #43.</p>
    </section>

    <section v-else class="disconnected">
      <p>Not connected to a local node.</p>
      <p v-if="error" class="err">{{ error }}</p>
      <button :disabled="checking" @click="refresh">{{ checking ? 'Checking…' : 'Retry' }}</button>
    </section>

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
dl {
  display: grid;
  grid-template-columns: 8rem 1fr;
  row-gap: 0.4rem;
}
dt {
  opacity: 0.6;
}
dd {
  margin: 0;
  font-variant-numeric: tabular-nums;
}
.hint,
.err {
  opacity: 0.7;
  font-size: 0.9rem;
}
.err {
  color: #cf222e;
  font-family: ui-monospace, monospace;
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
