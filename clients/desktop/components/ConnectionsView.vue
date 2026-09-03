<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { formatCard, useConnections } from '../composables/useConnections'
import { shortHex, timeAgo } from '../composables/useFormat'
import type { PrivateMessage } from '../electron/bridge'
import UserAvatar from './UserAvatar.vue'
import AppIcon from './AppIcon.vue'

const {
  myCard,
  list,
  error,
  busy,
  loadCard,
  refresh,
  addAndConnect,
  rotate,
  close,
  send,
  messages,
} = useConnections()

const peerText = ref('')
const copied = ref(false)
const cardOpen = ref(false)
const openPeer = ref<string | null>(null)
const draft = ref('')
const thread = ref<PrivateMessage[]>([])
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | undefined

async function copyCard(): Promise<void> {
  if (!myCard.value) return
  await navigator.clipboard.writeText(formatCard(myCard.value))
  copied.value = true
  setTimeout(() => (copied.value = false), 1500)
}

async function onAdd(): Promise<void> {
  const addr = await addAndConnect(peerText.value)
  if (addr) peerText.value = ''
}

async function openThread(addr: string): Promise<void> {
  openPeer.value = openPeer.value === addr ? null : addr
  if (openPeer.value) thread.value = await messages(addr)
}

async function onSend(addr: string): Promise<void> {
  if (!draft.value.trim()) return
  if (await send(addr, draft.value.trim())) {
    draft.value = ''
    thread.value = await messages(addr)
  }
}

// A message is "mine" when its author is this identity — used to flip the
// bubble to the right, messenger-style.
function isMine(m: PrivateMessage): boolean {
  return !!myCard.value && m.author === myCard.value.address
}

onMounted(() => {
  void loadCard()
  void refresh()
  timer = setInterval(() => {
    now.value = Date.now()
    void refresh()
    if (openPeer.value) void messages(openPeer.value).then((m) => (thread.value = m))
  }, 3000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <section class="connections">
    <header class="head">
      <h2>Connections</h2>
      <span class="chip">{{ list.length }} connected</span>
    </header>

    <!-- Your card: the thing you hand to someone so they can connect. -->
    <section class="card my-card">
      <div class="card-heading">
        <UserAvatar v-if="myCard" :seed="myCard.address" :size="38" />
        <div class="who">
          <span class="name">Your connection card</span>
          <span class="meta mono">{{ myCard ? shortHex(myCard.address, 10, 6) : '—' }}</span>
        </div>
        <span class="spacer" />
        <button class="btn is-small is-pill" @click="copyCard">
          <AppIcon name="copy" :size="13" />
          {{ copied ? 'Copied!' : 'Copy' }}
        </button>
      </div>
      <div class="card-body">
        <button class="btn is-small is-ghost reveal" @click="cardOpen = !cardOpen">
          {{ cardOpen ? 'Hide card text' : 'Show card text' }}
        </button>
        <textarea
          v-if="cardOpen && myCard"
          class="textarea card-text mono"
          readonly
          rows="3"
          :value="formatCard(myCard)"
        />
      </div>
    </section>

    <!-- Add someone by pasting their card. -->
    <form class="card add" @submit.prevent="onAdd">
      <div class="card-heading">
        <AppIcon name="plus" :size="16" />
        <h3>Add a connection</h3>
      </div>
      <div class="card-body">
        <textarea
          v-model="peerText"
          class="textarea"
          rows="2"
          placeholder="Paste a peer's connection card"
        />
        <div class="actions">
          <button class="btn is-primary is-pill" :disabled="busy" type="submit">
            {{ busy ? 'Connecting…' : 'Add & connect' }}
          </button>
        </div>
        <p v-if="error" class="err">{{ error }}</p>
      </div>
    </form>

    <p v-if="!list.length" class="card empty">
      No connections yet. Share your card, or paste a peer's above to connect.
    </p>

    <article v-for="c in list" :key="c.peer" class="card conn">
      <div class="conn-head">
        <UserAvatar :seed="c.peer" :size="42" />
        <div class="who">
          <span class="name mono">{{ shortHex(c.peer, 10, 6) }}</span>
          <span class="meta">epoch {{ c.epoch }}</span>
        </div>
        <div class="actions">
          <button class="btn is-small is-pill" @click="openThread(c.peer)">
            <AppIcon name="send" :size="13" />
            {{ openPeer === c.peer ? 'Hide' : 'Message' }}
          </button>
          <button class="btn is-small is-pill" @click="rotate(c.peer)">Rotate</button>
          <button class="btn is-small is-pill is-ghost close" @click="close(c.peer)">Close</button>
        </div>
      </div>

      <div v-if="openPeer === c.peer" class="thread">
        <p v-if="!thread.length" class="meta no-msgs">No messages yet — say hello.</p>
        <div v-for="(m, i) in thread" :key="i" class="bubble-row" :class="{ mine: isMine(m) }">
          <UserAvatar v-if="!isMine(m)" :seed="m.author" :size="26" />
          <div class="bubble">
            <span class="text">{{ m.text }}</span>
            <span v-if="m.timestamp" class="stamp">{{ timeAgo(m.timestamp, now) }}</span>
          </div>
        </div>

        <form class="reply" @submit.prevent="onSend(c.peer)">
          <input v-model="draft" class="input" placeholder="Private message…" />
          <button type="submit" class="btn is-primary is-icon" aria-label="Send">
            <AppIcon name="send" :size="15" />
          </button>
        </form>
      </div>
    </article>
  </section>
</template>

<style scoped>
.connections {
  display: grid;
  gap: 1rem;
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0.25rem;
}

.head h2 {
  font-size: 1.05rem;
}

.who {
  display: grid;
  line-height: 1.25;
  min-width: 0;
}

.who .name {
  font-weight: 600;
  font-size: 0.9rem;
}

.card-body {
  display: grid;
  gap: 0.6rem;
}

.reveal {
  justify-self: start;
}

.card-text {
  font-size: 0.72rem;
  word-break: break-all;
}

.add .actions {
  display: flex;
  justify-content: flex-end;
}

/* ---------- Connection rows ---------- */
.conn {
  padding: 0.85rem 1rem;
}

.conn-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.conn-head .actions {
  margin-left: auto;
  display: flex;
  gap: 0.35rem;
}

.conn-head .close:hover {
  color: var(--danger);
  border-color: var(--danger);
}

/* ---------- Message thread ---------- */
.thread {
  display: grid;
  gap: 0.5rem;
  margin-top: 0.85rem;
  padding-top: 0.85rem;
  border-top: 1px solid var(--border);
}

.no-msgs {
  margin: 0;
  text-align: center;
}

.bubble-row {
  display: flex;
  align-items: flex-end;
  gap: 0.45rem;
}

.bubble-row.mine {
  justify-content: flex-end;
}

.bubble {
  display: grid;
  gap: 0.2rem;
  max-width: 78%;
  padding: 0.5rem 0.8rem;
  border-radius: 1rem 1rem 1rem 0.25rem;
  background: var(--surface-alt);
  font-size: 0.88rem;
  overflow-wrap: anywhere;
}

.bubble-row.mine .bubble {
  border-radius: 1rem 1rem 0.25rem 1rem;
  background: var(--primary);
  color: #fff;
}

.bubble .stamp {
  font-size: 0.68rem;
  opacity: 0.65;
}

.reply {
  display: flex;
  gap: 0.45rem;
  margin-top: 0.25rem;
}

.reply .input {
  border-radius: var(--radius-pill);
}
</style>
