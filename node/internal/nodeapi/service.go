// Package nodeapi implements the Connect NodeService (#38) over the core
// controller. It maps protobuf requests to core operations and core errors to
// Connect status codes. The transport-level trust boundary (loopback + bearer
// token, #29) is enforced by the api server that mounts this handler.
package nodeapi

import (
	"context"
	"encoding/hex"
	"errors"

	"connectrpc.com/connect"

	"github.com/bpprotocol/blockparty/node/internal/core"
	"github.com/bpprotocol/blockparty/node/internal/nodepb"
	"github.com/bpprotocol/blockparty/node/internal/nodepb/nodepbconnect"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

// Service adapts a *core.Core to the generated NodeServiceHandler.
type Service struct {
	core *core.Core
}

var _ nodepbconnect.NodeServiceHandler = (*Service)(nil)

// New builds the node API service over the given core.
func New(c *core.Core) *Service { return &Service{core: c} }

func hexID(b []byte) string { return hex.EncodeToString(b) }

func (s *Service) GetStatus(_ context.Context, _ *connect.Request[nodepb.GetStatusRequest]) (*connect.Response[nodepb.GetStatusResponse], error) {
	st := s.core.Status()
	return connect.NewResponse(&nodepb.GetStatusResponse{
		Version:     st.Version,
		Mode:        st.Mode,
		WorldLoaded: st.WorldLoaded,
		World:       st.World,
		Identity:    st.Identity,
		BlockCount:  int64(st.BlockCount),
		CanAuthor:   st.CanAuthor,
	}), nil
}

func (s *Service) BootstrapWorld(_ context.Context, req *connect.Request[nodepb.BootstrapWorldRequest]) (*connect.Response[nodepb.BootstrapWorldResponse], error) {
	m := req.Msg
	st, err := s.core.BootstrapWorld(m.WorldSeed, m.IdentityPassphrase, m.KeystorePassphrase)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.BootstrapWorldResponse{
		World:    st.World,
		Identity: st.Identity,
	}), nil
}

func (s *Service) PostText(_ context.Context, req *connect.Request[nodepb.PostTextRequest]) (*connect.Response[nodepb.PostTextResponse], error) {
	id, err := s.core.PostText(int(req.Msg.PublicAudience), req.Msg.Text)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.PostTextResponse{Id: id}), nil
}

func (s *Service) GetBlock(_ context.Context, req *connect.Request[nodepb.GetBlockRequest]) (*connect.Response[nodepb.GetBlockResponse], error) {
	res, err := s.core.GetBlock(req.Msg.Id)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.GetBlockResponse{
		Summary:   summary(res.Record),
		Data:      res.Record.Block.Data,
		Decrypted: res.Decrypted,
		Text:      res.Text,
	}), nil
}

func (s *Service) ListBlocks(_ context.Context, req *connect.Request[nodepb.ListBlocksRequest]) (*connect.Response[nodepb.ListBlocksResponse], error) {
	m := req.Msg
	entries, err := s.core.ListBlocks(core.Filter{
		Audience: m.AudienceCode,
		Type:     m.TypeCode,
		Author:   m.Author,
		From:     m.From,
		To:       m.To,
	})
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*nodepb.BlockSummary, 0, len(entries))
	for _, e := range entries {
		out = append(out, &nodepb.BlockSummary{
			Id:           e.ID,
			AudienceCode: e.Audience,
			TypeCode:     e.Type,
			Author:       e.Author,
			Timestamp:    e.Timestamp,
			ReceivedAt:   e.ReceivedAt,
		})
	}
	return connect.NewResponse(&nodepb.ListBlocksResponse{Blocks: out}), nil
}

func summary(rec *store.Record) *nodepb.BlockSummary {
	b := rec.Block
	return &nodepb.BlockSummary{
		Id:           hexID(b.Id),
		AudienceCode: hexID(b.AudienceCode),
		TypeCode:     hexID(b.TypeCode),
		Author:       rec.Author,
		Timestamp:    b.Timestamp,
		ReceivedAt:   rec.ReceivedAt,
	}
}

// mapErr translates core/store errors to Connect status codes.
func mapErr(err error) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, core.ErrNoWorld),
		errors.Is(err, core.ErrAlreadyLoaded),
		errors.Is(err, core.ErrCannotAuthor):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, core.ErrRelayBootstrap):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}
