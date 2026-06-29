// Renderer-side type for the preload bridge. Kept in sync with electron/preload.ts.
export {}

declare global {
  interface Window {
    bpDesktop?: {
      versions: () => { electron: string; chrome: string; node: string }
    }
  }
}
