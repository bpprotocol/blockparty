import { describe, it, expect } from "vitest";
import { PROTOCOL_VERSION, hello } from "./index.js";

describe("@blockparty/sdk", () => {
  it("exposes the protocol version", () => {
    expect(PROTOCOL_VERSION).toBe("0.1.0");
  });

  it("greets with the version", () => {
    expect(hello()).toContain(PROTOCOL_VERSION);
  });
});
