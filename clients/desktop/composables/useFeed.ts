import { ref } from 'vue'
import type { BlockSummary } from '../electron/bridge'

export interface FeedItem {
  id: string
  author: string
  timestamp: number
  text: string
}

// mergeItem inserts an item keeping the list de-duplicated by id and ordered
// newest-first. Pure, so it is unit-tested.
export function mergeItem(items: FeedItem[], item: FeedItem): FeedItem[] {
  const without = items.filter((i) => i.id !== item.id)
  without.push(item)
  without.sort((a, b) => b.timestamp - a.timestamp || b.id.localeCompare(a.id))
  return without
}

export function useFeed() {
  const items = ref<FeedItem[]>([])
  const error = ref<string | null>(null)
  const loading = ref(false)
  let unsub: (() => void) | undefined

  function bridge() {
    return typeof window !== 'undefined' ? window.bpDesktop : undefined
  }

  // toItem fetches a block's text; returns null if it isn't a readable post.
  async function toItem(summary: BlockSummary): Promise<FeedItem | null> {
    const node = bridge()?.node
    if (!node) return null
    const g = await node.getBlock(summary.id)
    if (!g.ok || !g.value.decrypted || !g.value.text) return null
    return {
      id: summary.id,
      author: summary.author,
      timestamp: summary.timestamp,
      text: g.value.text,
    }
  }

  // load fetches the backlog (all blocks the node holds) and renders the posts.
  async function load(): Promise<void> {
    const node = bridge()?.node
    if (!node) return
    loading.value = true
    error.value = null
    const r = await node.listBlocks({ from: 1 })
    if (!r.ok) {
      error.value = r.error
      loading.value = false
      return
    }
    let next: FeedItem[] = []
    for (const s of r.value) {
      const item = await toItem(s)
      if (item) next = mergeItem(next, item)
    }
    items.value = next
    loading.value = false
  }

  // start subscribes to live block events and prepends new posts as they arrive.
  async function start(): Promise<void> {
    const b = bridge()
    if (!b) return
    await b.feed.subscribe('')
    unsub = b.feed.onEvent((summary) => {
      void toItem(summary).then((item) => {
        if (item) items.value = mergeItem(items.value, item)
      })
    })
  }

  function stop(): void {
    unsub?.()
    unsub = undefined
    void bridge()?.feed.unsubscribe()
  }

  return { items, error, loading, load, start, stop }
}
