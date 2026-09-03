<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRuntimeConfig } from '#imports'
import type { BootstrapRequest } from './electron/bridge'
import { useNode } from './composables/useNode'
import { shortHex } from './composables/useFormat'
import OnboardingView from './components/OnboardingView.vue'
import NodeDashboard from './components/NodeDashboard.vue'
import ComposeView from './components/ComposeView.vue'
import FeedView from './components/FeedView.vue'
import ConnectionsView from './components/ConnectionsView.vue'
import IdentityView from './components/IdentityView.vue'
import UserAvatar from './components/UserAvatar.vue'
import AppIcon from './components/AppIcon.vue'

const { status, lifecycle, error, busy, view, refresh, bootstrap } = useNode()
const bootstrapError = ref<string | null>(null)
let timer: ReturnType<typeof setInterval> | undefined

type Tab = 'feed' | 'connections' | 'identity'
const tab = ref<Tab>('feed')

// Connections and Identity need author keys; a read-only node only sees a feed,
// so the nav collapses to it (and any active tab falls back).
const canAuthor = computed(() => !!status.value?.canAuthor)
const tabs = computed(() =>
  (
    [
      { id: 'feed' as Tab, label: 'Feed', icon: 'feed', always: true },
      { id: 'connections' as Tab, label: 'Connections', icon: 'users', always: false },
      { id: 'identity' as Tab, label: 'Identity', icon: 'shield', always: false },
    ] as const
  ).filter((t) => t.always || canAuthor.value),
)
watch(canAuthor, (ok) => {
  if (!ok) tab.value = 'feed'
})

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
  <!-- App shell: shown once the node is ready and a World is loaded. -->
  <div v-if="view === 'ready' && status" class="shell">
    <header class="navbar">
      <div class="brand">
        <span class="mark">bp</span>
        <span class="name">BlockParty</span>
      </div>
      <div class="nav-actions">
        <span class="chip is-success"><span class="dot" /> Connected</span>
        <UserAvatar :seed="status.identity" :size="34" />
      </div>
    </header>

    <div class="layout">
      <aside class="col col-left">
        <section class="card profile">
          <UserAvatar :seed="status.identity" :size="64" />
          <p class="who mono">{{ shortHex(status.identity, 8, 6) }}</p>
          <p class="meta">
            <span class="mode">{{ status.mode }}</span> node · v{{ status.version }}
          </p>
          <span class="chip is-primary">
            <AppIcon name="globe" :size="12" /> {{ shortHex(status.world, 6, 4) }}
          </span>
        </section>

        <nav class="card nav">
          <button
            v-for="t in tabs"
            :key="t.id"
            class="nav-item"
            :class="{ 'is-active': tab === t.id }"
            @click="tab = t.id"
          >
            <AppIcon :name="t.icon" />
            <span>{{ t.label }}</span>
          </button>
        </nav>
      </aside>

      <main class="col col-main">
        <template v-if="tab === 'feed'">
          <ComposeView v-if="status.canAuthor" :author="status.identity" />
          <FeedView />
        </template>
        <ConnectionsView v-else-if="tab === 'connections'" />
        <IdentityView v-else />
      </main>

      <aside class="col col-right">
        <NodeDashboard :status="status" />

        <section v-if="lifecycle" class="card">
          <div class="card-heading">
            <AppIcon name="server" :size="16" />
            <h3>Node process</h3>
          </div>
          <div class="card-body lifecycle">
            <div class="row">
              <span class="meta">Mode</span><span class="val">{{ lifecycle.mode }}</span>
            </div>
            <div class="row">
              <span class="meta">State</span>
              <span
                class="chip"
                :class="lifecycle.state === 'running' ? 'is-success' : 'is-danger'"
              >
                {{ lifecycle.state }}
              </span>
            </div>
            <div class="row">
              <span class="meta">Endpoint</span
              ><span class="val mono">{{ lifecycle.endpoint }}</span>
            </div>
            <div v-if="lifecycle.restarts > 0" class="row">
              <span class="meta">Restarts</span><span class="val">{{ lifecycle.restarts }}</span>
            </div>
          </div>
        </section>
      </aside>
    </div>
  </div>

  <!-- Login / setup: a centered card over the background image, shown until the
       shell is ready (connecting, onboarding, or disconnected states). -->
  <div v-else class="auth" :style="authBgStyle">
    <div class="auth-card card">
      <div class="auth-brand">
        <span class="mark">bp</span>
        <span class="name">BlockParty</span>
      </div>

      <OnboardingView v-if="view === 'onboarding'" :busy="busy" @submit="onBootstrap" />

      <section v-else class="auth-status">
        <p v-if="view === 'no-bridge'" class="muted">Running outside Electron — no node bridge.</p>
        <template v-else>
          <span class="spinner" />
          <p class="muted">Connecting to the local node…</p>
        </template>
        <p v-if="error" class="err">{{ error }}</p>
        <button v-if="view !== 'no-bridge'" class="btn is-primary" @click="refresh">Retry</button>
      </section>

      <p v-if="bootstrapError" class="err">{{ bootstrapError }}</p>

      <p v-if="lifecycle" class="meta mono auth-lifecycle">
        node: {{ lifecycle.mode }} · {{ lifecycle.state }} · {{ lifecycle.endpoint }}
      </p>
    </div>
  </div>
</template>

<style scoped>
/* ---------- Shell ---------- */
.shell {
  min-height: 100vh;
}

.navbar {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  height: 60px;
  padding: 0 1.25rem;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  box-shadow: var(--shadow-sm);
}

.brand,
.auth-brand {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-weight: 700;
  letter-spacing: -0.01em;
}

.brand .mark,
.auth-brand .mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 34px;
  width: 34px;
  border-radius: 11px;
  background: linear-gradient(135deg, var(--accent), var(--primary));
  color: #fff;
  font-size: 0.85rem;
  font-weight: 800;
  box-shadow: var(--shadow-primary);
}

.nav-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.layout {
  display: grid;
  grid-template-columns: 260px minmax(0, 640px) 300px;
  gap: 1.25rem;
  justify-content: center;
  align-items: start;
  padding: 1.5rem 1.25rem 3rem;
}

.col {
  display: grid;
  gap: 1rem;
  align-content: start;
}

/* Sticky rails: the feed column is the one that scrolls. */
.col-left,
.col-right {
  position: sticky;
  top: 76px;
}

/* ---------- Left rail ---------- */
.profile {
  display: grid;
  justify-items: center;
  gap: 0.4rem;
  padding: 1.25rem 1rem;
  text-align: center;
}

.profile .who {
  margin: 0.25rem 0 0;
  font-weight: 600;
  color: var(--text);
}

.profile .meta {
  margin: 0;
}

.profile .mode {
  text-transform: capitalize;
}

.nav {
  padding: 0.4rem;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  width: 100%;
  padding: 0.6rem 0.75rem;
  font: inherit;
  font-size: 0.9rem;
  font-weight: 500;
  color: var(--text-medium);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition:
    background 0.2s,
    color 0.2s;
}

.nav-item:hover {
  background: var(--surface-alt);
  color: var(--text);
}

.nav-item.is-active {
  background: rgba(85, 150, 230, 0.14);
  color: var(--primary);
  font-weight: 600;
}

/* ---------- Right rail ---------- */
.lifecycle {
  display: grid;
  gap: 0.55rem;
}

.lifecycle .row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.lifecycle .val {
  font-size: 0.82rem;
  font-weight: 500;
  text-align: right;
  word-break: break-all;
}

/* ---------- Responsive ---------- */
@media (max-width: 1180px) {
  .layout {
    grid-template-columns: 230px minmax(0, 1fr);
  }
  .col-right {
    display: none;
  }
}

@media (max-width: 820px) {
  .layout {
    grid-template-columns: minmax(0, 1fr);
  }
  .col-left {
    position: static;
  }
  .profile {
    display: none;
  }
  .nav {
    display: flex;
    gap: 0.25rem;
  }
  .nav-item {
    justify-content: center;
  }
}

/* ---------- Login / setup ---------- */
.auth {
  position: relative;
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
}

/* Scrim for legibility over the photo. */
.auth::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(160deg, rgba(20, 28, 45, 0.55), rgba(10, 14, 24, 0.72));
}

.auth-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 30rem;
  max-height: calc(100vh - 4rem);
  overflow-y: auto;
  padding: 2rem 1.75rem;
  box-shadow: var(--shadow-md);
}

.auth-brand {
  margin-bottom: 1.5rem;
  font-size: 1.15rem;
}

.auth-status {
  display: grid;
  gap: 0.85rem;
  justify-items: start;
}

.auth-status p {
  margin: 0;
}

.auth-lifecycle {
  margin: 1.5rem 0 0;
  opacity: 0.75;
}

.spinner {
  height: 22px;
  width: 22px;
  border-radius: 50%;
  border: 2px solid var(--border-strong);
  border-top-color: var(--primary);
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
