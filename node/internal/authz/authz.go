// Package authz implements the node API trust boundary (#29). The personal-mode
// API is an "act-as-me" oracle — anything that can call it can post and sign as
// the user — so it is locked down by:
//
//   - loopback-only binding by default (non-loopback requires explicit opt-in),
//   - a bearer token, generated in the data dir, required on the RPC surface,
//   - a confirmation gate for dangerous, irreversible operations.
package authz

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const tokenFile = "api.token"

// TokenPath returns the API token location within a data directory.
func TokenPath(dataDir string) string { return filepath.Join(dataDir, tokenFile) }

// LoadOrCreateToken returns the API bearer token at <dataDir>/api.token,
// creating a fresh 256-bit token (file mode 0600) on first use. A local client
// on the same host reads this file to authenticate.
func LoadOrCreateToken(dataDir string) (string, error) {
	path := TokenPath(dataDir)
	if b, err := os.ReadFile(path); err == nil {
		if t := strings.TrimSpace(string(b)); t != "" {
			return t, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", err
	}
	return token, nil
}

// RequireToken wraps next, rejecting any request that does not carry a matching
// "Authorization: Bearer <token>" header. An empty configured token denies all.
func RequireToken(token string, log *slog.Logger) func(http.Handler) http.Handler {
	want := []byte(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := bearer(r.Header.Get("Authorization"))
			if token == "" || subtle.ConstantTimeCompare(got, want) != 1 {
				if log != nil {
					log.Warn("api: rejected unauthenticated request", "path", r.URL.Path, "remote", r.RemoteAddr)
				}
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func bearer(header string) []byte {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return []byte(header[len(prefix):])
	}
	return nil
}

// IsLoopback reports whether a host:port listen address binds only the loopback
// interface. An address that binds all interfaces (e.g. ":4400") is not loopback.
func IsLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ErrConfirmationRequired indicates a dangerous operation was attempted without
// the required explicit confirmation.
var ErrConfirmationRequired = errors.New("authz: operation requires explicit confirmation")

// Confirmer gates dangerous, irreversible operations (authoring an
// identity.burn, exporting keys). The headless node cannot prompt the user, so
// confirmation is surfaced to the client: the client must pass an explicit
// confirmation flag, which the node records.
type Confirmer interface {
	Confirm(op string, confirmed bool) error
}

// ExplicitConfirmer requires confirmed==true and logs every gated attempt.
type ExplicitConfirmer struct{ Log *slog.Logger }

// Confirm returns ErrConfirmationRequired unless confirmed is true.
func (c ExplicitConfirmer) Confirm(op string, confirmed bool) error {
	if !confirmed {
		if c.Log != nil {
			c.Log.Warn("dangerous operation blocked: confirmation required", "op", op)
		}
		return ErrConfirmationRequired
	}
	if c.Log != nil {
		c.Log.Warn("dangerous operation confirmed", "op", op)
	}
	return nil
}
