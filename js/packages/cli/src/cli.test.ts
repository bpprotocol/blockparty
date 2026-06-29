import { describe, it, expect } from "vitest";
import { banner } from "./cli.js";

describe("@blockparty/cli", () => {
  it("consumes @blockparty/sdk (package boundary works)", () => {
    // banner() pulls PROTOCOL_VERSION from the SDK; if the boundary were broken
    // this import/build would fail.
    expect(banner()).toContain("0.1.0");
    expect(banner()).toContain("BlockParty reference CLI");
  });
});
