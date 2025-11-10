//go:build okta
// +build okta

package okta

import (
	"context"
	"fmt"
	"os"

	"github.com/okta/okta-sdk-golang/v2/okta"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the Okta provider
type Provider struct {
	config   config.ProviderConfig
	client   *okta.Client
	orgURL   string
	apiToken string
}

// init registers the Okta provider
func init() {
	provider.Register("okta", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "okta"
}

// Initialize sets up the Okta provider
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Get Okta org URL from environment variable
	p.orgURL = os.Getenv("OKTA_ORG_URL")
	if p.orgURL == "" {
		return fmt.Errorf("OKTA_ORG_URL environment variable is required")
	}

	// Get API token from environment variable
	p.apiToken = os.Getenv("OKTA_API_TOKEN")
	if p.apiToken == "" {
		return fmt.Errorf("OKTA_API_TOKEN environment variable is required")
	}

	// Create Okta client
	_, client, err := okta.NewClient(
		ctx,
		okta.WithOrgUrl(p.orgURL),
		okta.WithToken(p.apiToken),
		okta.WithCache(false), // Disable caching for accurate resource collection
	)
	if err != nil {
		return fmt.Errorf("failed to create Okta client: %w", err)
	}

	p.client = client
	return nil
}

// GetSupportedResourceTypes returns all Okta resource types
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeOktaUser,
		resource.TypeOktaGroup,
		resource.TypeOktaApplication,
		resource.TypeOktaAuthorizationServer,
	}
}

// CollectResources collects Okta resources
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

	// Collect users
	if typeSet[resource.TypeOktaUser] {
		if err := p.collectUsers(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect users: %w", err)
		}
	}

	// Collect groups
	if typeSet[resource.TypeOktaGroup] {
		if err := p.collectGroups(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect groups: %w", err)
		}
	}

	// Collect applications
	if typeSet[resource.TypeOktaApplication] {
		if err := p.collectApplications(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect applications: %w", err)
		}
	}

	// Collect authorization servers
	if typeSet[resource.TypeOktaAuthorizationServer] {
		if err := p.collectAuthorizationServers(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect authorization servers: %w", err)
		}
	}

	return collection, nil
}

// DiscoverRelationships establishes relationships between Okta resources
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Discover relationships between groups and users
	for _, res := range collection.Resources {
		if res.Type == resource.TypeOktaGroup {
			if err := p.discoverGroupMemberships(ctx, res, collection); err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: failed to discover group memberships for %s: %v\n", res.ID, err)
			}
		}
	}

	return nil
}

// GetAccounts returns empty slice (Okta doesn't have accounts concept)
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// GetRegions returns empty slice (Okta doesn't have regions)
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// discoverGroupMemberships discovers user memberships for a group
func (p *Provider) discoverGroupMemberships(ctx context.Context, group *resource.Resource, collection *resource.Collection) error {
	// Get group members
	users, _, err := p.client.Group.ListGroupUsers(ctx, group.ID, nil)
	if err != nil {
		return err
	}

	// Create relationships
	for _, user := range users {
		group.Relationships = append(group.Relationships, resource.Relationship{
			Type:       resource.RelationContains,
			TargetID:   user.Id,
			TargetType: resource.TypeOktaUser,
			Properties: map[string]interface{}{
				"membership_type": "direct",
			},
		})
	}

	return nil
}
