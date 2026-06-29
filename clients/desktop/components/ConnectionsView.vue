<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { formatCard, useConnections } from '../composables/useConnections'
import type { PrivateMessage } from '../electron/bridge'

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
const openPeer = ref<string | null>(null)
const draft = ref('')
const thread = ref<PrivateMessage[]>([])
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

function short(s: string): string {
  return s ? `${s.slice(0, 10)}…` : 'unknown'
}

onMounted(() => {
  void loadCard()
  void refresh()
  timer = setInterval(() => {
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
    <h2>Connections</h2>

    <details class="card">
      <summary>Your connection card — share it to let someone connect</summary>
      <textarea v-if="myCard" readonly :value="formatCard(myCard)" rows="3" />
      <button class="ghost" @click="copyCard">{{ copied ? 'Copied!' : 'Copy card' }}</button>
    </details>

    <form class="add" @submit.prevent="onAdd">
      <textarea v-model="peerText" rows="2" placeholder="Paste a peer's connection card" />
      <button class="primary" :disabled="busy" type="submit">
        {{ busy ? 'Connecting…' : 'Add & connect' }}
      </button>
    </form>
    <p v-if="error" class="err">{{ error }}</p>

    <ul v-if="list.length" class="list">
      <li v-for="c in list" :key="c.peer" class="conn">
        <div class="row">
          <span class="peer">{{ short(c.peer) }}</span>
          <span class="epoch">epoch {{ c.epoch }}</span>
          <span class="actions">
            <button @click="openThread(c.peer)">
              {{ openPeer === c.peer ? 'Hide' : 'Messages' }}
            </button>
            <button @click="rotate(c.peer)">Rotate</button>
            <button @click="close(c.peer)">Close</button>
          </span>
        </div>

        <div v-if="openPeer === c.peer" class="thread">
          <p v-for="(m, i) in thread" :key="i" class="msg">
            <span class="who">{{ short(m.author) }}:</span> {{ m.text }}
          </p>
          <form class="reply" @submit.prevent="onSend(c.peer)">
            <input v-model="draft" placeholder="Private message…" />
            <button type="submit">Send</button>
          </form>
        </div>
      </li>
    </ul>
    <p v-else class="empty">No connections yet. Share your card or paste a peer's to connect.</p>
  </section>
</template>

<style scoped>
.connections {
  margin-top: 1.5rem;
}
textarea,
input {
  font: inherit;
  width: 100%;
  box-sizing: border-box;
  padding: 0.5rem;
  border: 1px solid color-mix(in srgb, currentColor 20%, transparent);
  border-radius: 6px;
  background: transparent;
  color: inherit;
}
.card textarea {
  margin-top: 0.5rem;
  font-family: ui-monospace, monospace;
  font-size: 0.75rem;
  word-break: break-all;
}
.add {
  display: grid;
  gap: 0.5rem;
  margin-top: 1rem;
}
.list {
  list-style: none;
  padding: 0;
  margin: 1rem 0 0;
  display: grid;
  gap: 0.5rem;
}
.conn {
  border: 1px solid color-mix(in srgb, currentColor 12%, transparent);
  border-radius: 8px;
  padding: 0.6rem 0.9rem;
}
.row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.peer {
  font-family: ui-monospace, monospace;
  font-weight: 600;
}
.epoch {
  opacity: 0.55;
  font-size: 0.8rem;
}
.actions {
  margin-left: auto;
  display: flex;
  gap: 0.4rem;
}
.thread {
  margin-top: 0.6rem;
  border-top: 1px solid color-mix(in srgb, currentColor 12%, transparent);
  padding-top: 0.5rem;
}
.msg {
  margin: 0.2rem 0;
}
.who {
  opacity: 0.55;
  font-family: ui-monospace, monospace;
  font-size: 0.8rem;
}
.reply {
  display: flex;
  gap: 0.4rem;
  margin-top: 0.4rem;
}
.reply input {
  flex: 1;
}
button {
  padding: 0.3rem 0.7rem;
  border-radius: 6px;
  cursor: pointer;
}
.primary {
  background: #1a7f37;
  color: white;
  border: none;
}
.ghost {
  margin-top: 0.4rem;
  background: transparent;
  border: 1px solid color-mix(in srgb, currentColor 25%, transparent);
  color: inherit;
}
.err {
  color: #cf222e;
  font-size: 0.9rem;
}
.empty {
  opacity: 0.6;
}
</style>
