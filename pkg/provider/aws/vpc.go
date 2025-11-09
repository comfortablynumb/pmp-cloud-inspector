package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectVPCs collects all VPCs in a region
func (p *Provider) collectVPCs(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return fmt.Errorf("failed to describe VPCs: %w", err)
	}

	for _, vpc := range result.Vpcs {
		res := p.convertVPCToResource(&vpc, region)
		collection.Add(res)
	}

	return nil
}

// collectSubnets collects all subnets in a region
func (p *Provider) collectSubnets(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{})
	if err != nil {
		return fmt.Errorf("failed to describe subnets: %w", err)
	}

	for _, subnet := range result.Subnets {
		res := p.convertSubnetToResource(&subnet, region)
		collection.Add(res)
	}

	return nil
}

// collectSecurityGroups collects all security groups in a region
func (p *Provider) collectSecurityGroups(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	client := ec2.NewFromConfig(cfg)

	result, err := client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return fmt.Errorf("failed to describe security groups: %w", err)
	}

	for _, sg := range result.SecurityGroups {
		res := p.convertSecurityGroupToResource(&sg, region)
		collection.Add(res)
	}

	return nil
}

// convertVPCToResource converts a VPC to a Resource
func (p *Provider) convertVPCToResource(vpc *ec2Types.Vpc, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"cidr_block": safeString(vpc.CidrBlock),
		"state":      string(vpc.State),
	}

	if vpc.IsDefault != nil {
		properties["is_default"] = *vpc.IsDefault
	}
	if vpc.DhcpOptionsId != nil {
		properties["dhcp_options_id"] = *vpc.DhcpOptionsId
	}
	if vpc.InstanceTenancy != "" {
		properties["instance_tenancy"] = string(vpc.InstanceTenancy)
	}

	var tags map[string]string
	var name string
	if len(vpc.Tags) > 0 {
		tags = make(map[string]string)
		for _, tag := range vpc.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}
		}
	}

	if name == "" {
		name = safeString(vpc.VpcId)
	}

	arn := fmt.Sprintf("arn:aws:ec2:%s:%s:vpc/%s", region, account, safeString(vpc.VpcId))

	return &resource.Resource{
		ID:         safeString(vpc.VpcId),
		Type:       resource.TypeAWSVPC,
		Name:       name,
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Tags:       tags,
		Properties: properties,
		RawData:    vpc,
	}
}

// convertSubnetToResource converts a Subnet to a Resource
func (p *Provider) convertSubnetToResource(subnet *ec2Types.Subnet, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"cidr_block":              safeString(subnet.CidrBlock),
		"state":                   string(subnet.State),
		"available_ip_count":      subnet.AvailableIpAddressCount,
		"map_public_ip_on_launch": subnet.MapPublicIpOnLaunch,
	}

	if subnet.AvailabilityZone != nil {
		properties["availability_zone"] = *subnet.AvailabilityZone
	}
	if subnet.AvailabilityZoneId != nil {
		properties["availability_zone_id"] = *subnet.AvailabilityZoneId
	}

	var tags map[string]string
	var name string
	if len(subnet.Tags) > 0 {
		tags = make(map[string]string)
		for _, tag := range subnet.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}
		}
	}

	if name == "" {
		name = safeString(subnet.SubnetId)
	}

	arn := fmt.Sprintf("arn:aws:ec2:%s:%s:subnet/%s", region, account, safeString(subnet.SubnetId))

	res := &resource.Resource{
		ID:         safeString(subnet.SubnetId),
		Type:       resource.TypeAWSSubnet,
		Name:       name,
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Tags:       tags,
		Properties: properties,
		RawData:    subnet,
	}

	// Add VPC relationship
	if subnet.VpcId != nil {
		res.Relationships = []resource.Relationship{
			{
				Type:       resource.RelationBelongsTo,
				TargetID:   *subnet.VpcId,
				TargetType: resource.TypeAWSVPC,
				Properties: map[string]interface{}{
					"vpc_id": *subnet.VpcId,
				},
			},
		}
	}

	return res
}

// convertSecurityGroupToResource converts a Security Group to a Resource
func (p *Provider) convertSecurityGroupToResource(sg *ec2Types.SecurityGroup, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"description": safeString(sg.Description),
	}

	// Add ingress rules
	if len(sg.IpPermissions) > 0 {
		ingressRules := make([]map[string]interface{}, 0, len(sg.IpPermissions))
		for _, perm := range sg.IpPermissions {
			rule := map[string]interface{}{
				"ip_protocol": safeString(perm.IpProtocol),
			}
			if perm.FromPort != nil {
				rule["from_port"] = *perm.FromPort
			}
			if perm.ToPort != nil {
				rule["to_port"] = *perm.ToPort
			}
			if len(perm.IpRanges) > 0 {
				cidrs := make([]string, 0, len(perm.IpRanges))
				for _, ipRange := range perm.IpRanges {
					cidrs = append(cidrs, safeString(ipRange.CidrIp))
				}
				rule["cidr_blocks"] = cidrs
			}
			ingressRules = append(ingressRules, rule)
		}
		properties["ingress_rules"] = ingressRules
	}

	// Add egress rules
	if len(sg.IpPermissionsEgress) > 0 {
		egressRules := make([]map[string]interface{}, 0, len(sg.IpPermissionsEgress))
		for _, perm := range sg.IpPermissionsEgress {
			rule := map[string]interface{}{
				"ip_protocol": safeString(perm.IpProtocol),
			}
			if perm.FromPort != nil {
				rule["from_port"] = *perm.FromPort
			}
			if perm.ToPort != nil {
				rule["to_port"] = *perm.ToPort
			}
			if len(perm.IpRanges) > 0 {
				cidrs := make([]string, 0, len(perm.IpRanges))
				for _, ipRange := range perm.IpRanges {
					cidrs = append(cidrs, safeString(ipRange.CidrIp))
				}
				rule["cidr_blocks"] = cidrs
			}
			egressRules = append(egressRules, rule)
		}
		properties["egress_rules"] = egressRules
	}

	var tags map[string]string
	var name string
	if len(sg.Tags) > 0 {
		tags = make(map[string]string)
		for _, tag := range sg.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}
		}
	}

	if name == "" {
		name = safeString(sg.GroupName)
	}

	arn := fmt.Sprintf("arn:aws:ec2:%s:%s:security-group/%s", region, account, safeString(sg.GroupId))

	res := &resource.Resource{
		ID:         safeString(sg.GroupId),
		Type:       resource.TypeAWSSecurityGroup,
		Name:       name,
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Tags:       tags,
		Properties: properties,
		RawData:    sg,
	}

	// Add VPC relationship
	if sg.VpcId != nil {
		res.Relationships = []resource.Relationship{
			{
				Type:       resource.RelationBelongsTo,
				TargetID:   *sg.VpcId,
				TargetType: resource.TypeAWSVPC,
				Properties: map[string]interface{}{
					"vpc_id": *sg.VpcId,
				},
			},
		}
	}

	return res
}
