package provider

import (
	"context"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider is the interface (trait) that all cloud providers must implement
type Provider interface {
	// Name returns the provider name (e.g., "aws", "gcp", "okta")
	Name() string

	// Initialize sets up the provider with the given configuration
	Initialize(ctx context.Context, config config.ProviderConfig) error

	// GetSupportedResourceTypes returns all resource types this provider supports
	GetSupportedResourceTypes() []resource.ResourceType

	// CollectResources collects all resources of the specified types
	// If types is empty, collect all supported types
	CollectResources(ctx context.Context, types []resource.ResourceType) (*resource.Collection, error)

	// DiscoverRelationships analyzes resources and establishes relationships between them
	DiscoverRelationships(ctx context.Context, collection *resource.Collection) error

	// GetAccounts returns available accounts for this provider
	GetAccounts(ctx context.Context) ([]string, error)

	// GetRegions returns available regions for this provider
	GetRegions(ctx context.Context) ([]string, error)
}

// Registry manages all registered providers
type Registry struct {
	providers map[string]ProviderFactory
}

// ProviderFactory is a function that creates a new provider instance
type ProviderFactory func() Provider

var globalRegistry = &Registry{
	providers: make(map[string]ProviderFactory),
}

// Register registers a new provider factory
func Register(name string, factory ProviderFactory) {
	globalRegistry.providers[name] = factory
}

// Get retrieves a provider factory by name
func Get(name string) (ProviderFactory, bool) {
	factory, ok := globalRegistry.providers[name]
	return factory, ok
}

// GetRegistry returns the global provider registry
func GetRegistry() *Registry {
	return globalRegistry
}

// List returns all registered provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// Create creates a new provider instance by name
func (r *Registry) Create(name string) (Provider, error) {
	factory, ok := r.providers[name]
	if !ok {
		return nil, &ProviderNotFoundError{Name: name}
	}
	return factory(), nil
}

// ProviderNotFoundError is returned when a provider is not found
type ProviderNotFoundError struct {
	Name string
}

func (e *ProviderNotFoundError) Error() string {
	return "provider not found: " + e.Name
}
