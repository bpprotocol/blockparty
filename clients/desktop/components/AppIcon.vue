<script setup lang="ts">
import { computed } from 'vue'

// Inline feather-style icon set. Inlined rather than pulled from an icon font
// so the packaged app renders identically offline over file://.
const PATHS: Record<string, string> = {
  feed: 'M4 5h16M4 12h16M4 19h10',
  users:
    'M17 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9.5 7a3.5 3.5 0 1 1-7 0 3.5 3.5 0 0 1 7 0zM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75',
  shield: 'M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z',
  server: 'M3 4h18v6H3zM3 14h18v6H3zM7 7h.01M7 17h.01',
  refresh: 'M21 12a9 9 0 1 1-3-6.7M21 3v6h-6',
  send: 'M22 2 11 13M22 2l-7 20-4-9-9-4 20-7z',
  copy: 'M9 9h11v11H9zM5 15H4V4h11v1',
  plus: 'M12 5v14M5 12h14',
  globe:
    'M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18zM3 12h18M12 3c2.5 2.7 3.8 5.7 3.8 9S14.5 18.3 12 21c-2.5-2.7-3.8-5.7-3.8-9S9.5 5.7 12 3z',
  key: 'M11.4 11.6a5.5 5.5 0 1 0-7.8 7.8 5.5 5.5 0 0 0 7.8-7.8zm0 0L15.5 7.5m0 0 3 3L22 7l-3-3m-3.5 3.5L19 4',
  flame:
    'M12 22c4 0 6-2.7 6-6 0-4-3-6-3-9 0 0-3 1.5-3 5 0-1.5-1-3-2.5-4C9 10 6 12 6 16c0 3.3 2 6 6 6z',
  logout: 'M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9',
}

const props = withDefaults(defineProps<{ name: keyof typeof PATHS | string; size?: number }>(), {
  size: 18,
})

const d = computed(() => PATHS[props.name] ?? PATHS.feed)
</script>

<template>
  <svg
    class="icon"
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
  >
    <path :d="d" />
  </svg>
</template>

<style scoped>
.icon {
  flex: none;
  display: block;
}
</style>
