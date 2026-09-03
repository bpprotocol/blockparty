<script setup lang="ts">
import { ref } from 'vue'
import type { BootstrapRequest } from '../electron/bridge'
import AppIcon from './AppIcon.vue'

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
      one. Your secrets go to the local node, which holds the keys — they are never stored here.
    </p>

    <form @submit.prevent="submit">
      <label class="field">
        <span class="label">World seed</span>
        <textarea
          v-model="worldSeed"
          class="textarea"
          rows="2"
          placeholder="Paste an existing World seed, or generate a new one"
          autocomplete="off"
        />
      </label>
      <button type="button" class="btn is-small is-pill generate" @click="generateSeed">
        <AppIcon name="plus" :size="13" />
        Generate new World seed
      </button>

      <label class="field">
        <span class="label">Identity passphrase</span>
        <input v-model="identityPassphrase" class="input" type="password" autocomplete="off" />
        <span class="help">
          Derives your identity within this World. The same passphrase recovers it later.
        </span>
      </label>

      <label class="field">
        <span class="label">Keystore passphrase</span>
        <input v-model="keystorePassphrase" class="input" type="password" autocomplete="off" />
        <span class="help">
          Encrypts the keystore on this machine; you'll enter it to unlock the node.
        </span>
      </label>

      <label class="field">
        <span class="label">Confirm keystore passphrase</span>
        <input v-model="keystoreConfirm" class="input" type="password" autocomplete="off" />
      </label>

      <p v-if="localError" class="err">{{ localError }}</p>

      <button type="submit" class="btn is-primary is-pill submit" :disabled="props.busy">
        {{ props.busy ? 'Setting up…' : 'Create World' }}
      </button>
    </form>
  </section>
</template>

<style scoped>
.onboarding h2 {
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

.generate {
  justify-self: start;
  margin-top: -0.4rem;
}

.submit {
  padding: 0.7rem 1.1rem;
  font-size: 0.9rem;
}
</style>
