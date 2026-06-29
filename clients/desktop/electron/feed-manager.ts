import type { BlockSummary, Result } from './bridge'
import type { NodeClient } from './node-client'

// FeedManager owns the live block stream in the main process. It opens a server
// stream to the node for a single audience and forwards each event to the
// renderer via the provided sink (webContents.send). Subscribing again replaces
// the previous stream.
export class FeedManager {
  private controller?: AbortController

  constructor(
    private readonly client: NodeClient,
    private readonly emit: (summary: BlockSummary) => void,
  ) {}

  subscribe(audienceCode: string): Result<void> {
    this.unsubscribe()
    const controller = new AbortController()
    this.controller = controller
    void this.pump(audienceCode, controller.signal)
    return { ok: true, value: undefined }
  }

  unsubscribe(): void {
    this.controller?.abort()
    this.controller = undefined
  }

  private async pump(audienceCode: string, signal: AbortSignal): Promise<void> {
    try {
      for await (const summary of this.client.streamBlocks(audienceCode, signal)) {
        if (signal.aborted) return
        this.emit(summary)
      }
    } catch {
      // Stream ended (aborted, or node restarting). The renderer re-subscribes
      // when it reconnects; nothing to do here.
    }
  }
}
