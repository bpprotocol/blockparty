<script setup lang="ts">
import { ref } from 'vue'
import AppIcon from './AppIcon.vue'

// Shown when the node holds an encrypted keystore but was started without its
// passphrase (no BPNODE_KEYSTORE_PASSPHRASE), so no World is loaded. The
// passphrase goes straight to the node, which derives the keys in memory.
const props = defineProps<{ busy: boolean }>()
const emit = defineEmits<{ submit: [passphrase: string]; clear: [] }>()

const passphrase = ref('')
const localError = ref<string | null>(null)

const clearOpen = ref(false)
const clearConfirmText = ref('')

function submit(): void {
  localError.value = null
  if (!passphrase.value) {
    localError.value = 'Enter your keystore passphrase.'
    return
  }
  emit('submit', passphrase.value)
  passphrase.value = ''
}

function cancelClear(): void {
  clearOpen.value = false
  clearConfirmText.value = ''
  localError.value = null
}

// Clearing throws away the World seed and identity for good, so — like the
// identity burn — it takes a second, deliberate confirmation: the user types
// CLEAR. The node enforces its own confirm flag on top of this (#29).
function doClear(): void {
  localError.value = null
  if (clearConfirmText.value !== 'CLEAR') {
    localError.value = 'Type CLEAR to confirm.'
    return
  }
  clearConfirmText.value = ''
  clearOpen.value = false
  emit('clear')
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

    <div class="recovery">
      <button v-if="!clearOpen" class="btn is-small is-ghost" @click="clearOpen = true">
        Forgotten your passphrase?
      </button>

      <form v-else class="op" @submit.prevent="doClear">
        <p class="warn">
          <strong>Clearing the keystore is irreversible.</strong> The World seed and identity it
          protects cannot be recovered without the passphrase, so you would start a new World — with
          a new address, no connections, and existing posts left unreadable.
        </p>
        <input
          v-model="clearConfirmText"
          class="input"
          placeholder="Type CLEAR to confirm"
          autocomplete="off"
        />
        <div class="row">
          <button type="button" class="btn is-small" @click="cancelClear">Cancel</button>
          <button type="submit" class="btn is-small is-danger" :disabled="props.busy">
            Clear keystore
          </button>
        </div>
      </form>
    </div>
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

.recovery {
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border);
}

.op {
  display: grid;
  gap: 0.6rem;
  padding: 0.85rem;
  border-radius: var(--radius-sm);
  background: var(--danger-soft);
}

.warn {
  margin: 0;
  font-size: 0.82rem;
  color: var(--text-medium);
}

.row {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
}
</style>
