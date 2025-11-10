//go:build jfrog
// +build jfrog

package jfrog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the JFrog Artifactory provider
type Provider struct {
	config   config.ProviderConfig
	baseURL  string
	username string
	password string
	apiKey   string
	client   *http.Client
}

// init registers the JFrog provider
func init() {
	provider.Register("jfrog", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "jfrog"
}

// Initialize sets up the JFrog provider
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Get base URL from environment variable
	p.baseURL = os.Getenv("JFROG_BASE_URL")
	if p.baseURL == "" {
		return fmt.Errorf("JFROG_BASE_URL environment variable is required")
	}

	// Authentication: API key or username/password from environment variables
	p.apiKey = os.Getenv("JFROG_API_KEY")
	if p.apiKey == "" {
		// Try username/password authentication
		p.username = os.Getenv("JFROG_USERNAME")
		p.password = os.Getenv("JFROG_PASSWORD")
		if p.username == "" || p.password == "" {
			return fmt.Errorf("JFROG_API_KEY or (JFROG_USERNAME and JFROG_PASSWORD) environment variables are required")
		}
	}

	p.client = &http.Client{
		Timeout: 30 * time.Second,
	}

	return nil
}

// GetSupportedResourceTypes returns all JFrog resource types
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeJFrogRepository,
		resource.TypeJFrogUser,
		resource.TypeJFrogGroup,
		resource.TypeJFrogPermission,
	}
}

// CollectResources collects JFrog resources
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

	// Collect repositories
	if typeSet[resource.TypeJFrogRepository] {
		if err := p.collectRepositories(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect repositories: %w", err)
		}
	}

	// Collect users
	if typeSet[resource.TypeJFrogUser] {
		if err := p.collectUsers(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect users: %w", err)
		}
	}

	// Collect groups
	if typeSet[resource.TypeJFrogGroup] {
		if err := p.collectGroups(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect groups: %w", err)
		}
	}

	// Collect permissions
	if typeSet[resource.TypeJFrogPermission] {
		if err := p.collectPermissions(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect permissions: %w", err)
		}
	}

	return collection, nil
}

// DiscoverRelationships establishes relationships between JFrog resources
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Relationships can be discovered here if needed
	return nil
}

// GetAccounts returns empty slice (JFrog doesn't have accounts concept)
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// GetRegions returns empty slice (JFrog doesn't have regions)
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

// doRequest performs an authenticated HTTP request
func (p *Provider) doRequest(method, path string) (*http.Response, error) {
	url := fmt.Sprintf("%s/artifactory/api/%s", p.baseURL, path)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	// Set authentication
	if p.apiKey != "" {
		req.Header.Set("X-JFrog-Art-Api", p.apiKey)
	} else {
		req.SetBasicAuth(p.username, p.password)
	}

	req.Header.Set("Content-Type", "application/json")

	return p.client.Do(req)
}

// parseResponse parses JSON response
func parseResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(v)
}
