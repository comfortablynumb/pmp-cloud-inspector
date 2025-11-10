package cost

import (
	"strings"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// AWSEstimator provides cost estimation for AWS resources
type AWSEstimator struct {
	// Simplified pricing data (USD per month)
	// In a production system, this would integrate with AWS Pricing API
	pricing map[resource.ResourceType]float64
}

// NewAWSEstimator creates a new AWS cost estimator
func NewAWSEstimator() *AWSEstimator {
	return &AWSEstimator{
		pricing: map[resource.ResourceType]float64{
			// EC2 - Average t3.medium instance (730 hours/month)
			resource.TypeAWSEC2Instance: 30.37,

			// RDS - Average db.t3.medium instance
			// resource.TypeAWSRDS: 54.75,

			// Lambda - Average 128MB, 1M requests, 100ms duration
			resource.TypeAWSLambda: 0.20,

			// S3 - Not per-resource, usage-based
			// resource.TypeAWSS3Bucket: 0.023, // per GB/month

			// DynamoDB - Average on-demand table
			resource.TypeAWSDynamoDBTable: 25.00,

			// EKS - Cluster cost
			resource.TypeAWSEKSCluster: 72.00,

			// ALB - Application Load Balancer
			resource.TypeAWSALB: 22.26,

			// NLB - Network Load Balancer
			resource.TypeAWSNLB: 22.26,

			// CloudFront - Average distribution
			resource.TypeAWSCloudFront: 50.00,

			// ElastiCache - Average cache.t3.medium
			resource.TypeAWSElastiCache: 47.45,

			// MemoryDB - Average db.t4g.medium
			resource.TypeAWSMemoryDB: 50.00,

			// API Gateway - Average REST API (1M requests)
			resource.TypeAWSAPIGateway: 3.50,

			// SNS - Average topic (1M requests)
			resource.TypeAWSSNSTopic: 0.50,

			// SQS - Average queue (1M requests)
			resource.TypeAWSSQSQueue: 0.40,

			// Secrets Manager - Per secret
			resource.TypeAWSSecret: 0.40,

			// ECR - Average repository (500GB storage)
			resource.TypeAWSECR: 50.00,
		},
	}
}

// EstimateCost estimates the cost for an AWS resource
func (e *AWSEstimator) EstimateCost(res *resource.Resource) (*resource.ResourceCost, error) {
	// Get base price from pricing map
	basePrice, ok := e.pricing[res.Type]
	if !ok {
		// No pricing data for this resource type
		return nil, nil
	}

	cost := &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown:       make(map[string]float64),
	}

	// Adjust pricing based on instance type or other properties
	switch res.Type {
	case resource.TypeAWSEC2Instance:
		cost = e.estimateEC2Cost(res, basePrice)
	case resource.TypeAWSElastiCache:
		cost = e.estimateElastiCacheCost(res, basePrice)
	case resource.TypeAWSMemoryDB:
		cost = e.estimateMemoryDBCost(res, basePrice)
	}

	return cost, nil
}

// estimateEC2Cost provides more accurate EC2 cost based on instance type
func (e *AWSEstimator) estimateEC2Cost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	cost := &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown:       make(map[string]float64),
	}

	// Check if instance is stopped (no compute cost)
	if state, ok := res.Properties["state"].(string); ok {
		if state == "stopped" || state == "terminated" {
			cost.MonthlyEstimate = 0
			cost.Breakdown["compute"] = 0
			return cost
		}
	}

	// Adjust based on instance type if available
	if instanceType, ok := res.Properties["instance_type"].(string); ok {
		multiplier := e.getInstanceTypeMultiplier(instanceType)
		cost.MonthlyEstimate = basePrice * multiplier
		cost.Breakdown["compute"] = cost.MonthlyEstimate
	}

	return cost
}

// getInstanceTypeMultiplier returns a cost multiplier based on instance type
func (e *AWSEstimator) getInstanceTypeMultiplier(instanceType string) float64 {
	// Simplified multipliers based on instance family
	switch {
	case strings.HasPrefix(instanceType, "t2."):
		return 0.5 // Smaller, cheaper instances
	case strings.HasPrefix(instanceType, "t3."):
		return 1.0 // Baseline
	case strings.HasPrefix(instanceType, "m5."):
		return 2.0 // General purpose, larger
	case strings.HasPrefix(instanceType, "c5."):
		return 2.5 // Compute optimized
	case strings.HasPrefix(instanceType, "r5."):
		return 3.0 // Memory optimized
	case strings.HasPrefix(instanceType, "p3."):
		return 25.0 // GPU instances
	default:
		return 1.0
	}
}

// estimateElastiCacheCost provides ElastiCache cost based on node type
func (e *AWSEstimator) estimateElastiCacheCost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	cost := &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown: map[string]float64{
			"cache_nodes": basePrice,
		},
	}

	// Adjust based on number of cache nodes
	if numNodes, ok := res.Properties["num_cache_nodes"].(float64); ok {
		cost.MonthlyEstimate = basePrice * numNodes
		cost.Breakdown["cache_nodes"] = cost.MonthlyEstimate
	}

	return cost
}

// estimateMemoryDBCost provides MemoryDB cost based on node type
func (e *AWSEstimator) estimateMemoryDBCost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	cost := &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown: map[string]float64{
			"memory_nodes": basePrice,
		},
	}

	// Adjust based on number of shards
	if numShards, ok := res.Properties["number_of_shards"].(float64); ok {
		cost.MonthlyEstimate = basePrice * numShards
		cost.Breakdown["memory_nodes"] = cost.MonthlyEstimate
	}

	return cost
}
