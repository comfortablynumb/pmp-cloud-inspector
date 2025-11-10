package github

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the GitHub provider
type Provider struct {
	config config.ProviderConfig
	client *github.Client

	// Configuration
	organizations []string
	token         string
}

// init registers the GitHub provider
func init() {
	provider.Register("github", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "github"
}

// Initialize sets up the GitHub provider with credentials and configuration
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Get GitHub token from environment variable
	p.token = os.Getenv("GITHUB_TOKEN")
	if p.token == "" {
		return fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	// Create authenticated client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: p.token},
	)
	tc := oauth2.NewClient(ctx, ts)
	p.client = github.NewClient(tc)

	// Set up organizations from accounts field
	if len(cfg.Accounts) > 0 {
		p.organizations = cfg.Accounts
	}

	return nil
}

// GetSupportedResourceTypes returns all GitHub resource types supported
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeGitHubOrganization,
		resource.TypeGitHubRepository,
		resource.TypeGitHubTeam,
		resource.TypeGitHubUser,
	}
}

// CollectResources collects all specified GitHub resources
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

	// If no organizations specified, get user's organizations
	if len(p.organizations) == 0 {
		orgs, err := p.getUserOrganizations(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get user organizations: %w", err)
		}
		p.organizations = orgs
	}

	// Collect organizations
	if typeSet[resource.TypeGitHubOrganization] {
		if err := p.collectOrganizations(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect organizations: %w", err)
		}
	}

	// Collect resources for each organization
	for _, org := range p.organizations {
		if typeSet[resource.TypeGitHubRepository] {
			if err := p.collectRepositories(ctx, collection, org); err != nil {
				return nil, fmt.Errorf("failed to collect repositories for %s: %w", org, err)
			}
		}

		if typeSet[resource.TypeGitHubTeam] {
			if err := p.collectTeams(ctx, collection, org); err != nil {
				return nil, fmt.Errorf("failed to collect teams for %s: %w", org, err)
			}
		}

		if typeSet[resource.TypeGitHubUser] {
			if err := p.collectUsers(ctx, collection, org); err != nil {
				return nil, fmt.Errorf("failed to collect users for %s: %w", org, err)
			}
		}
	}

	return collection, nil
}

// DiscoverRelationships establishes relationships between GitHub resources
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Build relationships based on GitHub resource structure
	for _, res := range collection.Resources {
		switch res.Type {
		case resource.TypeGitHubOrganization:
			p.discoverOrganizationRelationships(res, collection)
		case resource.TypeGitHubRepository:
			p.discoverRepositoryRelationships(res, collection)
		case resource.TypeGitHubTeam:
			p.discoverTeamRelationships(res, collection)
		}
	}

	return nil
}

// GetAccounts returns the GitHub organization(s)
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	if len(p.organizations) > 0 {
		return p.organizations, nil
	}

	return p.getUserOrganizations(ctx)
}

// GetRegions returns available regions (not applicable for GitHub)
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// getUserOrganizations gets the authenticated user's organizations
func (p *Provider) getUserOrganizations(ctx context.Context) ([]string, error) {
	opts := &github.ListOptions{PerPage: 100}
	var allOrgs []string

	for {
		orgs, resp, err := p.client.Organizations.List(ctx, "", opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list organizations: %w", err)
		}

		for _, org := range orgs {
			if org.Login != nil {
				allOrgs = append(allOrgs, *org.Login)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allOrgs, nil
}
