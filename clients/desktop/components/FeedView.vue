<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useFeed } from '../composables/useFeed'

const { items, error, loading, load, start, stop } = useFeed()

function short(s: string): string {
  return s ? `${s.slice(0, 10)}…` : 'unknown'
}

function when(ts: number): string {
  return ts ? new Date(ts * 1000).toLocaleString() : ''
}

onMounted(() => {
  void load()
  void start()
})
onUnmounted(stop)
</script>

<template>
  <section class="feed">
    <header>
      <h2>Feed</h2>
      <button :disabled="loading" @click="load">{{ loading ? 'Loading…' : 'Refresh' }}</button>
    </header>

    <p v-if="error" class="err">{{ error }}</p>
    <p v-else-if="items.length === 0 && !loading" class="empty">
      No posts yet. Posting lands in #45.
    </p>

    <ul>
      <li v-for="item in items" :key="item.id" class="post">
        <p class="text">{{ item.text }}</p>
        <p class="meta">{{ short(item.author) }} · {{ when(item.timestamp) }}</p>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.feed {
  margin-top: 1.5rem;
}
header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}
ul {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 0.75rem;
}
.post {
  border: 1px solid color-mix(in srgb, currentColor 12%, transparent);
  border-radius: 8px;
  padding: 0.75rem 1rem;
}
.text {
  margin: 0 0 0.4rem;
  white-space: pre-wrap;
}
.meta {
  margin: 0;
  font-size: 0.8rem;
  opacity: 0.55;
  font-family: ui-monospace, monospace;
}
.empty {
  opacity: 0.6;
}
.err {
  color: #cf222e;
}
button {
  padding: 0.3rem 0.8rem;
}
</style>
