// Package registry wires together all JDK providers into a single lookup map.
// It lives in internal/ (not pkg/providers/) to break the import cycle:
//
//	pkg/providers/corretto → pkg/providers (interface)
//	pkg/providers (registry) → pkg/providers/corretto   ← cycle
package registry

import (
	"fmt"
	"sort"

	"github.com/OctavoBit/octoj/pkg/providers"
	"github.com/OctavoBit/octoj/pkg/providers/corretto"
	"github.com/OctavoBit/octoj/pkg/providers/liberica"
	"github.com/OctavoBit/octoj/pkg/providers/temurin"
	"github.com/OctavoBit/octoj/pkg/providers/zulu"
)

// Registry holds all registered JDK providers.
type Registry struct {
	providers map[string]providers.Provider
}

// New creates a Registry pre-populated with all built-in providers.
func New() *Registry {
	r := &Registry{providers: make(map[string]providers.Provider)}
	r.register(temurin.New())
	r.register(corretto.New())
	r.register(zulu.New())
	r.register(liberica.New())
	return r
}

func (r *Registry) register(p providers.Provider) {
	r.providers[p.Name()] = p
}

// Get returns the provider with the given name, or an error if not found.
func (r *Registry) Get(name string) (providers.Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found — available: %s", name, r.namesJoined())
	}
	return p, nil
}

// All returns all registered providers in deterministic (alphabetical) order.
func (r *Registry) All() []providers.Provider {
	names := make([]string, 0, len(r.providers))
	for n := range r.providers {
		names = append(names, n)
	}
	sort.Strings(names)

	result := make([]providers.Provider, 0, len(names))
	for _, n := range names {
		result = append(result, r.providers[n])
	}
	return result
}

// Names returns the sorted names of all registered providers.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for n := range r.providers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) namesJoined() string {
	names := r.Names()
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}
