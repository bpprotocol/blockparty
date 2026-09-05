import { computed, onMounted, onUnmounted, ref } from 'vue'

// The window is frameless (electron/main.ts), so the renderer draws the title
// bar. macOS keeps its native traffic lights, so it must not draw buttons of
// its own — and its header needs room for them.
export function needsCustomControls(platform: string | undefined): boolean {
  if (!platform) return false // outside Electron: the browser has its own chrome
  return platform !== 'darwin'
}

export function reservesTrafficLightSpace(platform: string | undefined): boolean {
  return platform === 'darwin'
}

export function useWindow() {
  const maximized = ref(false)
  let unsub: (() => void) | undefined

  function bridge() {
    return typeof window !== 'undefined' ? window.bpDesktop : undefined
  }

  const platform = computed(() => bridge()?.platform)
  const showControls = computed(() => needsCustomControls(platform.value))
  const macInset = computed(() => reservesTrafficLightSpace(platform.value))

  async function minimize(): Promise<void> {
    await bridge()?.window.minimize()
  }

  async function toggleMaximize(): Promise<void> {
    const next = await bridge()?.window.toggleMaximize()
    if (typeof next === 'boolean') maximized.value = next
  }

  async function close(): Promise<void> {
    await bridge()?.window.close()
  }

  onMounted(() => {
    const w = bridge()?.window
    if (!w) return
    void w.isMaximized().then((m) => (maximized.value = m))
    // Keep the button in step with changes made outside the app — a
    // double-click on the header, a keyboard shortcut, the window manager.
    unsub = w.onMaximizeChange((m) => (maximized.value = m))
  })
  onUnmounted(() => unsub?.())

  return { maximized, showControls, macInset, minimize, toggleMaximize, close }
}
