// Package audiences implements BlockParty's audience model (issue #6),
// following protocol/specs/audiences.md (and the public-audience secret in
// encryption.md, the inbox audience in connections.md).
//
// Two kinds of audiences are derived here:
//
//   - Public audiences (bpprotocol.org/v1/audience/public-N, N in 1..16
//     reserved): world-scoped, confidential to the World but open within it.
//     A World member derives both the code and the 32-byte secret; an outsider
//     without the World seed can derive neither.
//   - Inbox audiences (bpprotocol.org/v1/audience/inbox/<address>): a
//     per-identity rendezvous any World member can derive. The inbox is a
//     rendezvous, not a confidential channel, so it has a code but no secret.
//
// Private (connection) audiences are derived from a handshake and live in the
// connections package (#7).
//
// An audience's code is its addressing label (stored in a block's
// audience_code); its secret feeds the AEAD layer (#5). Registry provides a
// local code -> audience map so a client can find the secret for a received
// block.
package audiences
