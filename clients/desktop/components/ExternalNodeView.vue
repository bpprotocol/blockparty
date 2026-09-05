<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { ExternalNodeConfig } from '../electron/bridge'
import AppIcon from './AppIcon.vue'

// Shown when the app cannot run a node itself — the client-only build (#51) —
// and has none configured yet, or when the user wants to point it somewhere
// else. The token never reaches this screen from disk; it only travels one way,
// to the main process, which holds it for the API calls (#29).
const props = withDefaults(defineProps<{ busy: boolean; canCancel?: boolean }>(), {
  canCancel: false,
})
const emit = defineEmits<{ submit: [cfg: ExternalNodeConfig]; cancel: [] }>()

const address = ref('')
const token = ref('')
const dataDir = ref('')
const useDataDir = ref(false)
const localError = ref<string | null>(null)

async function loadSaved(): Promise<void> {
  const saved = await window.bpDesktop?.externalNode.get()
  if (!saved) return
  address.value = saved.baseUrl
  dataDir.value = saved.dataDir ?? ''
  useDataDir.value = !!saved.dataDir && !saved.token
}

function submit(): void {
  localError.value = null
  if (!address.value.trim()) {
    localError.value = 'Enter the node API address.'
    return
  }
  if (useDataDir.value && !dataDir.value.trim()) {
    localError.value = "Enter the node's data directory, or paste its token instead."
    return
  }
  if (!useDataDir.value && !token.value.trim()) {
    localError.value = "Paste the node's API token, or point at its data directory instead."
    return
  }
  emit('submit', {
    baseUrl: address.value.trim(),
    token: useDataDir.value ? undefined : token.value.trim(),
    dataDir: useDataDir.value ? dataDir.value.trim() : undefined,
  })
}

onMounted(loadSaved)
</script>

<template>
  <section class="external">
    <h2>Connect to a node</h2>
    <p class="lead">
      This build doesn't ship a node of its own, so it works as a client for one you already run —
      on this machine or another. Point it at that node's local API.
    </p>

    <form @submit.prevent="submit">
      <label class="field">
        <span class="label">Node API address</span>
        <input v-model="address" class="input" placeholder="127.0.0.1:4400" autocomplete="off" />
        <span class="help">Host and port, or a full URL. The node binds loopback by default.</span>
      </label>

      <div class="toggle">
        <button
          type="button"
          class="btn is-small is-pill"
          :class="{ 'is-primary': !useDataDir }"
          @click="useDataDir = false"
        >
          Paste token
        </button>
        <button
          type="button"
          class="btn is-small is-pill"
          :class="{ 'is-primary': useDataDir }"
          @click="useDataDir = true"
        >
          Read from data dir
        </button>
      </div>

      <label v-if="!useDataDir" class="field">
        <span class="label">API token</span>
        <input v-model="token" class="input" type="password" autocomplete="off" />
        <span class="help">The contents of the node's <code>api.token</code>.</span>
      </label>

      <label v-else class="field">
        <span class="label">Node data directory</span>
        <input
          v-model="dataDir"
          class="input"
          placeholder="~/.config/blockparty/node"
          autocomplete="off"
        />
        <span class="help">
          Only works when that directory is readable from this machine; the app reads
          <code>api.token</code> from it.
        </span>
      </label>

      <p v-if="localError" class="err">{{ localError }}</p>

      <div class="actions">
        <button v-if="props.canCancel" type="button" class="btn is-small" @click="emit('cancel')">
          Cancel
        </button>
        <button type="submit" class="btn is-primary is-pill submit" :disabled="props.busy">
          <AppIcon name="server" :size="14" />
          {{ props.busy ? 'Connecting…' : 'Connect' }}
        </button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.external h2 {
  font-size: 1.15rem;
}

.lead {
  margin: 0.4rem 0 0;
  color: var(--text-muted);
  font-size: 0.88rem;
}

form {
  display: grid;
  gap: 1rem;
  margin-top: 1.5rem;
}

.toggle {
  display: flex;
  gap: 0.4rem;
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.submit {
  padding: 0.7rem 1.1rem;
  font-size: 0.9rem;
}

code {
  font-family: var(--font-mono);
  font-size: 0.85em;
}
</style>
