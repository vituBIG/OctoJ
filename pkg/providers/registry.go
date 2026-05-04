package providers

import (
	"fmt"

	"github.com/OctavoBit/octoj/pkg/providers/corretto"
	"github.com/OctavoBit/octoj/pkg/providers/liberica"
	"github.com/OctavoBit/octoj/pkg/providers/temurin"
	"github.com/OctavoBit/octoj/pkg/providers/zulu"
)

// Registry holds all registered JDK providers.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a Registry pre-populated with all built-in providers.
func NewRegistry() *Registry {
	r := &Registry{
		providers: make(map[string]Provider),
	}

	r.Register(temurin.New())
	r.Register(corretto.New())
	r.Register(zulu.New())
	r.Register(liberica.New())

	return r
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get returns the provider with the given name, or an error if not found.
func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found — available: temurin, corretto, zulu, liberica", name)
	}
	return p, nil
}

// All returns all registered providers.
func (r *Registry) All() []Provider {
	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// Names returns the names of all registered providers.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
