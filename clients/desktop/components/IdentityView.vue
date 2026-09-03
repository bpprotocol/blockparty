<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useIdentity } from '../composables/useIdentity'
import { shortHex } from '../composables/useFormat'
import type { IdentityCard } from '../electron/bridge'
import UserAvatar from './UserAvatar.vue'
import AppIcon from './AppIcon.vue'

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

onMounted(loadCard)
</script>

<template>
  <section class="identity">
    <!-- Profile header: the closest thing an identity has to a profile page. -->
    <section class="card profile">
      <div class="cover" />
      <div class="profile-body">
        <UserAvatar :seed="card?.address ?? ''" :size="84" />
        <h2 class="mono addr">{{ card ? shortHex(card.address, 12, 8) : '—' }}</h2>
        <span class="chip is-primary">
          <AppIcon name="globe" :size="12" /> world {{ shortHex(world, 6, 4) }}
        </span>
      </div>
    </section>

    <section v-if="card" class="card keys">
      <div class="card-heading">
        <AppIcon name="key" :size="16" />
        <h3>Keys</h3>
      </div>
      <dl class="card-body">
        <dt class="meta">Address</dt>
        <dd class="mono">{{ card.address }}</dd>
        <dt class="meta">Signing key</dt>
        <dd class="mono">{{ shortHex(card.mldsaPub, 24, 8) }}</dd>
        <dt class="meta">KEM key</dt>
        <dd class="mono">{{ shortHex(card.kyberPub, 24, 8) }}</dd>
      </dl>
    </section>

    <p v-if="notice" class="notice">{{ notice }}</p>
    <p v-if="error" class="err">{{ error }}</p>

    <!-- Key management: destructive, so it reads as a danger zone. -->
    <section class="card danger">
      <div class="card-heading">
        <AppIcon name="alert" :size="16" />
        <h3>Key management</h3>
      </div>

      <div class="card-body">
        <div class="op-row">
          <div>
            <p class="op-title">Rotate identity</p>
            <p class="meta">Derive a new identity from a new passphrase.</p>
          </div>
          <button v-if="!rotateOpen" class="btn is-small is-pill" @click="rotateOpen = true">
            Rotate…
          </button>
        </div>

        <form v-if="rotateOpen" class="op" @submit.prevent="doRotate">
          <p class="warn">
            Rotating derives a new identity from a new passphrase. Existing connections are dropped.
          </p>
          <input
            v-model="newPass"
            class="input"
            type="password"
            placeholder="New identity passphrase"
            autocomplete="off"
          />
          <div class="row">
            <button type="button" class="btn is-small" @click="rotateOpen = false">Cancel</button>
            <button type="submit" class="btn is-small is-primary" :disabled="busy">
              Confirm rotate
            </button>
          </div>
        </form>

        <div class="op-row">
          <div>
            <p class="op-title">Burn identity</p>
            <p class="meta">Publish proof this identity is dead. Irreversible.</p>
          </div>
          <button v-if="!burnOpen" class="btn is-small is-pill is-danger" @click="burnOpen = true">
            Burn…
          </button>
        </div>

        <form v-if="burnOpen" class="op is-burn" @submit.prevent="doBurn">
          <p class="warn">
            Burning publishes an <code>identity.burn</code> that reveals this identity's private
            keys so peers stop trusting it. <strong>This is irreversible.</strong>
          </p>
          <label class="field">
            <span class="label">Reason</span>
            <select v-model="burnReason" class="select">
              <option value="voluntary">voluntary</option>
              <option value="compromised">compromised</option>
              <option value="rotated">rotated</option>
              <option value="other">other</option>
            </select>
          </label>
          <input
            v-model="burnConfirmText"
            class="input"
            placeholder="Type BURN to confirm"
            autocomplete="off"
          />
          <div class="row">
            <button type="button" class="btn is-small" @click="burnOpen = false">Cancel</button>
            <button type="submit" class="btn is-small is-danger" :disabled="busy">
              Burn identity
            </button>
          </div>
        </form>
      </div>
    </section>
  </section>
</template>

<style scoped>
.identity {
  display: grid;
  gap: 1rem;
}

/* ---------- Profile header ---------- */
.profile {
  overflow: hidden;
}

.cover {
  height: 96px;
  background: linear-gradient(135deg, var(--accent), var(--primary));
}

.profile-body {
  display: grid;
  justify-items: center;
  gap: 0.5rem;
  padding: 0 1rem 1.25rem;
  margin-top: -42px;
  text-align: center;
}

.profile-body :deep(.avatar) {
  box-shadow: 0 0 0 4px var(--surface);
}

.addr {
  font-size: 0.95rem;
  font-weight: 600;
  word-break: break-all;
}

/* ---------- Keys ---------- */
.keys dl {
  display: grid;
  grid-template-columns: 7rem minmax(0, 1fr);
  align-items: baseline;
  gap: 0.4rem 0.75rem;
  margin: 0;
}

.keys dd {
  margin: 0;
  word-break: break-all;
}

/* ---------- Danger zone ---------- */
.danger {
  border-color: color-mix(in srgb, var(--danger) 35%, transparent);
}

.danger .card-heading {
  color: var(--danger);
  border-bottom-color: color-mix(in srgb, var(--danger) 22%, transparent);
}

.danger .card-body {
  display: grid;
  gap: 0.85rem;
}

.op-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.op-title {
  margin: 0;
  font-size: 0.9rem;
  font-weight: 600;
}

.op-row .meta {
  margin: 0;
}

.op {
  display: grid;
  gap: 0.6rem;
  padding: 0.85rem;
  border-radius: var(--radius-sm);
  background: var(--surface-alt);
}

.op.is-burn {
  background: var(--danger-soft);
}

.warn {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-medium);
}

.row {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
}

code {
  font-family: var(--font-mono);
  font-size: 0.85em;
}
</style>
