import { parseArgs } from "node:util";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  PROTOCOL_VERSION,
  openWorld,
  openIdentity,
  publicAudience,
  Resolver,
  verifyBlock,
  buildPing,
  handlePing,
  parsePingResponse,
  startRequest,
  acceptRequest,
  peerFromIdentity,
  idHex,
} from "@blockparty/sdk";
import { FileStore } from "./store.js";
import { buildPost, openPost } from "./post.js";

/** Banner identifying the CLI and the protocol version it targets. */
export function banner(): string {
  return `bp — BlockParty reference CLI (protocol v${PROTOCOL_VERSION})`;
}

const USAGE = `bp - BlockParty reference CLI

Usage:
  bp address --world SEED --pass PHRASE
  bp send    --net DIR --world SEED --pass PHRASE --audience N --text MSG
  bp inbox   --net DIR --world SEED --audience N
  bp ping    --node NAME
  bp demo    [--net DIR]
`;

type Out = (line: string) => void;

function opt(args: string[], options: Record<string, { type: "string" }>): Record<string, string> {
  const { values } = parseArgs({ args, options, allowPositionals: false });
  return values as Record<string, string>;
}

function cmdAddress(args: string[], out: Out): void {
  const v = opt(args, { world: { type: "string" }, pass: { type: "string" } });
  const id = openIdentity(openWorld(v.world ?? ""), v.pass ?? "");
  out(`address:   ${id.address}`);
  out(`kyber-pub: ${Buffer.from(id.kyber.publicKey).toString("base64")}`);
}

function cmdSend(args: string[], out: Out): void {
  const v = opt(args, {
    net: { type: "string" },
    world: { type: "string" },
    pass: { type: "string" },
    audience: { type: "string" },
    text: { type: "string" },
  });
  const world = openWorld(v.world ?? "");
  const id = openIdentity(world, v.pass ?? "");
  const aud = publicAudience(world, Number(v.audience ?? "1"));
  const block = buildPost(world, id, aud.code, aud.secret!, nowSeconds(), v.text ?? "");
  const path = new FileStore(v.net ?? "").write(block);
  out(`sent block ${idHex(block)} to ${path}`);
}

function cmdInbox(args: string[], out: Out): void {
  const v = opt(args, {
    net: { type: "string" },
    world: { type: "string" },
    audience: { type: "string" },
  });
  const world = openWorld(v.world ?? "");
  const aud = publicAudience(world, Number(v.audience ?? "1"));
  const resolver = new Resolver(world);
  let count = 0;
  for (const block of new FileStore(v.net ?? "").readAll()) {
    try {
      const text = openPost(resolver, block, aud.secret!);
      if (text !== undefined) {
        count += 1;
        out(`[${idHex(block).slice(0, 12)}] ${text}`);
      }
    } catch {
      out(`[${idHex(block).slice(0, 12)}] undecryptable`);
    }
  }
  out(`(${count} post(s) on public-${v.audience ?? "1"})`);
}

function cmdPing(args: string[], out: Out): void {
  const v = opt(args, { node: { type: "string" } });
  const sent = nowSeconds();
  const req = buildPing("ping-1", sent);
  const res = parsePingResponse(handlePing(req, nowSeconds(), nowSeconds(), v.node ?? "bp-cli"));
  out(
    `pong from "${res.node}": sent=${res.timestamp} received=${res.receivedAt} server=${res.serverTime}`,
  );
}

/**
 * Run the full two-party connection handshake and a private encrypted exchange
 * over the filesystem transport, in one process, printing each step.
 */
function cmdDemo(args: string[], out: Out): void {
  const v = opt(args, { net: { type: "string" } });
  const netDir = v.net ?? mkdtempSync(join(tmpdir(), "bp-demo-net-"));
  const store = new FileStore(netDir);

  const world = openWorld("bpprotocol.org/v1/demo-world");
  const alice = openIdentity(world, "alice-demo");
  const bob = openIdentity(world, "bob-demo");
  out(`world opened; alice=${alice.address} bob=${bob.address}`);

  const { pending, request } = startRequest(alice, peerFromIdentity(bob));
  out("1. alice -> connect.request");
  const { response, connection: bobConn } = acceptRequest(
    bob,
    peerFromIdentity(alice),
    request,
    "req-id",
  );
  out("2. bob   -> connect.response");
  const aliceConn = pending.complete(response);
  if (idHex2(aliceConn.audienceCode()) !== idHex2(bobConn.audienceCode())) {
    throw new Error("handshake disagreement");
  }
  out(`3. both derive private audience ${idHex2(aliceConn.audienceCode())}`);

  const msg = "the only winning move is to share the keys";
  const block = buildPost(
    world,
    alice,
    aliceConn.audienceCode(),
    aliceConn.audienceSecret(),
    nowSeconds(),
    msg,
  );
  const path = store.write(block);
  out(`4. alice writes encrypted block -> ${path}`);

  const resolver = new Resolver(world);
  for (const blk of store.readAll()) {
    if (!verifyBlock(blk, world, alice.mldsa.publicKey)) throw new Error("verify failed");
    const text = openPost(resolver, blk, bobConn.audienceSecret());
    if (text !== undefined) {
      out(`5. bob verifies + decrypts: "${text}"`);
      if (text !== msg) throw new Error("decrypted text mismatch");
    }
  }
  out("demo OK");
}

function nowSeconds(): number {
  return Math.floor(Date.now() / 1000);
}

function idHex2(code: Uint8Array): string {
  return Buffer.from(code).toString("hex");
}

/** Dispatch a CLI invocation. Returns the process exit code. */
export function run(argv: string[], out: Out = console.log, err: Out = console.error): number {
  const [command, ...rest] = argv;
  try {
    switch (command) {
      case "address":
        cmdAddress(rest, out);
        return 0;
      case "send":
        cmdSend(rest, out);
        return 0;
      case "inbox":
        cmdInbox(rest, out);
        return 0;
      case "ping":
        cmdPing(rest, out);
        return 0;
      case "demo":
        cmdDemo(rest, out);
        return 0;
      case "-h":
      case "--help":
      case "help":
      case undefined:
        err(USAGE);
        return command ? 0 : 2;
      default:
        err(`bp: unknown command ${JSON.stringify(command)}`);
        err(USAGE);
        return 2;
    }
  } catch (e) {
    err(`bp: ${e instanceof Error ? e.message : String(e)}`);
    return 1;
  }
}
