import { describe, it, expect } from "vitest";
import { mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { run, banner } from "./cli.js";

// Collect output lines from a run() invocation.
function capture(argv: string[]): { code: number; out: string[]; err: string[] } {
  const out: string[] = [];
  const err: string[] = [];
  const code = run(
    argv,
    (l) => out.push(l),
    (l) => err.push(l),
  );
  return { code, out, err };
}

describe("cli", () => {
  it("banner names the protocol version (boundary works)", () => {
    expect(banner()).toContain("0.1.0");
  });

  it("derives an address", () => {
    const { code, out } = capture(["address", "--world", "w", "--pass", "p"]);
    expect(code).toBe(0);
    expect(out.join("\n")).toContain("address:");
  });

  it("two instances exchange an encrypted block via a public audience", () => {
    const net = mkdtempSync(join(tmpdir(), "bp-cli-test-"));
    const send = capture([
      "send",
      "--net",
      net,
      "--world",
      "festival",
      "--pass",
      "alice",
      "--audience",
      "1",
      "--text",
      "anyone out there?",
    ]);
    expect(send.code).toBe(0);

    const inbox = capture(["inbox", "--net", net, "--world", "festival", "--audience", "1"]);
    expect(inbox.code).toBe(0);
    expect(inbox.out.join("\n")).toContain("anyone out there?");
  });

  it("runs the full handshake + private exchange demo", () => {
    const { code, out } = capture(["demo", "--net", mkdtempSync(join(tmpdir(), "bp-cli-demo-"))]);
    expect(code).toBe(0);
    expect(out.join("\n")).toContain("demo OK");
    expect(out.join("\n")).toContain("bob verifies + decrypts");
  });

  it("answers a local ping", () => {
    const { code, out } = capture(["ping", "--node", "test-node"]);
    expect(code).toBe(0);
    expect(out.join("\n")).toContain("pong from");
  });

  it("rejects an unknown command", () => {
    expect(capture(["bogus"]).code).toBe(2);
  });
});
