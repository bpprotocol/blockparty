<script setup lang="ts">
import { computed, ref } from 'vue'
import { PUBLIC_AUDIENCE_MAX, useCompose } from '../composables/useCompose'
import UserAvatar from './UserAvatar.vue'
import AppIcon from './AppIcon.vue'

const props = withDefaults(defineProps<{ author?: string }>(), { author: '' })
const emit = defineEmits<{ posted: [id: string] }>()

const { busy, error, post } = useCompose()
const text = ref('')
const audience = ref(1)
const audiences = Array.from({ length: PUBLIC_AUDIENCE_MAX }, (_, i) => i + 1)

const canPost = computed(() => !busy.value && text.value.trim().length > 0)

async function submit(): Promise<void> {
  if (!canPost.value) return
  const id = await post(audience.value, text.value)
  if (id) {
    text.value = ''
    emit('posted', id)
  }
}
</script>

<template>
  <section class="card compose">
    <form @submit.prevent="submit">
      <div class="compose-row">
        <UserAvatar :seed="props.author" :size="42" />
        <textarea
          v-model="text"
          class="textarea"
          rows="3"
          placeholder="What's on your mind?"
          :disabled="busy"
          @keydown.meta.enter="submit"
          @keydown.ctrl.enter="submit"
        />
      </div>

      <div class="compose-options">
        <label class="audience">
          <AppIcon name="globe" :size="14" />
          <span>Post to</span>
          <select v-model.number="audience" class="audience-select" :disabled="busy">
            <option v-for="n in audiences" :key="n" :value="n">public-{{ n }}</option>
          </select>
        </label>

        <span class="hint meta">⌘/Ctrl + ↵</span>

        <button type="submit" class="btn is-primary is-pill" :disabled="!canPost">
          <AppIcon name="send" :size="14" />
          {{ busy ? 'Posting…' : 'Post' }}
        </button>
      </div>

      <p v-if="error" class="err compose-err">{{ error }}</p>
    </form>
  </section>
</template>

<style scoped>
.compose-row {
  display: flex;
  align-items: flex-start;
  gap: 0.85rem;
  padding: 1rem;
}

.compose-row .textarea {
  padding: 0.5rem 0;
  background: transparent;
  border: none;
  font-size: 0.95rem;
}

.compose-row .textarea:focus {
  box-shadow: none;
}

.compose-options {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 0.75rem;
  border-top: 1px solid var(--border);
  background: var(--surface-alt);
  border-radius: 0 0 var(--radius) var(--radius);
}

.audience {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.3rem 0.7rem;
  border-radius: var(--radius-pill);
  background: var(--surface);
  border: 1px solid var(--border);
  font-size: 0.8rem;
  color: var(--text-muted);
}

.audience-select {
  font: inherit;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--primary);
  background: transparent;
  border: none;
  cursor: pointer;
}

.audience-select:focus {
  outline: none;
}

.hint {
  margin-left: auto;
}

.compose-err {
  padding: 0 1rem 0.85rem;
}
</style>
