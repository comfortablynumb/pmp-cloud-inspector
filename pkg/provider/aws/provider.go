package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Provider implements the AWS cloud provider
type Provider struct {
	config    config.ProviderConfig
	awsConfig aws.Config

	// AWS service clients
	iamClient *iam.Client
	ec2Client *ec2.Client
	ecrClient *ecr.Client
	stsClient *sts.Client

	// Configuration
	accounts []string
	regions  []string
}

// init registers the AWS provider
func init() {
	provider.Register("aws", func() provider.Provider {
		return &Provider{}
	})
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "aws"
}

// Initialize sets up the AWS provider with credentials and configuration
func (p *Provider) Initialize(ctx context.Context, cfg config.ProviderConfig) error {
	p.config = cfg

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	p.awsConfig = awsCfg

	// Initialize service clients
	p.iamClient = iam.NewFromConfig(awsCfg)
	p.ec2Client = ec2.NewFromConfig(awsCfg)
	p.ecrClient = ecr.NewFromConfig(awsCfg)
	p.stsClient = sts.NewFromConfig(awsCfg)

	// Set up regions
	if len(cfg.Regions) > 0 {
		p.regions = cfg.Regions
	} else {
		// Default to common regions or discover them
		p.regions = []string{
			"us-east-1",
			"us-west-2",
			"eu-west-1",
		}
	}

	// Set up accounts
	if len(cfg.Accounts) > 0 {
		p.accounts = cfg.Accounts
	} else {
		// Get current account
		accounts, err := p.GetAccounts(ctx)
		if err != nil {
			return fmt.Errorf("failed to get accounts: %w", err)
		}
		p.accounts = accounts
	}

	return nil
}

// GetSupportedResourceTypes returns all AWS resource types supported
func (p *Provider) GetSupportedResourceTypes() []resource.ResourceType {
	return []resource.ResourceType{
		resource.TypeAWSIAMUser,
		resource.TypeAWSIAMRole,
		resource.TypeAWSAccount,
		resource.TypeAWSVPC,
		resource.TypeAWSSubnet,
		resource.TypeAWSSecurityGroup,
		resource.TypeAWSECR,
	}
}

// CollectResources collects all specified AWS resources
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

	// Collect IAM resources (global, not regional)
	if typeSet[resource.TypeAWSIAMUser] {
		if err := p.collectIAMUsers(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect IAM users: %w", err)
		}
	}

	if typeSet[resource.TypeAWSIAMRole] {
		if err := p.collectIAMRoles(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect IAM roles: %w", err)
		}
	}

	if typeSet[resource.TypeAWSAccount] {
		if err := p.collectAccounts(ctx, collection); err != nil {
			return nil, fmt.Errorf("failed to collect accounts: %w", err)
		}
	}

	// Collect regional resources
	for _, region := range p.regions {
		regionalConfig := p.awsConfig.Copy()
		regionalConfig.Region = region

		if typeSet[resource.TypeAWSVPC] {
			if err := p.collectVPCs(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect VPCs in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSSubnet] {
			if err := p.collectSubnets(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect subnets in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSSecurityGroup] {
			if err := p.collectSecurityGroups(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect security groups in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSECR] {
			if err := p.collectECRRepositories(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect ECR repositories in %s: %w", region, err)
			}
		}
	}

	return collection, nil
}

// DiscoverRelationships establishes relationships between AWS resources
func (p *Provider) DiscoverRelationships(ctx context.Context, collection *resource.Collection) error {
	// Build relationships based on AWS resource structure
	for _, res := range collection.Resources {
		switch res.Type {
		case resource.TypeAWSSubnet:
			p.discoverSubnetRelationships(res, collection)
		case resource.TypeAWSSecurityGroup:
			p.discoverSecurityGroupRelationships(res, collection)
		}
	}

	return nil
}

// GetAccounts returns the AWS account ID(s)
func (p *Provider) GetAccounts(ctx context.Context) ([]string, error) {
	if p.stsClient == nil {
		return nil, fmt.Errorf("STS client not initialized")
	}

	result, err := p.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get caller identity: %w", err)
	}

	if result.Account != nil {
		return []string{*result.Account}, nil
	}

	return []string{}, nil
}

// GetRegions returns available AWS regions
func (p *Provider) GetRegions(ctx context.Context) ([]string, error) {
	if p.ec2Client == nil {
		return p.regions, nil
	}

	result, err := p.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	regions := make([]string, 0, len(result.Regions))
	for _, region := range result.Regions {
		if region.RegionName != nil {
			regions = append(regions, *region.RegionName)
		}
	}

	return regions, nil
}
