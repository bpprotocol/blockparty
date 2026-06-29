package blocktypes

import (
	"testing"

	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
)

func TestPingRoundTrip(t *testing.T) {
	req := BuildPing("ping-1", 1746396000)
	if req.Method != MethodCorePing || req.Params["timestamp"] != "1746396000" {
		t.Fatalf("unexpected ping request: %+v", req)
	}

	resp, err := HandlePing(req, 1746396002, 1746396003, "BlockParty Reference Node 1.0.0")
	if err != nil {
		t.Fatalf("HandlePing: %v", err)
	}
	if resp.Id != req.Id {
		t.Fatalf("response id %q != request id %q", resp.Id, req.Id)
	}

	got, err := ParsePingResponse(resp)
	if err != nil {
		t.Fatalf("ParsePingResponse: %v", err)
	}
	if got.Timestamp != 1746396000 || got.ReceivedAt != 1746396002 || got.ServerTime != 1746396003 {
		t.Fatalf("parsed ping result wrong: %+v", got)
	}
	if got.Node != "BlockParty Reference Node 1.0.0" {
		t.Fatalf("node = %q", got.Node)
	}
}

func TestHandlePingRejectsNonPing(t *testing.T) {
	req := &blockpb.RpcRequest{Id: "x", Method: "example.com/v1/rpc/other"}
	if _, err := HandlePing(req, 0, 0, ""); err != ErrNotPing {
		t.Fatalf("HandlePing(non-ping) = %v, want ErrNotPing", err)
	}
}
