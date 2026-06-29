package vectors

import (
	"encoding/json"
	"flag"
	"os"
	"testing"
)

const vectorsFile = "../../conformance/vectors.json"

var update = flag.Bool("update-vectors", false, "regenerate vectors.json")

// TestVectors locks the conformance vectors: Compute() must reproduce the
// committed vectors.json byte-for-byte. Run with -update-vectors to regenerate
// after an intentional change.
func TestVectors(t *testing.T) {
	got, err := json.MarshalIndent(Compute(), "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')

	if *update {
		if err := os.WriteFile(vectorsFile, got, 0o644); err != nil {
			t.Fatalf("write %s: %v", vectorsFile, err)
		}
		t.Logf("wrote %s", vectorsFile)
		return
	}

	want, err := os.ReadFile(vectorsFile)
	if err != nil {
		t.Fatalf("read %s (run with -update-vectors to create): %v", vectorsFile, err)
	}
	if string(got) != string(want) {
		t.Errorf("vectors.json is out of date; run `go test ./vectors -update-vectors`")
	}
}
