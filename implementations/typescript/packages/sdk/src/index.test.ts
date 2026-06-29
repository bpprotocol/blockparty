import { describe, it, expect } from "vitest";
import { PROTOCOL_VERSION, keccak256 } from "./index.js";

describe("@blockparty/sdk", () => {
  it("exposes the protocol version", () => {
    expect(PROTOCOL_VERSION).toBe("0.1.0");
  });

  it("re-exports the crypto module", () => {
    expect(typeof keccak256).toBe("function");
  });
});
