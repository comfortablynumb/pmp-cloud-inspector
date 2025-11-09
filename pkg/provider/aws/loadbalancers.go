package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing"
	elbTypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2Types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectClassicLoadBalancers collects all classic load balancers in a region
func (p *Provider) collectClassicLoadBalancers(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting Classic Load Balancers in %s...\n", region)
	client := elasticloadbalancing.NewFromConfig(cfg)

	paginator := elasticloadbalancing.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancing.DescribeLoadBalancersInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe classic load balancers: %w", err)
		}

		for _, lb := range output.LoadBalancerDescriptions {
			res := p.convertClassicLoadBalancerToResource(&lb, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found Classic ELB: %s\n", safeString(lb.LoadBalancerName))
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d Classic Load Balancers in %s\n", count, region)
	return nil
}

// collectLoadBalancersV2 collects all ALBs and NLBs in a region
func (p *Provider) collectLoadBalancersV2(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting ALBs/NLBs in %s...\n", region)
	client := elasticloadbalancingv2.NewFromConfig(cfg)

	paginator := elasticloadbalancingv2.NewDescribeLoadBalancersPaginator(client, &elasticloadbalancingv2.DescribeLoadBalancersInput{})

	albCount := 0
	nlbCount := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe load balancers v2: %w", err)
		}

		for _, lb := range output.LoadBalancers {
			res := p.convertLoadBalancerV2ToResource(&lb, region)
			collection.Add(res)

			if lb.Type == elbv2Types.LoadBalancerTypeEnumApplication {
				albCount++
				fmt.Fprintf(os.Stderr, "    Found ALB: %s\n", safeString(lb.LoadBalancerName))
			} else if lb.Type == elbv2Types.LoadBalancerTypeEnumNetwork {
				nlbCount++
				fmt.Fprintf(os.Stderr, "    Found NLB: %s\n", safeString(lb.LoadBalancerName))
			}
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d ALBs and %d NLBs in %s\n", albCount, nlbCount, region)
	return nil
}

// convertClassicLoadBalancerToResource converts a classic ELB to a Resource
func (p *Provider) convertClassicLoadBalancerToResource(lb *elbTypes.LoadBalancerDescription, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"dns_name": safeString(lb.DNSName),
		"scheme":   safeString(lb.Scheme),
	}

	if lb.VPCId != nil {
		properties["vpc_id"] = *lb.VPCId
	}
	if len(lb.AvailabilityZones) > 0 {
		properties["availability_zones"] = lb.AvailabilityZones
	}
	if len(lb.Subnets) > 0 {
		properties["subnets"] = lb.Subnets
	}
	if len(lb.Instances) > 0 {
		instanceIDs := make([]string, len(lb.Instances))
		for i, inst := range lb.Instances {
			instanceIDs[i] = safeString(inst.InstanceId)
		}
		properties["instances"] = instanceIDs
	}

	arn := fmt.Sprintf("arn:aws:elasticloadbalancing:%s:%s:loadbalancer/%s", region, account, safeString(lb.LoadBalancerName))

	res := &resource.Resource{
		ID:         safeString(lb.LoadBalancerName),
		Type:       resource.TypeAWSELB,
		Name:       safeString(lb.LoadBalancerName),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Properties: properties,
		RawData:    lb,
	}

	// Add VPC relationship
	if lb.VPCId != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   *lb.VPCId,
			TargetType: resource.TypeAWSVPC,
		})
	}

	return res
}

// convertLoadBalancerV2ToResource converts an ALB or NLB to a Resource
func (p *Provider) convertLoadBalancerV2ToResource(lb *elbv2Types.LoadBalancer, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	var resourceType resource.ResourceType
	if lb.Type == elbv2Types.LoadBalancerTypeEnumApplication {
		resourceType = resource.TypeAWSALB
	} else if lb.Type == elbv2Types.LoadBalancerTypeEnumNetwork {
		resourceType = resource.TypeAWSNLB
	} else {
		resourceType = resource.TypeAWSALB // default
	}

	properties := map[string]interface{}{
		"dns_name": safeString(lb.DNSName),
		"type":     string(lb.Type),
		"scheme":   string(lb.Scheme),
		"state":    string(lb.State.Code),
	}

	if lb.VpcId != nil {
		properties["vpc_id"] = *lb.VpcId
	}
	if len(lb.AvailabilityZones) > 0 {
		azs := make([]string, len(lb.AvailabilityZones))
		for i, az := range lb.AvailabilityZones {
			azs[i] = safeString(az.ZoneName)
		}
		properties["availability_zones"] = azs
	}
	if lb.IpAddressType != "" {
		properties["ip_address_type"] = string(lb.IpAddressType)
	}

	res := &resource.Resource{
		ID:         safeString(lb.LoadBalancerArn),
		Type:       resourceType,
		Name:       safeString(lb.LoadBalancerName),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        safeString(lb.LoadBalancerArn),
		Properties: properties,
		RawData:    lb,
	}

	// Add VPC relationship
	if lb.VpcId != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   *lb.VpcId,
			TargetType: resource.TypeAWSVPC,
		})
	}

	if lb.CreatedTime != nil {
		res.CreatedAt = lb.CreatedTime
	}

	return res
}
