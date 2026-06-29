// Renderer-side type for the preload bridge (window.bpDesktop). The contract is
// defined once in electron/bridge.ts; this only attaches it to Window.
import type { BpDesktop } from '../electron/bridge'

export {}

declare global {
  interface Window {
    bpDesktop?: BpDesktop
  }
}
