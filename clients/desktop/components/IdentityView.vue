<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useIdentity } from '../composables/useIdentity'
import type { IdentityCard } from '../electron/bridge'

const { busy, error, notice, rotate, burn } = useIdentity()

const card = ref<IdentityCard | null>(null)
const world = ref('')

const newPass = ref('')
const rotateOpen = ref(false)
const burnOpen = ref(false)
const burnReason = ref('compromised')
const burnConfirmText = ref('')

async function loadCard(): Promise<void> {
  const bridge = typeof window !== 'undefined' ? window.bpDesktop : undefined
  if (!bridge) return
  const [id, st] = await Promise.all([bridge.connections.getIdentity(), bridge.node.getStatus()])
  if (id.ok) card.value = id.value
  if (st.ok) world.value = st.value.world
}

async function doRotate(): Promise<void> {
  // The user explicitly confirms by opening this panel and submitting; pass
  // confirm: true. The node also enforces the gate server-side.
  const addr = await rotate(newPass.value, true)
  if (addr) {
    newPass.value = ''
    rotateOpen.value = false
    await loadCard()
  }
}

async function doBurn(): Promise<void> {
  // Require typing BURN as a second, deliberate confirmation before sending.
  if (burnConfirmText.value !== 'BURN') {
    error.value = 'Type BURN to confirm.'
    return
  }
  await burn(burnReason.value, true)
  burnConfirmText.value = ''
  burnOpen.value = false
  await loadCard()
}

function short(s: string): string {
  return s ? `${s.slice(0, 16)}…` : '—'
}

onMounted(loadCard)
</script>

<template>
  <section class="identity">
    <h2>Identity</h2>
    <dl v-if="card">
      <dt>Address</dt>
      <dd>{{ card.address }}</dd>
      <dt>World</dt>
      <dd>{{ short(world) }}</dd>
      <dt>Signing key</dt>
      <dd>{{ short(card.mldsaPub) }}</dd>
      <dt>KEM key</dt>
      <dd>{{ short(card.kyberPub) }}</dd>
    </dl>

    <p v-if="notice" class="notice">{{ notice }}</p>
    <p v-if="error" class="err">{{ error }}</p>

    <div class="danger">
      <h3>Key management</h3>

      <button v-if="!rotateOpen" class="ghost" @click="rotateOpen = true">Rotate identity…</button>
      <form v-else class="op" @submit.prevent="doRotate">
        <p class="warn">
          Rotating derives a new identity from a new passphrase. Existing connections are dropped.
        </p>
        <input
          v-model="newPass"
          type="password"
          placeholder="New identity passphrase"
          autocomplete="off"
        />
        <div class="row">
          <button type="button" class="ghost" @click="rotateOpen = false">Cancel</button>
          <button type="submit" class="primary" :disabled="busy">Confirm rotate</button>
        </div>
      </form>

      <button v-if="!burnOpen" class="danger-btn" @click="burnOpen = true">Burn identity…</button>
      <form v-else class="op" @submit.prevent="doBurn">
        <p class="warn">
          Burning publishes an <code>identity.burn</code> that reveals this identity's private keys
          so peers stop trusting it. <strong>This is irreversible.</strong>
        </p>
        <label>
          Reason
          <select v-model="burnReason">
            <option value="voluntary">voluntary</option>
            <option value="compromised">compromised</option>
            <option value="rotated">rotated</option>
            <option value="other">other</option>
          </select>
        </label>
        <input v-model="burnConfirmText" placeholder="Type BURN to confirm" autocomplete="off" />
        <div class="row">
          <button type="button" class="ghost" @click="burnOpen = false">Cancel</button>
          <button type="submit" class="danger-btn" :disabled="busy">Burn identity</button>
        </div>
      </form>
    </div>
  </section>
</template>

<style scoped>
.identity {
  margin-top: 1.5rem;
}
dl {
  display: grid;
  grid-template-columns: 8rem 1fr;
  row-gap: 0.3rem;
}
dt {
  opacity: 0.6;
}
dd {
  margin: 0;
  font-family: ui-monospace, monospace;
  font-size: 0.85rem;
  word-break: break-all;
}
.danger {
  margin-top: 1.25rem;
  border: 1px solid color-mix(in srgb, #cf222e 40%, transparent);
  border-radius: 8px;
  padding: 0.75rem 1rem;
}
.danger h3 {
  margin: 0 0 0.5rem;
  font-size: 0.95rem;
}
.op {
  display: grid;
  gap: 0.5rem;
  margin: 0.5rem 0;
}
.warn {
  font-size: 0.85rem;
  opacity: 0.8;
  margin: 0;
}
input,
select {
  font: inherit;
  padding: 0.45rem;
  border: 1px solid color-mix(in srgb, currentColor 20%, transparent);
  border-radius: 6px;
  background: transparent;
  color: inherit;
}
.row {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
}
button {
  padding: 0.4rem 0.9rem;
  border-radius: 6px;
  cursor: pointer;
}
.primary {
  background: #1a7f37;
  color: white;
  border: none;
}
.ghost {
  background: transparent;
  border: 1px solid color-mix(in srgb, currentColor 25%, transparent);
  color: inherit;
}
.danger-btn {
  background: #cf222e;
  color: white;
  border: none;
  margin-top: 0.5rem;
}
button:disabled {
  opacity: 0.6;
}
.notice {
  color: #1a7f37;
}
.err {
  color: #cf222e;
}
</style>
