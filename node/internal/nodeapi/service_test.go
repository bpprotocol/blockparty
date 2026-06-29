package nodeapi_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"

	"github.com/bpprotocol/blockparty/implementations/go/audiences"
	"github.com/bpprotocol/blockparty/implementations/go/derive"
	"github.com/bpprotocol/blockparty/node/internal/authz"
	"github.com/bpprotocol/blockparty/node/internal/config"
	"github.com/bpprotocol/blockparty/node/internal/core"
	"github.com/bpprotocol/blockparty/node/internal/nodeapi"
	"github.com/bpprotocol/blockparty/node/internal/nodepb"
	"github.com/bpprotocol/blockparty/node/internal/nodepb/nodepbconnect"
	"github.com/bpprotocol/blockparty/node/internal/store"
	"github.com/bpprotocol/blockparty/node/internal/world"
)

const (
	token  = "test-token"
	seed   = "service-test-world"
	idPass = "id-pass"
	ksPass = "unlock"
)

// setup builds an unconfigured personal core behind a token-protected Connect
// server and returns an authenticated client plus the data dir.
func setup(t *testing.T) nodepbconnect.NodeServiceClient {
	t.Helper()
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	st, err := store.Open(dir, log)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	cfg := config.Config{Mode: config.ModePersonal, DataDir: dir}
	w, err := world.Load(cfg) // unloaded — no seed
	if err != nil {
		t.Fatalf("world.Load: %v", err)
	}
	c, err := core.New(cfg, "test", log, st, w, nil)
	if err != nil {
		t.Fatalf("core.New: %v", err)
	}

	path, handler := nodepbconnect.NewNodeServiceHandler(nodeapi.New(c))
	mux := http.NewServeMux()
	mux.Handle(path, authz.RequireToken(token, log)(handler))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	auth := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	})
	return nodepbconnect.NewNodeServiceClient(srv.Client(), srv.URL, connect.WithInterceptors(auth))
}

func TestUnauthenticatedRejected(t *testing.T) {
	_ = setup(t) // configures the server
	// A client with no token interceptor must be rejected.
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, _ := store.Open(dir, log)
	t.Cleanup(func() { st.Close() })
	cfg := config.Config{Mode: config.ModePersonal, DataDir: dir}
	w, _ := world.Load(cfg)
	c, _ := core.New(cfg, "test", log, st, w, nil)
	path, handler := nodepbconnect.NewNodeServiceHandler(nodeapi.New(c))
	mux := http.NewServeMux()
	mux.Handle(path, authz.RequireToken(token, log)(handler))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	noAuth := nodepbconnect.NewNodeServiceClient(srv.Client(), srv.URL)
	_, err := noAuth.GetStatus(context.Background(), connect.NewRequest(&nodepb.GetStatusRequest{}))
	if err == nil {
		t.Fatal("expected unauthenticated request to be rejected")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Logf("note: code = %v (want unauthenticated)", connect.CodeOf(err))
	}
}

func TestBootstrapPostReadFlow(t *testing.T) {
	client := setup(t)
	ctx := context.Background()

	// 1. Before bootstrap: no World, cannot author.
	st, err := client.GetStatus(ctx, connect.NewRequest(&nodepb.GetStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st.Msg.WorldLoaded {
		t.Fatal("expected no World before bootstrap")
	}

	// 2. Bootstrap a World.
	bs, err := client.BootstrapWorld(ctx, connect.NewRequest(&nodepb.BootstrapWorldRequest{
		WorldSeed:          seed,
		IdentityPassphrase: idPass,
		KeystorePassphrase: ksPass,
	}))
	if err != nil {
		t.Fatalf("BootstrapWorld: %v", err)
	}
	if bs.Msg.World == "" || bs.Msg.Identity == "" {
		t.Fatal("bootstrap returned empty world/identity")
	}

	// 3. Status now reflects the loaded World and author capability.
	st, _ = client.GetStatus(ctx, connect.NewRequest(&nodepb.GetStatusRequest{}))
	if !st.Msg.WorldLoaded || !st.Msg.CanAuthor {
		t.Fatalf("after bootstrap world_loaded=%v can_author=%v", st.Msg.WorldLoaded, st.Msg.CanAuthor)
	}
	if st.Msg.Identity != bs.Msg.Identity {
		t.Errorf("identity mismatch: status %q vs bootstrap %q", st.Msg.Identity, bs.Msg.Identity)
	}

	// 4. Bootstrapping again must fail (single-World invariant).
	if _, err := client.BootstrapWorld(ctx, connect.NewRequest(&nodepb.BootstrapWorldRequest{
		WorldSeed: seed, KeystorePassphrase: ksPass,
	})); err == nil {
		t.Error("expected second bootstrap to fail")
	}

	// 5. Post a content.post to public-1.
	post, err := client.PostText(ctx, connect.NewRequest(&nodepb.PostTextRequest{
		PublicAudience: 1,
		Text:           "hello world",
	}))
	if err != nil {
		t.Fatalf("PostText: %v", err)
	}
	if post.Msg.Id == "" {
		t.Fatal("post returned empty id")
	}

	// 6. Read it back, decrypted.
	got, err := client.GetBlock(ctx, connect.NewRequest(&nodepb.GetBlockRequest{Id: post.Msg.Id}))
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
	}
	if !got.Msg.Decrypted {
		t.Error("expected node to decrypt a public-audience post")
	}
	if got.Msg.Text != "hello world" {
		t.Errorf("text = %q, want %q", got.Msg.Text, "hello world")
	}
	if got.Msg.Summary.Author != bs.Msg.Identity {
		t.Errorf("author = %q, want self %q", got.Msg.Summary.Author, bs.Msg.Identity)
	}

	// 7. List by audience and by author.
	audCode := derive.GetAudienceCode(derive.OpenWorld(seed), audiences.PublicAudienceID(1)).Hex()
	byAud, err := client.ListBlocks(ctx, connect.NewRequest(&nodepb.ListBlocksRequest{AudienceCode: audCode}))
	if err != nil {
		t.Fatalf("ListBlocks(audience): %v", err)
	}
	if len(byAud.Msg.Blocks) != 1 || byAud.Msg.Blocks[0].Id != post.Msg.Id {
		t.Errorf("ListBlocks(audience) = %d blocks, want the posted one", len(byAud.Msg.Blocks))
	}
	byAuthor, err := client.ListBlocks(ctx, connect.NewRequest(&nodepb.ListBlocksRequest{Author: bs.Msg.Identity}))
	if err != nil {
		t.Fatalf("ListBlocks(author): %v", err)
	}
	if len(byAuthor.Msg.Blocks) != 1 {
		t.Errorf("ListBlocks(author) = %d, want 1", len(byAuthor.Msg.Blocks))
	}

	// 8. Missing block → NotFound.
	_, err = client.GetBlock(ctx, connect.NewRequest(&nodepb.GetBlockRequest{Id: "deadbeef"}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("GetBlock(missing) code = %v, want not_found", connect.CodeOf(err))
	}
}

func TestRelayCannotBootstrapOrAuthor(t *testing.T) {
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	st, _ := store.Open(dir, log)
	t.Cleanup(func() { st.Close() })
	cfg := config.Config{Mode: config.ModeRelay, DataDir: dir}
	w, _ := world.Load(cfg)
	c, _ := core.New(cfg, "test", log, st, w, nil)

	path, handler := nodepbconnect.NewNodeServiceHandler(nodeapi.New(c))
	mux := http.NewServeMux()
	mux.Handle(path, authz.RequireToken(token, log)(handler))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	auth := connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	})
	client := nodepbconnect.NewNodeServiceClient(srv.Client(), srv.URL, connect.WithInterceptors(auth))

	if _, err := client.BootstrapWorld(context.Background(), connect.NewRequest(&nodepb.BootstrapWorldRequest{
		WorldSeed: seed, KeystorePassphrase: ksPass,
	})); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Errorf("relay bootstrap code = %v, want permission_denied", connect.CodeOf(err))
	}
}
