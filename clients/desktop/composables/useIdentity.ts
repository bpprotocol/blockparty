import { ref } from 'vue'

export function useIdentity() {
  const busy = ref(false)
  const error = ref<string | null>(null)
  const notice = ref<string | null>(null)

  function api() {
    return typeof window !== 'undefined' ? window.bpDesktop?.identity : undefined
  }

  // rotate replaces the identity passphrase. confirm must be the user's explicit
  // confirmation — the node rejects an unconfirmed call (the #29 gate).
  async function rotate(newPassphrase: string, confirm: boolean): Promise<string | null> {
    error.value = null
    notice.value = null
    if (!newPassphrase) {
      error.value = 'Enter a new identity passphrase.'
      return null
    }
    const id = api()
    if (!id) {
      error.value = 'Not connected to a node.'
      return null
    }
    busy.value = true
    const r = await id.rotate(newPassphrase, confirm)
    busy.value = false
    if (!r.ok) {
      error.value = r.error
      return null
    }
    notice.value = 'Identity rotated.'
    return r.value.identity
  }

  // burn publishes an identity.burn, revoking the identity. Irreversible.
  async function burn(reason: string, confirm: boolean): Promise<string | null> {
    error.value = null
    notice.value = null
    const id = api()
    if (!id) {
      error.value = 'Not connected to a node.'
      return null
    }
    busy.value = true
    const r = await id.burn(reason, confirm)
    busy.value = false
    if (!r.ok) {
      error.value = r.error
      return null
    }
    notice.value = 'Identity burned.'
    return r.value.blockId
  }

  return { busy, error, notice, rotate, burn }
}
