import { PROTOCOL_VERSION } from "@blockparty/sdk";

/**
 * Banner for the reference CLI. It reads PROTOCOL_VERSION from @blockparty/sdk,
 * proving the package boundary works: the CLI is a consumer of the reusable
 * library, not a place where protocol logic lives.
 */
export function banner(): string {
  return `bp — BlockParty reference CLI (protocol v${PROTOCOL_VERSION})`;
}
