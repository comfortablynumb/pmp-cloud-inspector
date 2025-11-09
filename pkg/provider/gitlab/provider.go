package gitlab

import (
	"context"
	"fmt"

	"github.com/xanzy/go-gitlab"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the GitLab provider
type Provider struct {
	config config.ProviderConfig
	client *gitlab.Client
	groups []string // Group names/IDs to inspect
}

// init registers the GitLab provider
func init() {
	provider.Register("gitlab", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "gitlab"
}

// Initialize sets up the GitLab provider
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Get token from options
	token, ok := cfg.Options["token"].(string)
	if !ok || token == "" {
		return fmt.Errorf("gitlab token is required in provider options")
	}

	// Get base URL (optional, defaults to gitlab.com)
	baseURL, _ := cfg.Options["base_url"].(string)

	var err error
	if baseURL != "" {
		p.client, err = gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	} else {
		p.client, err = gitlab.NewClient(token)
	}
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	// Get groups to inspect (use accounts as groups)
	if len(cfg.Accounts) > 0 {
		p.groups = cfg.Accounts
	}

	return nil
}

// GetSupportedResourceTypes returns all GitLab resource types
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeGitLabProject,
		resource.TypeGitLabGroup,
		resource.TypeGitLabUser,
	}
}

// CollectResources collects GitLab resources
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

	// Collect groups
	if typeSet[resource.TypeGitLabGroup] {
		if err := p.collectGroups(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect groups: %w", err)
		}
	}

	// Collect projects
	if typeSet[resource.TypeGitLabProject] {
		if err := p.collectProjects(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect projects: %w", err)
		}
	}

	// Collect users
	if typeSet[resource.TypeGitLabUser] {
		if err := p.collectUsers(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect users: %w", err)
		}
	}

	return collection, nil
}

// DiscoverRelationships establishes relationships between GitLab resources
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Build relationships based on GitLab resource structure
	for _, res := range collection.Resources {
		switch res.Type {
		case resource.TypeGitLabProject:
			p.discoverProjectRelationships(res, collection)
		}
	}

	return nil
}

// GetAccounts returns the configured groups
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	return p.groups, nil
}

// GetRegions returns empty slice (GitLab doesn't have regions)
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// safeString safely dereferences a string pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// safeInt safely dereferences an int pointer
func safeInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}
