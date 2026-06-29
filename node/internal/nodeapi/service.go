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

	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/node/internal/authz"
	"github.com/bpprotocol/blockparty/node/internal/connections"
	"github.com/bpprotocol/blockparty/node/internal/core"
	"github.com/bpprotocol/blockparty/node/internal/nodepb"
	"github.com/bpprotocol/blockparty/node/internal/nodepb/nodepbconnect"
	"github.com/bpprotocol/blockparty/node/internal/store"
)

// Service adapts a *core.Core to the generated NodeServiceHandler.
type Service struct {
	core      *core.Core
	confirmer authz.Confirmer
}

var _ nodepbconnect.NodeServiceHandler = (*Service)(nil)

// New builds the node API service over the given core. confirmer gates dangerous
// operations (identity rotation/burn); pass nil to default to requiring explicit
// confirmation.
func New(c *core.Core, confirmer authz.Confirmer) *Service {
	if confirmer == nil {
		confirmer = authz.ExplicitConfirmer{}
	}
	return &Service{core: c, confirmer: confirmer}
}

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

func (s *Service) SubscribeBlocks(ctx context.Context, req *connect.Request[nodepb.SubscribeBlocksRequest], stream *connect.ServerStream[nodepb.BlockEvent]) error {
	ch, cancel := s.core.SubscribeBlocks(req.Msg.AudienceCode)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case ev, ok := <-ch:
			if !ok {
				return nil
			}
			err := stream.Send(&nodepb.BlockEvent{
				Summary: &nodepb.BlockSummary{
					Id:           ev.ID,
					AudienceCode: ev.Audience,
					TypeCode:     ev.Type,
					Author:       ev.Author,
					Timestamp:    ev.Timestamp,
					ReceivedAt:   ev.ReceivedAt,
				},
			})
			if err != nil {
				return err
			}
		}
	}
}

func (s *Service) GetIdentity(_ context.Context, _ *connect.Request[nodepb.GetIdentityRequest]) (*connect.Response[nodepb.GetIdentityResponse], error) {
	card, err := s.core.IdentityCard()
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.GetIdentityResponse{
		Address:  card.Address,
		KyberPub: card.KyberPub,
		MldsaPub: card.MLDSAPub,
	}), nil
}

func (s *Service) AddPeer(_ context.Context, req *connect.Request[nodepb.AddPeerRequest]) (*connect.Response[nodepb.AddPeerResponse], error) {
	if err := s.core.AddConnectionPeer(req.Msg.Address, req.Msg.KyberPub, req.Msg.MldsaPub); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.AddPeerResponse{}), nil
}

func (s *Service) StartConnection(_ context.Context, req *connect.Request[nodepb.StartConnectionRequest]) (*connect.Response[nodepb.StartConnectionResponse], error) {
	id, err := s.core.StartConnection(derive.Address(req.Msg.Address))
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.StartConnectionResponse{RequestId: id}), nil
}

func (s *Service) ListConnections(_ context.Context, _ *connect.Request[nodepb.ListConnectionsRequest]) (*connect.Response[nodepb.ListConnectionsResponse], error) {
	conns := s.core.Connections()
	out := make([]*nodepb.ConnectionInfo, 0, len(conns))
	for _, c := range conns {
		out = append(out, &nodepb.ConnectionInfo{Peer: c.Peer, Epoch: c.Epoch, AudienceCode: c.AudienceCode})
	}
	return connect.NewResponse(&nodepb.ListConnectionsResponse{Connections: out}), nil
}

func (s *Service) RotateConnection(_ context.Context, req *connect.Request[nodepb.ConnectionRef]) (*connect.Response[nodepb.ConnectionResult], error) {
	if err := s.core.RotateConnection(req.Msg.Address); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.ConnectionResult{}), nil
}

func (s *Service) CloseConnection(_ context.Context, req *connect.Request[nodepb.ConnectionRef]) (*connect.Response[nodepb.ConnectionResult], error) {
	if err := s.core.CloseConnection(req.Msg.Address); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.ConnectionResult{}), nil
}

func (s *Service) SendPrivateText(_ context.Context, req *connect.Request[nodepb.SendPrivateTextRequest]) (*connect.Response[nodepb.PostTextResponse], error) {
	id, err := s.core.SendPrivateText(req.Msg.Address, req.Msg.Text)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.PostTextResponse{Id: id}), nil
}

func (s *Service) ListConnectionMessages(_ context.Context, req *connect.Request[nodepb.ConnectionRef]) (*connect.Response[nodepb.ListConnectionMessagesResponse], error) {
	msgs, err := s.core.ConnectionMessages(req.Msg.Address)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*nodepb.PrivateMessage, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, &nodepb.PrivateMessage{Author: m.Author, Text: m.Text, Timestamp: m.Timestamp})
	}
	return connect.NewResponse(&nodepb.ListConnectionMessagesResponse{Messages: out}), nil
}

func (s *Service) RotateIdentity(_ context.Context, req *connect.Request[nodepb.RotateIdentityRequest]) (*connect.Response[nodepb.RotateIdentityResponse], error) {
	if err := s.confirmer.Confirm("identity.rotate", req.Msg.Confirm); err != nil {
		return nil, mapErr(err)
	}
	addr, err := s.core.RotateIdentity(req.Msg.NewPassphrase)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.RotateIdentityResponse{Identity: addr}), nil
}

func (s *Service) BurnIdentity(_ context.Context, req *connect.Request[nodepb.BurnIdentityRequest]) (*connect.Response[nodepb.BurnIdentityResponse], error) {
	if err := s.confirmer.Confirm("identity.burn", req.Msg.Confirm); err != nil {
		return nil, mapErr(err)
	}
	id, err := s.core.BurnIdentity(req.Msg.Notice)
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&nodepb.BurnIdentityResponse{BlockId: id}), nil
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
	case errors.Is(err, connections.ErrNoConnection):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, authz.ErrConfirmationRequired):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}
