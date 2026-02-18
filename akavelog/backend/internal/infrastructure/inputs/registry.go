package inputs

import (
	"fmt"
	"net/http"
	"sync"
)

// Registry holds registered input factories. The backend uses it to create inputs.
// Infrastructure packages (e.g. httpinput) register their factory in init().
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// NewRegistry returns a new Registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

// Register adds a factory for an input type.
func (r *Registry) Register(factory Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[factory.Name()] = factory
}

// Create builds a MessageInput for the given type and config.
func (r *Registry) Create(name string, cfg Config, buffer InputBuffer) (MessageInput, error) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown input type: %s", name)
	}
	return factory.Create(cfg, buffer)
}

// ValidateConfig runs the factory's optional ValidateConfig before create. Returns nil if type unknown or no validator.
func (r *Registry) ValidateConfig(typeName string, cfg Config) error {
	r.mu.RLock()
	factory, ok := r.factories[typeName]
	r.mu.RUnlock()
	if !ok {
		return nil
	}
	if v, ok := factory.(interface{ ValidateConfig(Config) error }); ok {
		return v.ValidateConfig(cfg)
	}
	return nil
}

// ListRegistered returns all registered input type names.
func (r *Registry) ListRegistered() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// GetTypeInfo returns the config spec for the given input type. ok is false if the type is not registered.
func (r *Registry) GetTypeInfo(name string) (info InputTypeInfo, ok bool) {
	r.mu.RLock()
	factory, ok := r.factories[name]
	r.mu.RUnlock()
	if !ok {
		return InputTypeInfo{}, false
	}
	return factory.ConfigSpec(), true
}

// AllTypesInfo returns config specs for all registered input types.
func (r *Registry) AllTypesInfo() []InputTypeInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]InputTypeInfo, 0, len(r.factories))
	for _, factory := range r.factories {
		out = append(out, factory.ConfigSpec())
	}
	return out
}

// MountHTTPEndpoints creates inputs from specs and mounts HTTPEndpointInput handlers onto mux.
func (r *Registry) MountHTTPEndpoints(mux *http.ServeMux, specs []InputSpec, buffer InputBuffer) error {
	for _, spec := range specs {
		input, err := r.Create(spec.Type, spec.ConfigWithDescription(), buffer)
		if err != nil {
			return err
		}
		ep, ok := input.(HTTPEndpointInput)
		if !ok {
			continue
		}
		mux.Handle(ep.Path(), ep.Handler())
	}
	return nil
}
