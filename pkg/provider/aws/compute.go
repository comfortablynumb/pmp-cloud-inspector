package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectEC2Instances collects all EC2 instances in a region
func (p *Provider) collectEC2Instances(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting EC2 instances in %s...\n", region)
	client := ec2.NewFromConfig(cfg)

	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to describe instances: %w", err)
		}

		for _, reservation := range output.Reservations {
			for _, instance := range reservation.Instances {
				res := p.convertEC2InstanceToResource(&instance, region)
				collection.Add(res)
				count++
				fmt.Fprintf(os.Stderr, "    Found EC2 instance: %s (%s)\n", safeString(instance.InstanceId), instance.State.Name)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d EC2 instances in %s\n", count, region)
	return nil
}

// collectEKSClusters collects all EKS clusters in a region
func (p *Provider) collectEKSClusters(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting EKS clusters in %s...\n", region)
	client := eks.NewFromConfig(cfg)

	paginator := eks.NewListClustersPaginator(client, &eks.ListClustersInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list EKS clusters: %w", err)
		}

		for _, clusterName := range output.Clusters {
			// Get detailed cluster information
			describeOutput, err := client.DescribeCluster(ctx, &eks.DescribeClusterInput{
				Name: &clusterName,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: failed to describe EKS cluster %s: %v\n", clusterName, err)
				continue
			}

			res := p.convertEKSClusterToResource(describeOutput.Cluster, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found EKS cluster: %s\n", clusterName)
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d EKS clusters in %s\n", count, region)
	return nil
}

// collectLambdaFunctions collects all Lambda functions in a region
func (p *Provider) collectLambdaFunctions(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting Lambda functions in %s...\n", region)
	client := lambda.NewFromConfig(cfg)

	paginator := lambda.NewListFunctionsPaginator(client, &lambda.ListFunctionsInput{})

	count := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list Lambda functions: %w", err)
		}

		for _, function := range output.Functions {
			res := p.convertLambdaFunctionToResource(&function, region)
			collection.Add(res)
			count++
			fmt.Fprintf(os.Stderr, "    Found Lambda function: %s\n", safeString(function.FunctionName))
		}
	}

	fmt.Fprintf(os.Stderr, "  Collected %d Lambda functions in %s\n", count, region)
	return nil
}

// convertEC2InstanceToResource converts an EC2 instance to a Resource
func (p *Provider) convertEC2InstanceToResource(instance *ec2Types.Instance, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"instance_type": string(instance.InstanceType),
		"state":         string(instance.State.Name),
	}

	if instance.PrivateIpAddress != nil {
		properties["private_ip"] = *instance.PrivateIpAddress
	}
	if instance.PublicIpAddress != nil {
		properties["public_ip"] = *instance.PublicIpAddress
	}
	if instance.VpcId != nil {
		properties["vpc_id"] = *instance.VpcId
	}
	if instance.SubnetId != nil {
		properties["subnet_id"] = *instance.SubnetId
	}
	if instance.Placement != nil && instance.Placement.AvailabilityZone != nil {
		properties["availability_zone"] = *instance.Placement.AvailabilityZone
	}

	var tags map[string]string
	var name string
	if len(instance.Tags) > 0 {
		tags = make(map[string]string)
		for _, tag := range instance.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
				if *tag.Key == "Name" {
					name = *tag.Value
				}
			}
		}
	}

	if name == "" {
		name = safeString(instance.InstanceId)
	}

	arn := fmt.Sprintf("arn:aws:ec2:%s:%s:instance/%s", region, account, safeString(instance.InstanceId))

	res := &resource.Resource{
		ID:         safeString(instance.InstanceId),
		Type:       resource.TypeAWSEC2Instance,
		Name:       name,
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        arn,
		Tags:       tags,
		Properties: properties,
		RawData:    instance,
	}

	// Add relationships
	if instance.VpcId != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   *instance.VpcId,
			TargetType: resource.TypeAWSVPC,
		})
	}

	if instance.SubnetId != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   *instance.SubnetId,
			TargetType: resource.TypeAWSSubnet,
		})
	}

	return res
}

// convertEKSClusterToResource converts an EKS cluster to a Resource
func (p *Provider) convertEKSClusterToResource(cluster *eksTypes.Cluster, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"version": safeString(cluster.Version),
		"status":  string(cluster.Status),
	}

	if cluster.Endpoint != nil {
		properties["endpoint"] = *cluster.Endpoint
	}
	if cluster.PlatformVersion != nil {
		properties["platform_version"] = *cluster.PlatformVersion
	}
	if cluster.ResourcesVpcConfig != nil {
		if cluster.ResourcesVpcConfig.VpcId != nil {
			properties["vpc_id"] = *cluster.ResourcesVpcConfig.VpcId
		}
		if len(cluster.ResourcesVpcConfig.SubnetIds) > 0 {
			properties["subnet_ids"] = cluster.ResourcesVpcConfig.SubnetIds
		}
	}

	var createdAt *resource.Resource
	if cluster.CreatedAt != nil {
		t := *cluster.CreatedAt
		createdAt = &resource.Resource{CreatedAt: &t}
	}

	res := &resource.Resource{
		ID:         safeString(cluster.Name),
		Type:       resource.TypeAWSEKSCluster,
		Name:       safeString(cluster.Name),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        safeString(cluster.Arn),
		Tags:       cluster.Tags,
		Properties: properties,
		RawData:    cluster,
	}

	if createdAt != nil && createdAt.CreatedAt != nil {
		res.CreatedAt = createdAt.CreatedAt
	}

	// Add VPC relationship
	if cluster.ResourcesVpcConfig != nil && cluster.ResourcesVpcConfig.VpcId != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   *cluster.ResourcesVpcConfig.VpcId,
			TargetType: resource.TypeAWSVPC,
		})
	}

	return res
}

// convertLambdaFunctionToResource converts a Lambda function to a Resource
func (p *Provider) convertLambdaFunctionToResource(function *lambdaTypes.FunctionConfiguration, region string) *resource.Resource {
	var account string
	if len(p.accounts) > 0 {
		account = p.accounts[0]
	}

	properties := map[string]interface{}{
		"runtime": string(function.Runtime),
		"handler": safeString(function.Handler),
	}

	if function.CodeSize != 0 {
		properties["code_size"] = function.CodeSize
	}
	if function.MemorySize != nil {
		properties["memory_size"] = *function.MemorySize
	}
	if function.Timeout != nil {
		properties["timeout"] = *function.Timeout
	}
	if function.LastModified != nil {
		properties["last_modified"] = *function.LastModified
	}
	if function.VpcConfig != nil && function.VpcConfig.VpcId != nil {
		properties["vpc_id"] = *function.VpcConfig.VpcId
	}

	res := &resource.Resource{
		ID:         safeString(function.FunctionName),
		Type:       resource.TypeAWSLambda,
		Name:       safeString(function.FunctionName),
		Provider:   "aws",
		Account:    account,
		Region:     region,
		ARN:        safeString(function.FunctionArn),
		Properties: properties,
		RawData:    function,
	}

	// Add VPC relationship if Lambda is in VPC
	if function.VpcConfig != nil && function.VpcConfig.VpcId != nil {
		res.Relationships = append(res.Relationships, resource.Relationship{
			Type:       resource.RelationBelongsTo,
			TargetID:   *function.VpcConfig.VpcId,
			TargetType: resource.TypeAWSVPC,
		})
	}

	return res
}
