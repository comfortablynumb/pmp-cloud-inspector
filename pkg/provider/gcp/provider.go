package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/functions/apiv1"
	"cloud.google.com/go/functions/apiv1/functionspb"
	"cloud.google.com/go/run/apiv2"
	"cloud.google.com/go/run/apiv2/runpb"
	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the GCP provider
type Provider struct {
	config            config.ProviderConfig
	projectID         string
	regions           []string
	computeClient     *compute.InstancesClient
	networksClient    *compute.NetworksClient
	subnetworksClient *compute.SubnetworksClient
	storageClient     *storage.Client
	functionsClient   *functions.CloudFunctionsClient
	runClient         *run.ServicesClient
}

// init registers the GCP provider
func init() {
	provider.Register("gcp", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "gcp"
}

// Initialize sets up the GCP provider
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Get project ID
	projectID, ok := cfg.Options["project_id"].(string)
	if !ok || projectID == "" {
		return fmt.Errorf("gcp project_id is required in provider options")
	}
	p.projectID = projectID

	// Get credentials file (optional, uses Application Default Credentials if not provided)
	var opts []option.ClientOption
	if credsFile, ok := cfg.Options["credentials_file"].(string); ok && credsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credsFile))
	}

	// Initialize clients
	var err error
	p.computeClient, err = compute.NewInstancesRESTClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create compute client: %w", err)
	}

	p.networksClient, err = compute.NewNetworksRESTClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create networks client: %w", err)
	}

	p.subnetworksClient, err = compute.NewSubnetworksRESTClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create subnetworks client: %w", err)
	}

	p.storageClient, err = storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}

	p.functionsClient, err = functions.NewCloudFunctionsClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create functions client: %w", err)
	}

	p.runClient, err = run.NewServicesClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create Cloud Run client: %w", err)
	}

	// Set up regions
	if len(cfg.Regions) > 0 {
		p.regions = cfg.Regions
	} else {
		p.regions = []string{
			"us-central1",
			"us-east1",
			"europe-west1",
		}
	}

	return nil
}

// GetSupportedResourceTypes returns all GCP resource types
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeGCPProject,
		resource.TypeGCPComputeInstance,
		resource.TypeGCPVPC,
		resource.TypeGCPSubnet,
		resource.TypeGCPStorageBucket,
		resource.TypeGCPCloudFunction,
		resource.TypeGCPCloudRun,
	}
}

// CollectResources collects GCP resources
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

	// Collect global resources
	if typeSet[resource.TypeGCPVPC] {
		if err := p.collectNetworks(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect VPCs: %w", err)
		}
	}

	if typeSet[resource.TypeGCPStorageBucket] {
		if err := p.collectStorageBuckets(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect storage buckets: %w", err)
		}
	}

	// Collect regional resources
	for _, region := range p.regions {
		if typeSet[resource.TypeGCPComputeInstance] {
			if err := p.collectComputeInstances(ctx, collection, region); err != nil {
				return nil, fmt.Errorf("failed to collect compute instances in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeGCPSubnet] {
			if err := p.collectSubnetworks(ctx, collection, region); err != nil {
				return nil, fmt.Errorf("failed to collect subnetworks in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeGCPCloudFunction] {
			if err := p.collectCloudFunctions(ctx, collection, region); err != nil {
				return nil, fmt.Errorf("failed to collect Cloud Functions in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeGCPCloudRun] {
			if err := p.collectCloudRunServices(ctx, collection, region); err != nil {
				return nil, fmt.Errorf("failed to collect Cloud Run services in %s: %w", region, err)
			}
		}
	}

	return collection, nil
}

// DiscoverRelationships establishes relationships between GCP resources
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Relationships can be discovered here
	return nil
}

// GetAccounts returns the project ID
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	return []string{p.projectID}, nil
}

// GetRegions returns configured regions
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	return p.regions, nil
}

// Close closes all clients
func (p *Provider) Close() error {
	if p.computeClient != nil {
		_ = p.computeClient.Close()
	}
	if p.networksClient != nil {
		_ = p.networksClient.Close()
	}
	if p.subnetworksClient != nil {
		_ = p.subnetworksClient.Close()
	}
	if p.storageClient != nil {
		_ = p.storageClient.Close()
	}
	if p.functionsClient != nil {
		_ = p.functionsClient.Close()
	}
	if p.runClient != nil {
		_ = p.runClient.Close()
	}
	return nil
}

// safeString safely dereferences a string pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// safeUint64 safely dereferences a uint64 pointer
func safeUint64(u *uint64) uint64 {
	if u == nil {
		return 0
	}
	return *u
}
