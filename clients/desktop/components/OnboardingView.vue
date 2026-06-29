<script setup lang="ts">
import { ref } from 'vue'
import type { BootstrapRequest } from '../electron/bridge'

const props = defineProps<{ busy: boolean }>()
const emit = defineEmits<{ submit: [req: BootstrapRequest] }>()

const worldSeed = ref('')
const identityPassphrase = ref('')
const keystorePassphrase = ref('')
const keystoreConfirm = ref('')
const localError = ref<string | null>(null)

// A fresh World needs a high-entropy seed phrase, shared by everyone who joins
// it. Generate one client-side; the user can also paste an existing seed to join.
function generateSeed(): void {
  const bytes = new Uint8Array(24)
  crypto.getRandomValues(bytes)
  worldSeed.value = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}

function submit(): void {
  localError.value = null
  if (!worldSeed.value.trim()) {
    localError.value = 'Enter or generate a World seed.'
    return
  }
  if (!keystorePassphrase.value) {
    localError.value = 'A keystore passphrase is required to encrypt your keys.'
    return
  }
  if (keystorePassphrase.value !== keystoreConfirm.value) {
    localError.value = 'Keystore passphrases do not match.'
    return
  }
  emit('submit', {
    worldSeed: worldSeed.value.trim(),
    identityPassphrase: identityPassphrase.value,
    keystorePassphrase: keystorePassphrase.value,
  })
}
</script>

<template>
  <section class="onboarding">
    <h2>Set up your World</h2>
    <p class="lead">
      This node has no World loaded. Join an existing World by pasting its seed, or create a new
      one. Your secrets are sent to the local node, which holds the keys — they are never stored
      here.
    </p>

    <form @submit.prevent="submit">
      <label>
        <span>World seed</span>
        <textarea
          v-model="worldSeed"
          rows="2"
          placeholder="Paste an existing World seed, or generate a new one"
          autocomplete="off"
        />
      </label>
      <button type="button" class="ghost" @click="generateSeed">Generate new World seed</button>

      <label>
        <span>Identity passphrase</span>
        <input v-model="identityPassphrase" type="password" autocomplete="off" />
        <small
          >Derives your identity within this World. The same passphrase recovers it later.</small
        >
      </label>

      <label>
        <span>Keystore passphrase</span>
        <input v-model="keystorePassphrase" type="password" autocomplete="off" />
        <small>Encrypts the keystore on this machine; you'll enter it to unlock the node.</small>
      </label>
      <label>
        <span>Confirm keystore passphrase</span>
        <input v-model="keystoreConfirm" type="password" autocomplete="off" />
      </label>

      <p v-if="localError" class="err">{{ localError }}</p>
      <button type="submit" class="primary" :disabled="props.busy">
        {{ props.busy ? 'Setting up…' : 'Create World' }}
      </button>
    </form>
  </section>
</template>

<style scoped>
.onboarding {
  max-width: 34rem;
}
.lead {
  opacity: 0.75;
  font-size: 0.95rem;
}
form {
  display: grid;
  gap: 1rem;
  margin-top: 1.5rem;
}
label {
  display: grid;
  gap: 0.3rem;
}
label > span {
  font-weight: 600;
  font-size: 0.9rem;
}
input,
textarea {
  font: inherit;
  padding: 0.5rem;
  border: 1px solid color-mix(in srgb, currentColor 25%, transparent);
  border-radius: 6px;
  background: transparent;
  color: inherit;
}
small {
  opacity: 0.6;
}
button {
  padding: 0.5rem 1rem;
  border-radius: 6px;
  cursor: pointer;
}
.ghost {
  justify-self: start;
  background: transparent;
  border: 1px solid color-mix(in srgb, currentColor 30%, transparent);
  color: inherit;
}
.primary {
  background: #1a7f37;
  color: white;
  border: none;
}
.primary:disabled {
  opacity: 0.6;
}
.err {
  color: #cf222e;
  font-size: 0.9rem;
}
</style>
