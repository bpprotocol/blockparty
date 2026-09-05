// Builds the bpnode binary that ships inside a packaged desktop app (#48).
//
// electron-builder calls this from its beforePack hook (scripts/before-pack.mjs)
// with the platform and architecture it is about to package, so every artifact
// carries a node built for that target. It also runs standalone —
// `pnpm build:node` — to populate resources/ for a local `pnpm pack`.

import { execFileSync } from 'node:child_process'
import { mkdirSync, rmSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
export const DESKTOP_DIR = path.resolve(here, '..')
export const NODE_DIR = path.resolve(DESKTOP_DIR, '../../node')
export const RESOURCES_DIR = path.join(DESKTOP_DIR, 'resources')

// Electron platform/arch names → the Go toolchain's GOOS/GOARCH, plus the
// binary name the supervisor looks for (node-config.ts uses the same rule).
const GOOS = { darwin: 'darwin', win32: 'windows', linux: 'linux' }
const GOARCH = { x64: 'amd64', arm64: 'arm64', armv7l: 'arm', ia32: '386' }

export function goEnvFor(platform, arch) {
  const goos = GOOS[platform]
  const goarch = GOARCH[arch]
  if (!goos) throw new Error(`unsupported platform for bpnode: ${platform}`)
  if (!goarch) throw new Error(`unsupported architecture for bpnode: ${arch}`)
  return { GOOS: goos, GOARCH: goarch, binName: goos === 'windows' ? 'bpnode.exe' : 'bpnode' }
}

// hostTarget is what `pnpm build:node` builds when nothing is specified.
export function hostTarget() {
  return { platform: process.platform, arch: process.arch === 'x64' ? 'x64' : process.arch }
}

// buildNode compiles bpnode into resources/. The resources dir is emptied first
// so a build for one platform never leaves another platform's binary behind for
// electron-builder to copy.
export function buildNode({ platform, arch, outDir = RESOURCES_DIR, nodeDir = NODE_DIR } = {}) {
  const target = { ...hostTarget(), ...(platform ? { platform } : {}), ...(arch ? { arch } : {}) }
  const { GOOS: goos, GOARCH: goarch, binName } = goEnvFor(target.platform, target.arch)
  const outPath = path.join(outDir, binName)

  rmSync(outDir, { recursive: true, force: true })
  mkdirSync(outDir, { recursive: true })

  console.log(`[build-node] ${goos}/${goarch} → ${path.relative(DESKTOP_DIR, outPath)}`)
  execFileSync('go', ['build', '-trimpath', '-ldflags', '-s -w', '-o', outPath, './cmd/bpnode'], {
    cwd: nodeDir,
    stdio: 'inherit',
    // The node is pure Go, so CGO off keeps cross-compilation working and the
    // binary self-contained inside the app bundle.
    env: { ...process.env, GOOS: goos, GOARCH: goarch, CGO_ENABLED: '0' },
  })
  return outPath
}

// Run directly: build for this machine.
if (process.argv[1] && import.meta.url === `file://${process.argv[1]}`) {
  buildNode()
}
