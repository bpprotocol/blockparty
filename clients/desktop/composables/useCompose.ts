import { ref } from 'vue'

export const PUBLIC_AUDIENCE_MAX = 16

// validatePost checks a compose intent before it is sent. Pure, so it is
// unit-tested. Returns an error message, or null when valid.
export function validatePost(audience: number, text: string): string | null {
  if (!text.trim()) return 'Write something to post.'
  if (!Number.isInteger(audience) || audience < 1 || audience > PUBLIC_AUDIENCE_MAX) {
    return `Choose a public audience (1–${PUBLIC_AUDIENCE_MAX}).`
  }
  return null
}

export function useCompose() {
  const busy = ref(false)
  const error = ref<string | null>(null)

  // post submits the compose intent to the node, which signs (world_sig +
  // author_sig) and encrypts it — the client holds no keys. content.post is not
  // a dangerous op, so it needs no confirmation gate (#29). Returns the new block
  // id, or null on failure (with error set).
  async function post(audience: number, text: string): Promise<string | null> {
    error.value = validatePost(audience, text)
    if (error.value) return null

    const node = typeof window !== 'undefined' ? window.bpDesktop?.node : undefined
    if (!node) {
      error.value = 'Not connected to a node.'
      return null
    }
    busy.value = true
    const r = await node.postText({ publicAudience: audience, text: text.trim() })
    busy.value = false
    if (!r.ok) {
      error.value = r.error
      return null
    }
    return r.value.id
  }

  return { busy, error, post }
}
