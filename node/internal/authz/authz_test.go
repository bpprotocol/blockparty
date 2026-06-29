package authz

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestLoadOrCreateTokenPersists(t *testing.T) {
	dir := t.TempDir()
	tok, err := LoadOrCreateToken(dir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if len(tok) != 64 {
		t.Errorf("token len = %d, want 64 hex chars", len(tok))
	}
	again, err := LoadOrCreateToken(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if again != tok {
		t.Error("token changed across reloads")
	}
	info, err := os.Stat(TokenPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token perms = %o, want 600", perm)
	}
}

func TestRequireToken(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RequireToken("secret-token", nil)(ok)

	cases := []struct {
		name   string
		header string
		want   int
	}{
		{"no header", "", http.StatusUnauthorized},
		{"wrong token", "Bearer nope", http.StatusUnauthorized},
		{"not bearer", "Basic secret-token", http.StatusUnauthorized},
		{"correct", "Bearer secret-token", http.StatusOK},
		{"case-insensitive scheme", "bearer secret-token", http.StatusOK},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Errorf("status = %d, want %d", rec.Code, c.want)
			}
		})
	}
}

func TestEmptyTokenDeniesAll(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RequireToken("", nil)(ok)
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 when no token configured", rec.Code)
	}
}

func TestIsLoopback(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:4400": true,
		"localhost:4400": true,
		"[::1]:4400":     true,
		"0.0.0.0:4400":   false,
		":4400":          false,
		"192.168.1.5:80": false,
	}
	for addr, want := range cases {
		if got := IsLoopback(addr); got != want {
			t.Errorf("IsLoopback(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestExplicitConfirmer(t *testing.T) {
	c := ExplicitConfirmer{}
	if err := c.Confirm("identity.burn", false); err != ErrConfirmationRequired {
		t.Errorf("unconfirmed err = %v, want ErrConfirmationRequired", err)
	}
	if err := c.Confirm("identity.burn", true); err != nil {
		t.Errorf("confirmed err = %v, want nil", err)
	}
}
