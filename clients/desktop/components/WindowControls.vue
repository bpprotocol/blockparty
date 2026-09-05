<script setup lang="ts">
// Minimize / maximize / close for the frameless window. Rendered only where
// the app draws its own controls (Windows, Linux) — macOS has traffic lights.
defineProps<{ maximized: boolean }>()
const emit = defineEmits<{ minimize: []; toggle: []; close: [] }>()
</script>

<template>
  <div class="controls">
    <button class="ctl" aria-label="Minimize" title="Minimize" @click="emit('minimize')">
      <svg viewBox="0 0 10 10" aria-hidden="true"><path d="M0 5h10" /></svg>
    </button>
    <button
      class="ctl"
      :aria-label="maximized ? 'Restore' : 'Maximize'"
      :title="maximized ? 'Restore' : 'Maximize'"
      @click="emit('toggle')"
    >
      <svg v-if="maximized" viewBox="0 0 10 10" aria-hidden="true">
        <path d="M2.5 2.5V0.5h7v7h-2M0.5 2.5h7v7h-7z" />
      </svg>
      <svg v-else viewBox="0 0 10 10" aria-hidden="true">
        <path d="M0.5 0.5h9v9h-9z" />
      </svg>
    </button>
    <button class="ctl is-close" aria-label="Close" title="Close" @click="emit('close')">
      <svg viewBox="0 0 10 10" aria-hidden="true"><path d="M0.5 0.5l9 9M9.5 0.5l-9 9" /></svg>
    </button>
  </div>
</template>

<style scoped>
.controls {
  display: flex;
  align-items: stretch;
  align-self: stretch;
  /* Buttons must stay clickable inside the header's drag region. */
  -webkit-app-region: no-drag;
}

.ctl {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s;
}

.ctl svg {
  width: 10px;
  height: 10px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.1;
}

.ctl:hover {
  background: var(--surface-alt);
  color: var(--text);
}

.ctl.is-close:hover {
  background: var(--danger);
  color: #fff;
}
</style>
