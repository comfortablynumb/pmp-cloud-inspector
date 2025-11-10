//go:build azure
// +build azure

package azure

import (
	"context"
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/ratelimit"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the Azure cloud provider
type Provider struct {
	config         config.ProviderConfig
	credential     *azidentity.DefaultAzureCredential
	subscriptionID string
	rateLimiter    *ratelimit.Limiter
}

// init registers the Azure provider
func init() {
	provider.Register("azure", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "azure"
}

// Initialize sets up the Azure provider with credentials and configuration
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Get Azure subscription ID from environment variable
	p.subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if p.subscriptionID == "" {
		return fmt.Errorf("AZURE_SUBSCRIPTION_ID environment variable is required")
	}

	// Create credential using DefaultAzureCredential
	// This supports multiple authentication methods:
	// - Environment variables (AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET)
	// - Managed Identity
	// - Azure CLI authentication
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return fmt.Errorf("failed to create Azure credential: %w", err)
	}

	p.credential = cred

	// Initialize rate limiter
	p.rateLimiter = ratelimit.NewFromMilliseconds(cfg.RateLimitMs)

	return nil
}

// GetSupportedResourceTypes returns all Azure resource types supported
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeAzureResourceGroup,
		resource.TypeAzureVM,
		resource.TypeAzureVNet,
		resource.TypeAzureSubnet,
		resource.TypeAzureStorageAccount,
		resource.TypeAzureAppService,
		resource.TypeAzureSQLDatabase,
		resource.TypeAzureKeyVault,
	}
}

// CollectResources collects all specified Azure resources
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

	// Collect Resource Groups first (many other resources depend on them)
	if typeSet[resource.TypeAzureResourceGroup] {
		if err := p.collectResourceGroups(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect resource groups: %w", err)
		}
		if err := p.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Collect VMs
	if typeSet[resource.TypeAzureVM] {
		if err := p.collectVirtualMachines(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect virtual machines: %w", err)
		}
		if err := p.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Collect Virtual Networks
	if typeSet[resource.TypeAzureVNet] {
		if err := p.collectVirtualNetworks(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect virtual networks: %w", err)
		}
		if err := p.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Collect Storage Accounts
	if typeSet[resource.TypeAzureStorageAccount] {
		if err := p.collectStorageAccounts(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect storage accounts: %w", err)
		}
		if err := p.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Collect App Services
	if typeSet[resource.TypeAzureAppService] {
		if err := p.collectAppServices(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect app services: %w", err)
		}
		if err := p.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Collect SQL Databases
	if typeSet[resource.TypeAzureSQLDatabase] {
		if err := p.collectSQLDatabases(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect SQL databases: %w", err)
		}
		if err := p.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	// Collect Key Vaults
	if typeSet[resource.TypeAzureKeyVault] {
		if err := p.collectKeyVaults(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect key vaults: %w", err)
		}
		if err := p.rateLimiter.Wait(ctx); err != nil {
			return nil, err
		}
	}

	return collection, nil
}

// DiscoverRelationships establishes relationships between Azure resources
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Build relationships based on Azure resource structure
	// Example: VMs -> VNets, Subnets, Resource Groups, etc.
	for _, res := range collection.Resources {
		switch res.Type {
		case resource.TypeAzureVM:
			p.discoverVMRelationships(res, collection)
		case resource.TypeAzureVNet:
			p.discoverVNetRelationships(res, collection)
		}
	}

	return nil
}

// GetAccounts returns the Azure subscription ID
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	return []string{p.subscriptionID}, nil
}

// GetRegions returns available Azure regions
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	// Get locations from resource providers client
	client, err := armresources.NewProvidersClient(p.subscriptionID, p.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create providers client: %w", err)
	}

	// Get locations for Microsoft.Compute provider as a reference
	pager := client.NewListPager(nil)
	locations := make(map[string]bool)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get providers: %w", err)
		}

		for _, provider := range page.Value {
			if provider.Namespace != nil && *provider.Namespace == "Microsoft.Compute" {
				for _, resourceType := range provider.ResourceTypes {
					if resourceType.Locations != nil {
						for _, location := range resourceType.Locations {
							if location != nil {
								locations[*location] = true
							}
						}
					}
				}
				break
			}
		}
	}

	regionList := make([]string, 0, len(locations))
	for location := range locations {
		regionList = append(regionList, location)
	}

	return regionList, nil
}

// Helper functions for relationship discovery

func (p *Provider) discoverVMRelationships(vm *resource.Resource, collection *resource.Collection) {
	// Find resource group relationship
	if rgName, ok := vm.Properties["resource_group"].(string); ok {
		for _, res := range collection.Resources {
			if res.Type == resource.TypeAzureResourceGroup && res.Name == rgName {
				vm.Relationships = append(vm.Relationships, resource.Relationship{
					Type:       "in_resource_group",
					TargetID:   res.ID,
					TargetType: resource.TypeAzureResourceGroup,
				})
				break
			}
		}
	}
}

func (p *Provider) discoverVNetRelationships(vnet *resource.Resource, collection *resource.Collection) {
	// Find resource group relationship
	if rgName, ok := vnet.Properties["resource_group"].(string); ok {
		for _, res := range collection.Resources {
			if res.Type == resource.TypeAzureResourceGroup && res.Name == rgName {
				vnet.Relationships = append(vnet.Relationships, resource.Relationship{
					Type:       "in_resource_group",
					TargetID:   res.ID,
					TargetType: resource.TypeAzureResourceGroup,
				})
				break
			}
		}
	}
}
