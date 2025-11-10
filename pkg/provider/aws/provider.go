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
		resource.TypeAWSEC2Instance,
		resource.TypeAWSECR,
		resource.TypeAWSEKSCluster,
		resource.TypeAWSELB,
		resource.TypeAWSALB,
		resource.TypeAWSNLB,
		resource.TypeAWSLambda,
		resource.TypeAWSAPIGateway,
		resource.TypeAWSCloudFront,
		resource.TypeAWSMemoryDB,
		resource.TypeAWSElastiCache,
		resource.TypeAWSSecret,
		resource.TypeAWSSNSTopic,
		resource.TypeAWSSQSQueue,
		resource.TypeAWSDynamoDBTable,
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
		p.collectAccounts(collection)
	}

	// Collect global resources (CloudFront)
	if typeSet[resource.TypeAWSCloudFront] {
		if err := p.collectCloudFrontDistributions(ctx, collection, p.awsConfig); err != nil {
			return nil, fmt.Errorf("failed to collect CloudFront distributions: %w", err)
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

		if typeSet[resource.TypeAWSEC2Instance] {
			if err := p.collectEC2Instances(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect EC2 instances in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSECR] {
			if err := p.collectECRRepositories(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect ECR repositories in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSEKSCluster] {
			if err := p.collectEKSClusters(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect EKS clusters in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSELB] {
			if err := p.collectClassicLoadBalancers(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect ELBs in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSALB] || typeSet[resource.TypeAWSNLB] {
			if err := p.collectLoadBalancersV2(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect ALBs/NLBs in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSLambda] {
			if err := p.collectLambdaFunctions(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect Lambda functions in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSAPIGateway] {
			if err := p.collectAPIGatewayAPIs(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect API Gateways in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSMemoryDB] {
			if err := p.collectMemoryDBClusters(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect MemoryDB clusters in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSElastiCache] {
			if err := p.collectElastiCacheClusters(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect ElastiCache clusters in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSSecret] {
			if err := p.collectSecrets(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect secrets in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSSNSTopic] {
			if err := p.collectSNSTopics(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect SNS topics in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSSQSQueue] {
			if err := p.collectSQSQueues(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect SQS queues in %s: %w", region, err)
			}
		}

		if typeSet[resource.TypeAWSDynamoDBTable] {
			if err := p.collectDynamoDBTables(ctx, collection, region, regionalConfig); err != nil {
				return nil, fmt.Errorf("failed to collect DynamoDB tables in %s: %w", region, err)
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
		case resource.TypeAWSVPC:
			p.discoverVPCRelationships(res, collection)
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
