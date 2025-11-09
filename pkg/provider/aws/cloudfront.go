package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	cloudfrontTypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectCloudFrontDistributions collects all CloudFront distributions (global service)
func (p *Provider) collectCloudFrontDistributions(ctx context.Context, collection *resource.Collection, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting CloudFront distributions (global)...\n")
	client := cloudfront.NewFromConfig(cfg)

	paginator := cloudfront.NewListDistributionsPaginator(client, &cloudfront.ListDistributionsInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list CloudFront distributions: %w", err)
		}

		if output.DistributionList != nil {
			for _, dist := range output.DistributionList.Items {
				res := p.convertCloudFrontDistributionToResource(&dist)
				collection.Add(res)
				count++
				fmt.Fprintf(os.Stderr, "    Found CloudFront distribution: %s (%s)\n", safeString(dist.Id), safeString(dist.DomainName))
			}
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d CloudFront distributions\n", count)
	return nil
}

// convertCloudFrontDistributionToResource converts a CloudFront distribution to a Resource
func (p *Provider) convertCloudFrontDistributionToResource(dist *cloudfrontTypes.DistributionSummary) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"domain_name": safeString(dist.DomainName),
		"status":      safeString(dist.Status),
		"enabled":     safeBool(dist.Enabled),
	}

	if dist.Comment != nil {
		properties["comment"] = *dist.Comment
	}
	if dist.PriceClass != "" {
		properties["price_class"] = string(dist.PriceClass)
	}
	if dist.HttpVersion != "" {
		properties["http_version"] = string(dist.HttpVersion)
	}
	if dist.IsIPV6Enabled != nil {
		properties["ipv6_enabled"] = *dist.IsIPV6Enabled
	}
	if dist.WebACLId != nil {
		properties["web_acl_id"] = *dist.WebACLId
	}

	res := &resource.Resource{
		ID:         safeString(dist.Id),
		Type:       resource.TypeAWSCloudFront,
		Name:       safeString(dist.Id),
		Provider:   "aws",
		Account:    account,
		Region:     "global", // CloudFront is a global service
		ARN:        safeString(dist.ARN),
		Properties: properties,
		RawData:    dist,
	}

	if dist.LastModifiedTime != nil {
		res.UpdatedAt = dist.LastModifiedTime
	}

	return res
}

// safeBool safely dereferences a bool pointer
func safeBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
