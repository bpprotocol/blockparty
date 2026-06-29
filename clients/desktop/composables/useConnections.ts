import { ref } from 'vue'
import type { ConnectionInfo, IdentityCard, PrivateMessage } from '../electron/bridge'

// A connection card is shared as a single dot-delimited string of the three hex
// fields. format/parse are pure, so they are unit-tested.
export function formatCard(card: IdentityCard): string {
  return `${card.address}.${card.kyberPub}.${card.mldsaPub}`
}

export function parseCard(text: string): IdentityCard | null {
  const parts = text.trim().split('.')
  if (parts.length !== 3 || parts.some((p) => !p)) return null
  return { address: parts[0], kyberPub: parts[1], mldsaPub: parts[2] }
}

export function useConnections() {
  const myCard = ref<IdentityCard | null>(null)
  const list = ref<ConnectionInfo[]>([])
  const error = ref<string | null>(null)
  const busy = ref(false)

  function api() {
    return typeof window !== 'undefined' ? window.bpDesktop?.connections : undefined
  }

  async function loadCard(): Promise<void> {
    const r = await api()?.getIdentity()
    if (r?.ok) myCard.value = r.value
  }

  async function refresh(): Promise<void> {
    const r = await api()?.list()
    if (r?.ok) list.value = r.value
    else if (r) error.value = r.error
  }

  // addAndConnect parses a pasted card, registers the peer, and initiates the
  // handshake. Returns the peer address on success.
  async function addAndConnect(cardText: string): Promise<string | null> {
    error.value = null
    const card = parseCard(cardText)
    if (!card) {
      error.value = 'That does not look like a connection card.'
      return null
    }
    const conn = api()
    if (!conn) {
      error.value = 'Not connected to a node.'
      return null
    }
    busy.value = true
    const added = await conn.addPeer(card)
    if (!added.ok) {
      busy.value = false
      error.value = added.error
      return null
    }
    const started = await conn.start(card.address)
    busy.value = false
    if (!started.ok) {
      error.value = started.error
      return null
    }
    await refresh()
    return card.address
  }

  async function rotate(address: string): Promise<void> {
    await api()?.rotate(address)
    await refresh()
  }

  async function close(address: string): Promise<void> {
    await api()?.close(address)
    await refresh()
  }

  async function send(address: string, text: string): Promise<boolean> {
    const r = await api()?.sendText(address, text)
    return !!r?.ok
  }

  async function messages(address: string): Promise<PrivateMessage[]> {
    const r = await api()?.messages(address)
    return r?.ok ? r.value : []
  }

  return {
    myCard,
    list,
    error,
    busy,
    loadCard,
    refresh,
    addAndConnect,
    rotate,
    close,
    send,
    messages,
  }
}
