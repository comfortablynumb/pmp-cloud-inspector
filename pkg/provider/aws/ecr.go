package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrTypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectECRRepositories collects all ECR repositories in a region
func (p *Provider) collectECRRepositories(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	client := ecr.NewFromConfig(cfg)

	paginator := ecr.NewDescribeRepositoriesPaginator(client, &ecr.DescribeRepositoriesInput{})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe ECR repositories: %w", err)
		}

		for _, repo := range output.Repositories {
			res := p.convertECRRepositoryToResource(&repo, region)
			collection.Add(res)
		}
	}

	return nil
}

// convertECRRepositoryToResource converts an ECR repository to a Resource
func (p *Provider) convertECRRepositoryToResource(repo *ecrTypes.Repository, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"registry_id":    safeString(repo.RegistryId),
		"repository_uri": safeString(repo.RepositoryUri),
	}

	if repo.CreatedAt != nil {
		properties["created_at"] = repo.CreatedAt.Format(time.RFC3339)
	}

	if repo.ImageTagMutability != "" {
		properties["image_tag_mutability"] = string(repo.ImageTagMutability)
	}

	if repo.ImageScanningConfiguration != nil {
		properties["image_scanning"] = map[string]interface{}{
			"scan_on_push": repo.ImageScanningConfiguration.ScanOnPush,
		}
	}

	if repo.EncryptionConfiguration != nil {
		encConfig := map[string]interface{}{
			"encryption_type": string(repo.EncryptionConfiguration.EncryptionType),
		}
		if repo.EncryptionConfiguration.KmsKey != nil {
			encConfig["kms_key"] = *repo.EncryptionConfiguration.KmsKey
		}
		properties["encryption"] = encConfig
	}

	var createdAt *time.Time
	if repo.CreatedAt != nil {
		createdAt = repo.CreatedAt
	}

	return &resource.Resource{
		ID:         safeString(repo.RepositoryArn),
		Type:       resource.TypeAWSECR,
		Name:       safeString(repo.RepositoryName),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        safeString(repo.RepositoryArn),
		Properties: properties,
		RawData:    repo,
		CreatedAt:  createdAt,
	}
}
