<script setup lang="ts">
import type { NodeStatus } from '../electron/bridge'

defineProps<{ status: NodeStatus }>()

function short(s: string): string {
  return s ? `${s.slice(0, 12)}…` : '—'
}
</script>

<template>
  <section class="status">
    <dl>
      <dt>Node</dt>
      <dd>v{{ status.version }} · {{ status.mode }} mode</dd>
      <dt>World</dt>
      <dd>{{ short(status.world) }}</dd>
      <dt>Identity</dt>
      <dd>{{ status.identity ? short(status.identity) : '—' }}</dd>
      <dt>Blocks</dt>
      <dd>{{ status.blockCount }}</dd>
      <dt>Can author</dt>
      <dd>{{ status.canAuthor ? 'yes' : 'no' }}</dd>
    </dl>
    <p class="hint">Feed and composing land in #44–#45.</p>
  </section>
</template>

<style scoped>
dl {
  display: grid;
  grid-template-columns: 8rem 1fr;
  row-gap: 0.4rem;
}
dt {
  opacity: 0.6;
}
dd {
  margin: 0;
  font-variant-numeric: tabular-nums;
}
.hint {
  opacity: 0.6;
  font-size: 0.9rem;
}
</style>
