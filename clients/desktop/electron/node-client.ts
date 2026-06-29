import { createClient, type Client, type Interceptor, type Transport } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-node'
import { NodeService } from './gen/node_pb'
import type {
  BlockSummary,
  BootstrapRequest,
  ConnectionInfo,
  IdentityCard,
  ListFilter,
  NodeApi,
  NodeStatus,
  PostTextRequest,
  PrivateMessage,
  Result,
} from './bridge'

// bearerInterceptor attaches the node API token to every request (the #29 trust
// boundary). The token is held only in the main process.
export function bearerInterceptor(token: string): Interceptor {
  return (next) => (req) => {
    req.header.set('Authorization', `Bearer ${token}`)
    return next(req)
  }
}

// nodeTransport builds an HTTP/1.1 Connect transport to the node API.
export function nodeTransport(baseUrl: string, token: string): Transport {
  return createConnectTransport({
    baseUrl,
    httpVersion: '1.1',
    interceptors: [bearerInterceptor(token)],
  })
}

// NodeClient wraps the generated Connect client, converting wire messages to the
// plain DTOs the renderer consumes and turning errors into Result envelopes.
export class NodeClient implements NodeApi {
  private readonly client: Client<typeof NodeService>

  constructor(transport: Transport) {
    this.client = createClient(NodeService, transport)
  }

  async getStatus(): Promise<Result<NodeStatus>> {
    return wrap(async () => {
      const r = await this.client.getStatus({})
      return {
        version: r.version,
        mode: r.mode,
        worldLoaded: r.worldLoaded,
        world: r.world,
        identity: r.identity,
        blockCount: Number(r.blockCount),
        canAuthor: r.canAuthor,
      }
    })
  }

  async bootstrapWorld(
    req: BootstrapRequest,
  ): Promise<Result<{ world: string; identity: string }>> {
    return wrap(async () => {
      const r = await this.client.bootstrapWorld(req)
      return { world: r.world, identity: r.identity }
    })
  }

  async postText(req: PostTextRequest): Promise<Result<{ id: string }>> {
    return wrap(async () => {
      const r = await this.client.postText({ publicAudience: req.publicAudience, text: req.text })
      return { id: r.id }
    })
  }

  async getBlock(id: string): Promise<Result<import('./bridge').BlockView>> {
    return wrap(async () => {
      const r = await this.client.getBlock({ id })
      return {
        summary: summaryDTO(r.summary),
        decrypted: r.decrypted,
        text: r.text,
      }
    })
  }

  async listBlocks(filter: ListFilter): Promise<Result<BlockSummary[]>> {
    return wrap(async () => {
      const r = await this.client.listBlocks({
        audienceCode: filter.audienceCode ?? '',
        typeCode: filter.typeCode ?? '',
        author: filter.author ?? '',
        from: BigInt(filter.from ?? 0),
        to: BigInt(filter.to ?? 0),
      })
      return r.blocks.map(summaryDTO)
    })
  }

  // streamBlocks yields a BlockSummary for each block accepted onto an audience
  // ("" = all) after subscription, until the signal is aborted (#44).
  async *streamBlocks(audienceCode: string, signal: AbortSignal): AsyncGenerator<BlockSummary> {
    for await (const ev of this.client.subscribeBlocks({ audienceCode }, { signal })) {
      if (ev.summary) yield summaryDTO(ev.summary)
    }
  }

  // --- Connections (#46) ---

  getIdentity(): Promise<Result<IdentityCard>> {
    return wrap(async () => {
      const r = await this.client.getIdentity({})
      return { address: r.address, kyberPub: r.kyberPub, mldsaPub: r.mldsaPub }
    })
  }

  addPeer(card: IdentityCard): Promise<Result<void>> {
    return wrap(async () => {
      await this.client.addPeer(card)
    })
  }

  startConnection(address: string): Promise<Result<{ requestId: string }>> {
    return wrap(async () => {
      const r = await this.client.startConnection({ address })
      return { requestId: r.requestId }
    })
  }

  listConnections(): Promise<Result<ConnectionInfo[]>> {
    return wrap(async () => {
      const r = await this.client.listConnections({})
      return r.connections.map((c) => ({
        peer: c.peer,
        epoch: Number(c.epoch),
        audienceCode: c.audienceCode,
      }))
    })
  }

  rotateConnection(address: string): Promise<Result<void>> {
    return wrap(async () => {
      await this.client.rotateConnection({ address })
    })
  }

  closeConnection(address: string): Promise<Result<void>> {
    return wrap(async () => {
      await this.client.closeConnection({ address })
    })
  }

  sendPrivateText(address: string, text: string): Promise<Result<{ id: string }>> {
    return wrap(async () => {
      const r = await this.client.sendPrivateText({ address, text })
      return { id: r.id }
    })
  }

  connectionMessages(address: string): Promise<Result<PrivateMessage[]>> {
    return wrap(async () => {
      const r = await this.client.listConnectionMessages({ address })
      return r.messages.map((m) => ({
        author: m.author,
        text: m.text,
        timestamp: Number(m.timestamp),
      }))
    })
  }
}

function summaryDTO(s: { [k: string]: unknown } | undefined): BlockSummary {
  return {
    id: str(s?.id),
    audienceCode: str(s?.audienceCode),
    typeCode: str(s?.typeCode),
    author: str(s?.author),
    timestamp: Number((s?.timestamp as bigint) ?? 0n),
    receivedAt: Number((s?.receivedAt as bigint) ?? 0n),
  }
}

function str(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

async function wrap<T>(fn: () => Promise<T>): Promise<Result<T>> {
  try {
    return { ok: true, value: await fn() }
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) }
  }
}
