#!/usr/bin/env bash
# End-to-end demo of the BlockParty reference CLI over the filesystem transport.
# Two "instances" (alice, bob) share a World and exchange an encrypted block via
# a public audience, then we run the connection-handshake demo.
set -euo pipefail
cd "$(dirname "$0")"

NET="$(mktemp -d)"
WORLD="festival-2026"

echo "== build bp =="
go build -o ./bp ./cmd/bp
trap 'rm -f ./bp' EXIT

echo
echo "== identities =="
./bp address --world "$WORLD" --pass alice
./bp address --world "$WORLD" --pass bob

echo
echo "== alice posts an encrypted block to public-1 =="
./bp send --net "$NET" --world "$WORLD" --pass alice --audience 1 --text "anyone out there?"

echo
echo "== bob reads public-1 (decrypts) =="
./bp inbox --net "$NET" --world "$WORLD" --audience 1

echo
echo "== private connection handshake + encrypted exchange =="
./bp demo --net "$(mktemp -d)"

echo
echo "demo complete."
