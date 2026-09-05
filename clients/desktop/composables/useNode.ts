import { computed, ref } from 'vue'
import type {
  BootstrapRequest,
  ExternalNodeConfig,
  LifecycleState,
  NodeStatus,
} from '../electron/bridge'

// Which top-level view the app shows. Derived from the node connection + status,
// implementing the onboarding decision (#43 / #26 Q5): a loaded World goes
// straight to the dashboard, an unconfigured node shows onboarding — unless it
// already holds a keystore, which only needs its passphrase to unlock. A
// client-only build with no node configured yet asks for one first (#51).
export type View =
  'no-bridge' | 'disconnected' | 'external-node' | 'unlock' | 'onboarding' | 'ready'

export function deriveView(input: {
  hasBridge: boolean
  connected: boolean
  status: NodeStatus | null
  // needsEndpoint: this build cannot run a node and has none configured, so
  // there is nothing to connect to yet — ask for an endpoint (#51).
  needsEndpoint?: boolean
}): View {
  if (!input.hasBridge) return 'no-bridge'
  if (!input.connected || !input.status) {
    return input.needsEndpoint ? 'external-node' : 'disconnected'
  }
  if (input.status.worldLoaded) return 'ready'
  // A keystore on disk means the node was started without its passphrase (no
  // BPNODE_KEYSTORE_PASSPHRASE): ask for it. Bootstrapping would fail — the
  // node refuses to overwrite an existing keystore.
  return input.status.keystoreExists ? 'unlock' : 'onboarding'
}

export function useNode() {
  const status = ref<NodeStatus | null>(null)
  const lifecycle = ref<LifecycleState | null>(null)
  const connected = ref(false)
  const error = ref<string | null>(null)
  const busy = ref(false)

  const hasBridge = computed(() => typeof window !== 'undefined' && !!window.bpDesktop)
  const view = computed<View>(() =>
    deriveView({
      hasBridge: hasBridge.value,
      connected: connected.value,
      status: status.value,
      needsEndpoint: lifecycle.value?.needsEndpoint ?? false,
    }),
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

  // unlock opens the keystore the node already holds. Like bootstrap, the
  // secret goes straight to the node — the renderer never stores it — and a
  // success refreshes, flipping the view to the dashboard.
  async function unlock(keystorePassphrase: string): Promise<{ ok: boolean; error?: string }> {
    const bridge = typeof window !== 'undefined' ? window.bpDesktop : undefined
    if (!bridge) return { ok: false, error: 'no bridge' }
    if (!keystorePassphrase) return { ok: false, error: 'Enter your keystore passphrase.' }
    busy.value = true
    const r = await bridge.node.unlockKeystore(keystorePassphrase)
    busy.value = false
    if (r.ok) {
      await refresh()
      return { ok: true }
    }
    return { ok: false, error: r.error }
  }

  // clearKeystore discards the keystore the node holds — the way out when its
  // passphrase is lost. Irreversible, so the node requires an explicit
  // confirmation (#29) that the UI collects before calling this. On success the
  // node reports no keystore, which routes the view to onboarding.
  async function clearKeystore(): Promise<{ ok: boolean; error?: string }> {
    const bridge = typeof window !== 'undefined' ? window.bpDesktop : undefined
    if (!bridge) return { ok: false, error: 'no bridge' }
    busy.value = true
    const r = await bridge.node.clearKeystore(true)
    busy.value = false
    if (r.ok) {
      await refresh()
      return { ok: true }
    }
    return { ok: false, error: r.error }
  }

  // setExternalNode points the app at a node someone else runs. The main
  // process probes it before saving, so a bad address or token comes back as an
  // error here rather than leaving the app disconnected (#51).
  async function setExternalNode(
    cfg: ExternalNodeConfig,
  ): Promise<{ ok: boolean; error?: string }> {
    const bridge = typeof window !== 'undefined' ? window.bpDesktop : undefined
    if (!bridge) return { ok: false, error: 'no bridge' }
    busy.value = true
    const r = await bridge.externalNode.set(cfg)
    busy.value = false
    if (r.ok) {
      await refresh()
      return { ok: true }
    }
    return { ok: false, error: r.error }
  }

  return {
    status,
    lifecycle,
    connected,
    error,
    busy,
    view,
    hasBridge,
    refresh,
    bootstrap,
    unlock,
    clearKeystore,
    setExternalNode,
  }
}
