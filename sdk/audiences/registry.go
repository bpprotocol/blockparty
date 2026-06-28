package audiences

import "github.com/bpprotocol/blockparty/sdk/derive"

// Registry is a local map from audience code to the audience a client knows,
// letting it find the secret for a received block's audience_code. It is the
// "local audience map" referenced in the spec. Registry is not safe for
// concurrent mutation; guard it externally if shared across goroutines.
type Registry struct {
	byCode map[string]Audience // keyed by code hex
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{byCode: make(map[string]Audience)}
}

// Add records an audience, keyed by its code.
func (r *Registry) Add(a Audience) {
	r.byCode[a.Code.Hex()] = a
}

// ByCode returns the audience for a raw audience code (e.g. a block's
// audience_code), and whether it was found.
func (r *Registry) ByCode(code []byte) (Audience, bool) {
	a, ok := r.byCode[derive.Code(code).Hex()]
	return a, ok
}

// Len returns the number of audiences in the registry.
func (r *Registry) Len() int { return len(r.byCode) }
