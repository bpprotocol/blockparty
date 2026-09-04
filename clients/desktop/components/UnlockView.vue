<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from './AppIcon.vue'

// Shown when the node holds an encrypted keystore but was started without its
// passphrase (no BPNODE_KEYSTORE_PASSPHRASE), so no World is loaded. The
// passphrase goes straight to the node, which derives the keys in memory.
const props = defineProps<{ busy: boolean }>()
const emit = defineEmits<{ submit: [passphrase: string] }>()

const passphrase = ref('')
const localError = ref<string | null>(null)

function submit(): void {
  localError.value = null
  if (!passphrase.value) {
    localError.value = 'Enter your keystore passphrase.'
    return
  }
  emit('submit', passphrase.value)
  passphrase.value = ''
}
</script>

<template>
  <section class="unlock">
    <h2>Welcome back</h2>
    <p class="lead">
      This node already holds an encrypted keystore. Enter its passphrase to unlock your World and
      identity — the keys are re-derived in memory by the node and never written to disk.
    </p>

    <form @submit.prevent="submit">
      <label class="field">
        <span class="label">Keystore passphrase</span>
        <input
          v-model="passphrase"
          class="input"
          type="password"
          autocomplete="current-password"
          autofocus
        />
      </label>

      <p v-if="localError" class="err">{{ localError }}</p>

      <button type="submit" class="btn is-primary is-pill submit" :disabled="props.busy">
        <AppIcon name="key" :size="14" />
        {{ props.busy ? 'Unlocking…' : 'Unlock' }}
      </button>
    </form>

    <p class="hint meta">
      Forgotten it? The World seed inside the keystore can't be recovered — clear the node's data
      directory to start a new World.
    </p>
  </section>
</template>

<style scoped>
.unlock h2 {
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

.submit {
  padding: 0.7rem 1.1rem;
  font-size: 0.9rem;
}

.hint {
  margin: 1.25rem 0 0;
}
</style>
