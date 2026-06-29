<script setup lang="ts">
import { ref } from 'vue'
import { PUBLIC_AUDIENCE_MAX, useCompose } from '../composables/useCompose'

const emit = defineEmits<{ posted: [id: string] }>()

const { busy, error, post } = useCompose()
const text = ref('')
const audience = ref(1)
const audiences = Array.from({ length: PUBLIC_AUDIENCE_MAX }, (_, i) => i + 1)

async function submit(): Promise<void> {
  const id = await post(audience.value, text.value)
  if (id) {
    text.value = ''
    emit('posted', id)
  }
}
</script>

<template>
  <section class="compose">
    <form @submit.prevent="submit">
      <textarea
        v-model="text"
        rows="3"
        placeholder="Share something…"
        :disabled="busy"
        @keydown.meta.enter="submit"
        @keydown.ctrl.enter="submit"
      />
      <div class="row">
        <label>
          to
          <select v-model.number="audience" :disabled="busy">
            <option v-for="n in audiences" :key="n" :value="n">public-{{ n }}</option>
          </select>
        </label>
        <button type="submit" class="primary" :disabled="busy">
          {{ busy ? 'Posting…' : 'Post' }}
        </button>
      </div>
      <p v-if="error" class="err">{{ error }}</p>
    </form>
  </section>
</template>

<style scoped>
.compose {
  margin-top: 1.5rem;
}
form {
  display: grid;
  gap: 0.5rem;
}
textarea {
  font: inherit;
  padding: 0.6rem;
  border: 1px solid color-mix(in srgb, currentColor 20%, transparent);
  border-radius: 8px;
  background: transparent;
  color: inherit;
  resize: vertical;
}
.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
label {
  opacity: 0.7;
  font-size: 0.9rem;
}
select {
  font: inherit;
  margin-left: 0.3rem;
}
.primary {
  background: #1a7f37;
  color: white;
  border: none;
  padding: 0.45rem 1.1rem;
  border-radius: 6px;
  cursor: pointer;
}
.primary:disabled {
  opacity: 0.6;
}
.err {
  color: #cf222e;
  font-size: 0.9rem;
  margin: 0;
}
</style>
