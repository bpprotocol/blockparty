<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useFeed } from '../composables/useFeed'
import { shortHex, timeAgo } from '../composables/useFormat'
import UserAvatar from './UserAvatar.vue'
import AppIcon from './AppIcon.vue'

const { items, error, loading, load, start, stop } = useFeed()

// Relative timestamps re-render on a slow tick so "just now" ages by itself.
const now = ref(Date.now())
const copiedId = ref<string | null>(null)
let tick: ReturnType<typeof setInterval> | undefined

async function copyId(id: string): Promise<void> {
  await navigator.clipboard.writeText(id)
  copiedId.value = id
  setTimeout(() => {
    if (copiedId.value === id) copiedId.value = null
  }, 1500)
}

onMounted(() => {
  void load()
  void start()
  tick = setInterval(() => (now.value = Date.now()), 30000)
})
onUnmounted(() => {
  if (tick) clearInterval(tick)
  stop()
})
</script>

<template>
  <section class="feed">
    <header class="feed-head">
      <h2>Feed</h2>
      <button class="btn is-small is-pill" :disabled="loading" @click="load">
        <AppIcon name="refresh" :size="14" />
        {{ loading ? 'Loading…' : 'Refresh' }}
      </button>
    </header>

    <p v-if="error" class="card err feed-err">{{ error }}</p>

    <p v-else-if="items.length === 0 && !loading" class="card empty">
      No public posts yet. Share something above, or wait for peers to show up.
    </p>

    <article v-for="item in items" :key="item.id" class="card post">
      <header class="post-head">
        <UserAvatar :seed="item.author" :size="42" />
        <div class="who">
          <span class="name mono">{{ shortHex(item.author, 8, 4) }}</span>
          <span class="meta">{{ timeAgo(item.timestamp, now) }}</span>
        </div>
        <span class="chip is-primary">public-{{ item.publicAudience }}</span>
      </header>

      <p class="post-text">{{ item.text }}</p>

      <footer class="post-foot">
        <span class="meta mono">{{ shortHex(item.id, 10, 6) }}</span>
        <button class="btn is-small is-ghost is-pill" @click="copyId(item.id)">
          <AppIcon name="copy" :size="13" />
          {{ copiedId === item.id ? 'Copied' : 'Copy ID' }}
        </button>
      </footer>
    </article>
  </section>
</template>

<style scoped>
.feed {
  display: grid;
  gap: 1rem;
}

.feed-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 0.25rem;
}

.feed-head h2 {
  font-size: 1.05rem;
}

.feed-err {
  padding: 1rem;
}

.post {
  padding: 1rem 1.1rem 0.6rem;
  transition: box-shadow 0.2s;
}

.post:hover {
  box-shadow: var(--shadow);
}

.post-head {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.post-head .who {
  display: grid;
  line-height: 1.25;
}

.post-head .name {
  font-weight: 600;
  color: var(--text);
}

.post-head .chip {
  margin-left: auto;
}

.post-text {
  margin: 0.85rem 0 0.75rem;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.post-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border);
}
</style>
