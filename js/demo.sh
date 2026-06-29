#!/usr/bin/env bash
# End-to-end demo of the BlockParty Node CLI over the filesystem transport.
# Two "instances" (alice, bob) share a World and exchange an encrypted block via
# a public audience, then we run the connection-handshake demo.
set -euo pipefail
cd "$(dirname "$0")"

echo "== build =="
pnpm -r build >/dev/null

BP="node packages/cli/dist/index.js"
NET="$(mktemp -d)"
WORLD="festival-2026"

echo
echo "== identities =="
$BP address --world "$WORLD" --pass alice
$BP address --world "$WORLD" --pass bob

echo
echo "== alice posts an encrypted block to public-1 =="
$BP send --net "$NET" --world "$WORLD" --pass alice --audience 1 --text "anyone out there?"

echo
echo "== bob reads public-1 (decrypts) =="
$BP inbox --net "$NET" --world "$WORLD" --audience 1

echo
echo "== private connection handshake + encrypted exchange =="
$BP demo

echo
echo "demo complete."
