package blocktypes

import (
	"errors"
	"strconv"

	"github.com/bpprotocol/blockparty/implementations/go/blockpb"
)

// MethodCorePing is the canonical method identifier for the core ping RPC.
const MethodCorePing = "bpprotocol.org/v1/rpc/core.ping"

// ErrNotPing is returned when an RPC message is not a core.ping.
var ErrNotPing = errors.New("blocktypes: not a core.ping")

// BuildPing builds a core.ping rpc.request carrying the sender's timestamp.
func BuildPing(id string, timestamp int64) *blockpb.RpcRequest {
	return &blockpb.RpcRequest{
		Id:     id,
		Method: MethodCorePing,
		Params: map[string]string{"timestamp": strconv.FormatInt(timestamp, 10)},
	}
}

// HandlePing builds the rpc.response for a core.ping request, echoing the
// original timestamp and recording when it was received and the responder's
// clock and node string. It returns ErrNotPing if req is not a core.ping.
func HandlePing(req *blockpb.RpcRequest, receivedAt, serverTime int64, node string) (*blockpb.RpcResponse, error) {
	if req.Method != MethodCorePing {
		return nil, ErrNotPing
	}
	ts, err := strconv.ParseInt(req.Params["timestamp"], 10, 64)
	if err != nil {
		return nil, err
	}
	return &blockpb.RpcResponse{
		Id: req.Id,
		Result: map[string]string{
			"timestamp":   strconv.FormatInt(ts, 10),
			"received_at": strconv.FormatInt(receivedAt, 10),
			"server_time": strconv.FormatInt(serverTime, 10),
			"node":        node,
		},
	}, nil
}

// PingResult is the parsed payload of a core.ping rpc.response.
type PingResult struct {
	Timestamp  int64
	ReceivedAt int64
	ServerTime int64
	Node       string
}

// ParsePingResponse parses a core.ping rpc.response.
func ParsePingResponse(resp *blockpb.RpcResponse) (PingResult, error) {
	var r PingResult
	var err error
	if r.Timestamp, err = strconv.ParseInt(resp.Result["timestamp"], 10, 64); err != nil {
		return r, err
	}
	if r.ReceivedAt, err = strconv.ParseInt(resp.Result["received_at"], 10, 64); err != nil {
		return r, err
	}
	if r.ServerTime, err = strconv.ParseInt(resp.Result["server_time"], 10, 64); err != nil {
		return r, err
	}
	r.Node = resp.Result["node"]
	return r, nil
}
