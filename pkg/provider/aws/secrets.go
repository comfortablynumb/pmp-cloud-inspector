package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	secretsTypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectSecrets collects all secrets in a region
func (p *Provider) collectSecrets(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting Secrets Manager secrets in %s...\n", region)
	client := secretsmanager.NewFromConfig(cfg)

	paginator := secretsmanager.NewListSecretsPaginator(client, &secretsmanager.ListSecretsInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range output.SecretList {
			res := p.convertSecretToResource(&secret, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found secret: %s\n", safeString(secret.Name))
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d secrets in %s\n", count, region)
	return nil
}

// convertSecretToResource converts a Secrets Manager secret to a Resource
func (p *Provider) convertSecretToResource(secret *secretsTypes.SecretListEntry, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{}

	if secret.Description != nil {
		properties["description"] = *secret.Description
	}
	if secret.RotationEnabled != nil {
		properties["rotation_enabled"] = *secret.RotationEnabled
	}
	if secret.RotationLambdaARN != nil {
		properties["rotation_lambda_arn"] = *secret.RotationLambdaARN
	}
	if secret.KmsKeyId != nil {
		properties["kms_key_id"] = *secret.KmsKeyId
	}
	if secret.LastRotatedDate != nil {
		properties["last_rotated_date"] = secret.LastRotatedDate.String()
	}
	if secret.LastChangedDate != nil {
		properties["last_changed_date"] = secret.LastChangedDate.String()
	}
	if secret.LastAccessedDate != nil {
		properties["last_accessed_date"] = secret.LastAccessedDate.String()
	}

	res := &resource.Resource{
		ID:         safeString(secret.ARN),
		Type:       resource.TypeAWSSecret,
		Name:       safeString(secret.Name),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        safeString(secret.ARN),
		Properties: properties,
		RawData:    secret,
	}

	if secret.CreatedDate != nil {
		res.CreatedAt = secret.CreatedDate
	}

	if secret.LastChangedDate != nil {
		res.UpdatedAt = secret.LastChangedDate
	}

	// Add KMS key relationship if present
	if secret.KmsKeyId != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationDependsOn,
			TargetID:   *secret.KmsKeyId,
			TargetType: "aws:kms:key", // Future resource type
		})
	}

	return res
}
