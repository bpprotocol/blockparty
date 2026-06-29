package encryption

import (
	"encoding/hex"
	"testing"
)

// Golden conformance vectors for the AEAD layer over fixed inputs (fixed nonce
// so the ciphertext is reproducible).
const (
	goldenContentKey = "0808611f62904c55850934ce82b79273f64ed63902e6760cf696cbeacfda78d6"
	goldenAAD        = "00000001a451d8b7e0ec778506b8001ebd31c2b98572d37a189d92544082f34254f01d8e000000006553f100"
	goldenData       = "627070726f746f636f6c2f76312f746573742d6e6f6e63650a9bfa3ae750385a912d05d193027bdd1daad10ef7a2536c48cb2be391d7f1" // nonce || ciphertext
)

var (
	goldenNonce     = []byte("bpprotocol/v1/test-nonce") // exactly 24 bytes
	goldenPlaintext = []byte("hello, audience")
)

func TestGoldenAEAD(t *testing.T) {
	e := newEnv()
	aad := AAD(1, e.typeCode, e.audCode, testTimestamp)

	if got := hex.EncodeToString(ContentKey(e.secret, e.audCode, goldenNonce)); got != goldenContentKey {
		t.Errorf("ContentKey changed:\n got  %s\n want %s", got, goldenContentKey)
	}
	if got := hex.EncodeToString(aad); got != goldenAAD {
		t.Errorf("AAD changed:\n got  %s\n want %s", got, goldenAAD)
	}
	data, err := sealWithNonce(e.secret, e.audCode, goldenNonce, goldenPlaintext, aad)
	if err != nil {
		t.Fatalf("sealWithNonce: %v", err)
	}
	if got := hex.EncodeToString(data); got != goldenData {
		t.Errorf("sealed data changed:\n got  %s\n want %s", got, goldenData)
	}
}
