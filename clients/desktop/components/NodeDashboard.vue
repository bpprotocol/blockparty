<script setup lang="ts">
import type { NodeStatus } from '../electron/bridge'
import { shortHex } from '../composables/useFormat'
import AppIcon from './AppIcon.vue'

defineProps<{ status: NodeStatus }>()
</script>

<template>
  <section class="card">
    <div class="card-heading">
      <AppIcon name="globe" :size="16" />
      <h3>World</h3>
      <span class="spacer" />
      <span class="chip" :class="status.canAuthor ? 'is-success' : ''">
        {{ status.canAuthor ? 'can post' : 'read only' }}
      </span>
    </div>

    <div class="card-body stats">
      <div class="tile">
        <span class="num">{{ status.blockCount }}</span>
        <span class="meta">blocks</span>
      </div>
      <div class="tile">
        <span class="num">v{{ status.version }}</span>
        <span class="meta">{{ status.mode }} mode</span>
      </div>

      <dl class="rows">
        <dt class="meta">World</dt>
        <dd class="mono">{{ shortHex(status.world, 8, 6) }}</dd>
        <dt class="meta">Identity</dt>
        <dd class="mono">{{ status.identity ? shortHex(status.identity, 8, 6) : '—' }}</dd>
      </dl>
    </div>
  </section>
</template>

<style scoped>
.stats {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.6rem;
}

.tile {
  display: grid;
  gap: 0.15rem;
  padding: 0.7rem 0.75rem;
  border-radius: var(--radius-sm);
  background: var(--surface-alt);
  text-align: center;
}

.tile .num {
  font-size: 1.15rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--text);
}

.tile .meta {
  text-transform: capitalize;
}

.rows {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 0.35rem 0.75rem;
  margin: 0.25rem 0 0;
}

.rows dd {
  margin: 0;
  text-align: right;
  font-weight: 500;
  word-break: break-all;
}
</style>
