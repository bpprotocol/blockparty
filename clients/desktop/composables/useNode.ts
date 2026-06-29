import { computed, ref } from 'vue'
import type { BootstrapRequest, LifecycleState, NodeStatus } from '../electron/bridge'

// Which top-level view the app shows. Derived from the node connection + status,
// implementing the onboarding decision (#43 / #26 Q5): a loaded World goes
// straight to the dashboard, an unconfigured node shows onboarding.
export type View = 'no-bridge' | 'disconnected' | 'onboarding' | 'ready'

export function deriveView(input: {
  hasBridge: boolean
  connected: boolean
  status: NodeStatus | null
}): View {
  if (!input.hasBridge) return 'no-bridge'
  if (!input.connected || !input.status) return 'disconnected'
  return input.status.worldLoaded ? 'ready' : 'onboarding'
}

export function useNode() {
  const status = ref<NodeStatus | null>(null)
  const lifecycle = ref<LifecycleState | null>(null)
  const connected = ref(false)
  const error = ref<string | null>(null)
  const busy = ref(false)

  const hasBridge = computed(() => typeof window !== 'undefined' && !!window.bpDesktop)
  const view = computed<View>(() =>
    deriveView({ hasBridge: hasBridge.value, connected: connected.value, status: status.value }),
  )

  async function refresh(): Promise<void> {
    const bridge = typeof window !== 'undefined' ? window.bpDesktop : undefined
    if (!bridge) {
      connected.value = false
      error.value = 'Preload bridge unavailable (running outside Electron)'
      return
    }
    lifecycle.value = await bridge.lifecycle.getState()
    const r = await bridge.node.getStatus()
    if (r.ok) {
      status.value = r.value
      connected.value = true
      error.value = null
    } else {
      connected.value = false
      error.value = r.error
    }
  }

  // bootstrap configures the node's World from the user's secrets (the node
  // holds the keys; the renderer never persists them). On success it refreshes,
  // which flips the view to the dashboard.
  async function bootstrap(req: BootstrapRequest): Promise<{ ok: boolean; error?: string }> {
    const bridge = typeof window !== 'undefined' ? window.bpDesktop : undefined
    if (!bridge) return { ok: false, error: 'no bridge' }
    busy.value = true
    const r = await bridge.node.bootstrapWorld(req)
    busy.value = false
    if (r.ok) {
      await refresh()
      return { ok: true }
    }
    return { ok: false, error: r.error }
  }

  return { status, lifecycle, connected, error, busy, view, hasBridge, refresh, bootstrap }
}
