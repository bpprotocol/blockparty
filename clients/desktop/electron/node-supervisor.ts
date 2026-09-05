import { spawn, type ChildProcess } from 'node:child_process'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import type { LifecycleMode, LifecycleState, NodeState } from './bridge'

export interface SupervisorOptions {
  mode: LifecycleMode
  baseUrl: string // http(s)://host:port the API is reached on
  apiAddr: string // host:port the managed node binds
  dataDir: string // node data dir (holds api.token)
  binPath: string // path to the bpnode binary (managed mode)
  nodeMode: string // 'personal' | 'relay'
  // canManage: a bpnode binary for this platform ships with the build. False in
  // a client-only package (#51), where spawning is not an option at all.
  canManage?: boolean
  // attachConfigured: attach mode has an endpoint to attach to. False means the
  // user has not pointed the app at an external node yet.
  attachConfigured?: boolean
  token?: string // bearer token for an external node, when supplied directly
  readyTimeoutMs?: number
  stopTimeoutMs?: number
  maxRestarts?: number
}

// NodeUnavailableError is a typed, user-actionable failure to reach a node —
// as opposed to a spawn ENOENT. `reason` tells the UI which path to offer.
export class NodeUnavailableError extends Error {
  constructor(
    readonly reason: 'no-endpoint' | 'no-bundled-node',
    message: string,
  ) {
    super(message)
    this.name = 'NodeUnavailableError'
  }
}

export interface NodeConnection {
  baseUrl: string
  token: string
}

const LOG_RING = 500

// NodeSupervisor manages the local node process: in managed mode it spawns and
// supervises a bundled bpnode (start/stop/restart, crash recovery, log capture);
// in attach mode it skips spawning and connects to an externally-run daemon.
export class NodeSupervisor {
  private readonly opts: Required<SupervisorOptions>
  private child?: ChildProcess
  private state: NodeState = 'stopped'
  private shuttingDown = false
  private restarts = 0
  private readonly logs: string[] = []

  constructor(opts: SupervisorOptions) {
    this.opts = {
      readyTimeoutMs: 15_000,
      stopTimeoutMs: 8_000,
      maxRestarts: 5,
      // Callers that don't say (tests, older call sites) get the historical
      // behaviour: managed mode implies a binary, attach implies an endpoint.
      canManage: opts.mode === 'managed',
      attachConfigured: true,
      token: '',
      ...opts,
    }
  }

  // start brings the node up and returns how to reach it. In attach mode it does
  // not spawn anything.
  async start(): Promise<NodeConnection> {
    if (this.opts.mode === 'attach') {
      if (!this.opts.attachConfigured) {
        this.state = 'stopped'
        this.log('no node endpoint configured; waiting for one')
        throw new NodeUnavailableError(
          'no-endpoint',
          'no node endpoint configured — connect this app to a node you run',
        )
      }
      this.state = 'running'
      this.log(`attaching to external node at ${this.opts.baseUrl}`)
      return { baseUrl: this.opts.baseUrl, token: this.token() }
    }
    // Never spawn what isn't there: a client-only build has no binary, so this
    // fails with a message the UI can act on instead of a spawn ENOENT (#51).
    if (!this.opts.canManage) {
      this.state = 'stopped'
      throw new NodeUnavailableError(
        'no-bundled-node',
        `no bpnode binary at ${this.opts.binPath} — this build cannot run a node itself`,
      )
    }
    await this.spawnAndWait()
    return { baseUrl: this.opts.baseUrl, token: this.readToken(false) }
  }

  // attachTo points an attach-only app at a different external node, without a
  // restart: the caller rebuilds its client from the returned connection.
  attachTo(external: { baseUrl: string; token?: string; dataDir?: string }): NodeConnection {
    this.opts.baseUrl = external.baseUrl
    this.opts.attachConfigured = true
    this.opts.token = external.token ?? ''
    if (external.dataDir) this.opts.dataDir = external.dataDir
    this.state = 'running'
    this.log(`attaching to external node at ${this.opts.baseUrl}`)
    return { baseUrl: this.opts.baseUrl, token: this.token() }
  }

  // stop terminates a managed node gracefully (SIGTERM, then SIGKILL on timeout).
  async stop(): Promise<void> {
    this.shuttingDown = true
    const child = this.child
    if (!child || child.exitCode !== null || child.signalCode !== null) {
      this.state = 'stopped'
      return
    }
    const exited = new Promise<void>((resolve) => child.once('exit', () => resolve()))
    child.kill('SIGTERM')
    const clean = await Promise.race([
      exited.then(() => true),
      delay(this.opts.stopTimeoutMs).then(() => false),
    ])
    if (!clean) {
      this.log('node did not stop in time; sending SIGKILL')
      child.kill('SIGKILL')
      await exited
    }
    this.state = 'stopped'
    this.log('node stopped')
  }

  getState(): LifecycleState {
    return {
      mode: this.opts.mode,
      state: this.state,
      endpoint: this.opts.attachConfigured ? this.opts.baseUrl : '',
      restarts: this.restarts,
      canManage: this.opts.canManage,
      needsEndpoint: this.opts.mode === 'attach' && !this.opts.attachConfigured,
    }
  }

  recentLogs(): string[] {
    return [...this.logs]
  }

  private async spawnAndWait(): Promise<void> {
    this.spawn()
    await this.waitReady()
    this.state = 'running'
    this.log(`node ready at ${this.opts.baseUrl}`)
  }

  private spawn(): void {
    this.state = 'starting'
    const args = [
      '--mode',
      this.opts.nodeMode,
      '--data-dir',
      this.opts.dataDir,
      '--api-addr',
      this.opts.apiAddr,
      '--log-format',
      'json',
    ]
    this.log(`spawning ${this.opts.binPath} ${args.join(' ')}`)
    const child = spawn(this.opts.binPath, args, { stdio: ['ignore', 'pipe', 'pipe'] })
    child.stdout?.on('data', (d: Buffer) => this.capture(d))
    child.stderr?.on('data', (d: Buffer) => this.capture(d))
    child.on('exit', (code, signal) => this.onExit(code, signal))
    child.on('error', (err) => this.log(`spawn error: ${err.message}`))
    this.child = child
  }

  private onExit(code: number | null, signal: NodeJS.Signals | null): void {
    this.log(`node exited (code=${code ?? '-'} signal=${signal ?? '-'})`)
    if (this.shuttingDown) {
      this.state = 'stopped'
      return
    }
    this.state = 'crashed'
    if (this.restarts >= this.opts.maxRestarts) {
      this.log('max restarts reached; not restarting')
      return
    }
    this.restarts++
    const backoff = Math.min(500 * 2 ** (this.restarts - 1), 10_000)
    this.log(`restarting in ${backoff}ms (attempt ${this.restarts}/${this.opts.maxRestarts})`)
    setTimeout(() => {
      if (!this.shuttingDown) {
        this.spawnAndWait().catch((e: unknown) => this.log(`restart failed: ${String(e)}`))
      }
    }, backoff)
  }

  private async waitReady(): Promise<void> {
    const deadline = Date.now() + this.opts.readyTimeoutMs
    while (Date.now() < deadline) {
      try {
        const res = await fetch(`${this.opts.baseUrl}/healthz`, {
          signal: AbortSignal.timeout(1000),
        })
        if (res.ok) return
      } catch {
        // not up yet
      }
      await delay(150)
    }
    throw new Error(`node did not become ready within ${this.opts.readyTimeoutMs}ms`)
  }

  // token prefers a directly supplied bearer token (an external node the user
  // configured) and otherwise reads api.token from the node's data dir.
  private token(): string {
    return this.opts.token || this.readToken(true)
  }

  private readToken(optional: boolean): string {
    try {
      return readFileSync(path.join(this.opts.dataDir, 'api.token'), 'utf8').trim()
    } catch (e) {
      if (optional) return ''
      throw new Error(`could not read node API token: ${String(e)}`)
    }
  }

  private capture(buf: Buffer): void {
    for (const line of buf.toString('utf8').split('\n')) {
      if (line.trim()) this.log(line.trim())
    }
  }

  private log(line: string): void {
    this.logs.push(line)
    if (this.logs.length > LOG_RING) this.logs.shift()
  }
}

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}
