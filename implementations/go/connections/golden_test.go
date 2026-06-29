package connections

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// Fixed inputs for the deterministic derivation vectors. The live handshake is
// non-deterministic (ephemeral keys, random nonces), so we lock the pure
// derivation functions instead.
var (
	vSS1       = bytes.Repeat([]byte{0x11}, 32)
	vSS2       = bytes.Repeat([]byte{0x22}, 32)
	vNonce     = bytes.Repeat([]byte{0x33}, 32)
	vNonceResp = bytes.Repeat([]byte{0x44}, 32)
	vSS3       = bytes.Repeat([]byte{0x55}, 32)
	vCT3       = bytes.Repeat([]byte{0x66}, 32)
)

const (
	goldenConnSecret0   = "29e33a2370510ba0d382c9acd53b5cb1022c9291ef3e15eab5e8855e9786e5b1"
	goldenAudienceCode0 = "df1b068f174fba21e8d4fb34c3a03a88"
	goldenAudienceCode1 = "23dd77260c274f6135c9181773353eba"
)

func TestGoldenDerivations(t *testing.T) {
	cs0 := connectionSecret0(vSS1, vSS2, vNonce, vNonceResp)
	if got := hex.EncodeToString(cs0); got != goldenConnSecret0 {
		t.Errorf("connectionSecret0 changed:\n got  %s\n want %s", got, goldenConnSecret0)
	}
	if got := audienceCode(cs0, 0).Hex(); got != goldenAudienceCode0 {
		t.Errorf("audienceCode(epoch0) changed:\n got  %s\n want %s", got, goldenAudienceCode0)
	}
	cs1 := rotatedSecret(cs0, vSS3, vCT3, 1)
	if got := audienceCode(cs1, 1).Hex(); got != goldenAudienceCode1 {
		t.Errorf("audienceCode(epoch1) changed:\n got  %s\n want %s", got, goldenAudienceCode1)
	}
}
