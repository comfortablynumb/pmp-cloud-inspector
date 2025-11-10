//go:build auth0
// +build auth0

package auth0

import (
	"context"
	"fmt"
	"os"

	"github.com/auth0/go-auth0/management"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the Auth0 provider
type Provider struct {
	config       config.ProviderConfig
	client       *management.Management
	domain       string
	clientID     string
	clientSecret string
}

// init registers the Auth0 provider
func init() {
	provider.Register("auth0", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "auth0"
}

// Initialize sets up the Auth0 provider with credentials
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Get Auth0 credentials from environment variables
	p.domain = os.Getenv("AUTH0_DOMAIN")
	if p.domain == "" {
		return fmt.Errorf("AUTH0_DOMAIN environment variable is required")
	}

	// Support both client credentials and management API token
	// Client credentials are preferred for production use
	p.clientID = os.Getenv("AUTH0_CLIENT_ID")
	p.clientSecret = os.Getenv("AUTH0_CLIENT_SECRET")

	var err error
	if p.clientID != "" && p.clientSecret != "" {
		// Use client credentials
		p.client, err = management.New(
			p.domain,
			management.WithClientCredentials(ctx, p.clientID, p.clientSecret),
		)
	} else {
		// Fallback to management API token
		token := os.Getenv("AUTH0_MANAGEMENT_API_TOKEN")
		if token == "" {
			return fmt.Errorf("either AUTH0_CLIENT_ID and AUTH0_CLIENT_SECRET, or AUTH0_MANAGEMENT_API_TOKEN must be set")
		}
		p.client, err = management.New(
			p.domain,
			management.WithStaticToken(token),
		)
	}

	if err != nil {
		return fmt.Errorf("failed to create Auth0 client: %w", err)
	}

	return nil
}

// GetSupportedResourceTypes returns all resource types this provider supports
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeAuth0User,
		resource.TypeAuth0Role,
		resource.TypeAuth0Client,
		resource.TypeAuth0ResourceServer,
		resource.TypeAuth0Connection,
	}
}

// CollectResources collects all resources of the specified types
func (p *Provider) CollectResources(ctx context.Context, types []resource.ResourceType) (*resource.Collection, error) {
	collection := resource.NewCollection()

	// If no types specified, collect all supported types
	if len(types) == 0 {
		types = p.GetSupportedResourceTypes()
	}

	// Create a set for quick lookup
	typeSet := make(map[resource.ResourceType]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	// Collect resources based on requested types
	if typeSet[resource.TypeAuth0User] {
		if err := p.collectUsers(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect users: %w", err)
		}
	}

	if typeSet[resource.TypeAuth0Role] {
		if err := p.collectRoles(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect roles: %w", err)
		}
	}

	if typeSet[resource.TypeAuth0Client] {
		if err := p.collectClients(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect clients: %w", err)
		}
	}

	if typeSet[resource.TypeAuth0ResourceServer] {
		if err := p.collectResourceServers(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect resource servers: %w", err)
		}
	}

	if typeSet[resource.TypeAuth0Connection] {
		if err := p.collectConnections(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect connections: %w", err)
		}
	}

	return collection, nil
}

// DiscoverRelationships analyzes resources and establishes relationships between them
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Build relationships based on Auth0 resource structure
	// Example: Users -> Roles, Clients -> Resource Servers, etc.
	for _, res := range collection.Resources {
		switch res.Type {
		case resource.TypeAuth0User:
			// Users can have roles
			if userID, ok := res.Properties["user_id"].(string); ok {
				roleList, err := p.client.User.Roles(ctx, userID)
				if err == nil && roleList != nil && len(roleList.Roles) > 0 {
					for _, role := range roleList.Roles {
						if role.ID != nil {
							res.Relationships = append(res.Relationships, resource.Relationship{
								Type:       "has_role",
								TargetID:   *role.ID,
								TargetType: resource.TypeAuth0Role,
							})
						}
					}
				}
			}
		}
	}

	return nil
}

// GetAccounts returns available accounts (tenant) for this provider
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	// Auth0 has a single tenant per domain
	return []string{p.domain}, nil
}

// GetRegions returns available regions for this provider
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	// Auth0 is a managed service with no user-facing regions
	return []string{}, nil
}
