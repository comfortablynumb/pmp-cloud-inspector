package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	apigwTypes "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/aws/aws-sdk-go-v2/service/apigatewayv2"
	apigwv2Types "github.com/aws/aws-sdk-go-v2/service/apigatewayv2/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectAPIGatewayAPIs collects both REST APIs (v1) and HTTP APIs (v2) in a region
func (p *Provider) collectAPIGatewayAPIs(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	// Collect REST APIs (v1)
	if err := p.collectRESTAPIs(ctx, collection, region, cfg); err != nil {
		return err
	}

	// Collect HTTP APIs (v2)
	if err := p.collectHTTPAPIs(ctx, collection, region, cfg); err != nil {
		return err
	}

	return nil
}

// collectRESTAPIs collects REST APIs (API Gateway v1)
func (p *Provider) collectRESTAPIs(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting API Gateway REST APIs in %s...\n", region)
	client := apigateway.NewFromConfig(cfg)

	paginator := apigateway.NewGetRestApisPaginator(client, &apigateway.GetRestApisInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe REST APIs: %w", err)
		}

		for _, api := range output.Items {
			res := p.convertRESTAPIToResource(&api, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found REST API: %s (%s)\n", safeString(api.Name), safeString(api.Id))
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d REST APIs in %s\n", count, region)
	return nil
}

// collectHTTPAPIs collects HTTP APIs (API Gateway v2)
func (p *Provider) collectHTTPAPIs(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting API Gateway HTTP APIs in %s...\n", region)
	client := apigatewayv2.NewFromConfig(cfg)

	var nextToken *string
	count := 0

	for {
		input := &apigatewayv2.GetApisInput{
			NextToken: nextToken,
		}

		output, err := client.GetApis(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to describe HTTP APIs: %w", err)
		}

		for _, api := range output.Items {
			res := p.convertHTTPAPIToResource(&api, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found HTTP API: %s (%s)\n", safeString(api.Name), safeString(api.ApiId))
		}

		if output.NextToken == nil || *output.NextToken == "" {
			break
		}
		nextToken = output.NextToken
	}

	fmt.Fprintf(os.Stderr, "  Collected %d HTTP APIs in %s\n", count, region)
	return nil
}

// convertRESTAPIToResource converts a REST API to a Resource
func (p *Provider) convertRESTAPIToResource(api *apigwTypes.RestApi, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{}

	if api.Description != nil {
		properties["description"] = *api.Description
	}
	if api.Version != nil {
		properties["version"] = *api.Version
	}
	if len(api.EndpointConfiguration.Types) > 0 {
		properties["endpoint_types"] = api.EndpointConfiguration.Types
	}
	if api.ApiKeySource != "" {
		properties["api_key_source"] = string(api.ApiKeySource)
	}

	arn := fmt.Sprintf("arn:aws:apigateway:%s::/restapis/%s", region, safeString(api.Id))

	res := &resource.Resource{
		ID:         safeString(api.Id),
		Type:       resource.TypeAWSAPIGateway,
		Name:       safeString(api.Name),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Properties: properties,
		RawData:    api,
	}

	if api.CreatedDate != nil {
		res.CreatedAt = api.CreatedDate
	}

	return res
}

// convertHTTPAPIToResource converts an HTTP API to a Resource
func (p *Provider) convertHTTPAPIToResource(api *apigwv2Types.Api, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"protocol_type": string(api.ProtocolType),
	}

	if api.Description != nil {
		properties["description"] = *api.Description
	}
	if api.Version != nil {
		properties["version"] = *api.Version
	}
	if api.ApiEndpoint != nil {
		properties["api_endpoint"] = *api.ApiEndpoint
	}
	if api.RouteSelectionExpression != nil {
		properties["route_selection_expression"] = *api.RouteSelectionExpression
	}

	arn := fmt.Sprintf("arn:aws:apigateway:%s::/apis/%s", region, safeString(api.ApiId))

	res := &resource.Resource{
		ID:         safeString(api.ApiId),
		Type:       resource.TypeAWSAPIGateway,
		Name:       safeString(api.Name),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Properties: properties,
		RawData:    api,
	}

	if api.CreatedDate != nil {
		res.CreatedAt = api.CreatedDate
	}

	return res
}
