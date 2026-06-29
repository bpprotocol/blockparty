package audiences

import (
	"encoding/hex"
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/derive"
)

// Golden conformance vectors in the test world.
const (
	goldenPublicOneSecret = "8f63e9df89faa3212db39bad276d43b26259d85513f28b3ef43668a09fc24820"
	goldenInboxCode       = "a63d7db7e718498a6f2fda53c73f2c4f"
)

var goldenInboxAddr = derive.Address("cff62cff35a0cc4271c262c07a86cbccc63ad13a")

func TestGoldenAudiences(t *testing.T) {
	w := derive.OpenWorld(testWorldPhrase)
	if got := hex.EncodeToString(PublicAudienceSecret(w, PublicAudienceID(1))); got != goldenPublicOneSecret {
		t.Errorf("public-1 secret changed:\n got  %s\n want %s", got, goldenPublicOneSecret)
	}
	if got := InboxAudience(w, goldenInboxAddr).Code.Hex(); got != goldenInboxCode {
		t.Errorf("inbox code changed:\n got  %s\n want %s", got, goldenInboxCode)
	}
}
