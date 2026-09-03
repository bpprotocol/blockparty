<script setup lang="ts">
import { computed } from 'vue'
import { avatarGradient, initials } from '../composables/useFormat'

// A generated avatar: identities have no profile picture, so we derive a stable
// colour and monogram from the address hex. Same peer, same face.
const props = withDefaults(defineProps<{ seed: string; size?: number }>(), { size: 42 })

const style = computed(() => ({
  height: `${props.size}px`,
  width: `${props.size}px`,
  backgroundImage: avatarGradient(props.seed),
  fontSize: `${Math.max(10, Math.round(props.size * 0.34))}px`,
}))
</script>

<template>
  <span class="avatar" :style="style" :title="seed || undefined">{{ initials(seed) }}</span>
</template>

<style scoped>
.avatar {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: #fff;
  font-weight: 700;
  letter-spacing: 0.03em;
  text-transform: uppercase;
  user-select: none;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.18);
}
</style>
