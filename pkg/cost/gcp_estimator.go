package cost

import (
	"strings"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// GCPEstimator provides cost estimation for GCP resources
type GCPEstimator struct {
	// Simplified pricing data (USD per month)
	// In a production system, this would integrate with GCP Cloud Billing Catalog API
	pricing map[resource.ResourceType]float64
}

// NewGCPEstimator creates a new GCP cost estimator
func NewGCPEstimator() *GCPEstimator {
	return &GCPEstimator{
		pricing: map[resource.ResourceType]float64{
			// Compute Instance - Average n1-standard-1 (730 hours/month)
			resource.TypeGCPComputeInstance: 24.27,

			// VPC Network - Free tier
			resource.TypeGCPVPC: 0.00,

			// Cloud Storage Bucket - Average Standard Storage (100GB)
			resource.TypeGCPStorageBucket: 2.60,

			// Cloud Function - Average 128MB, 1M invocations
			resource.TypeGCPCloudFunction: 0.40,

			// Cloud Run Service - Average 1 vCPU, 512MB
			resource.TypeGCPCloudRun: 8.00,
		},
	}
}

// EstimateCost estimates the cost for a GCP resource
func (e *GCPEstimator) EstimateCost(res *resource.Resource) (*resource.ResourceCost, error) {
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

	// Adjust pricing based on machine type or other properties
	switch res.Type {
	case resource.TypeGCPComputeInstance:
		cost = e.estimateComputeCost(res, basePrice)
	case resource.TypeGCPStorageBucket:
		cost = e.estimateStorageCost(res, basePrice)
	}

	return cost, nil
}

// estimateComputeCost provides more accurate Compute Engine cost based on machine type
func (e *GCPEstimator) estimateComputeCost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	cost := &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown:       make(map[string]float64),
	}

	// Check if instance is stopped
	if status, ok := res.Properties["status"].(string); ok {
		if status == "TERMINATED" || status == "STOPPED" {
			cost.MonthlyEstimate = 0
			cost.Breakdown["compute"] = 0
			return cost
		}
	}

	// Adjust based on machine type
	if machineType, ok := res.Properties["machine_type"].(string); ok {
		multiplier := e.getMachineTypeMultiplier(machineType)
		cost.MonthlyEstimate = basePrice * multiplier
		cost.Breakdown["compute"] = cost.MonthlyEstimate
	}

	return cost
}

// getMachineTypeMultiplier returns a cost multiplier based on machine type
func (e *GCPEstimator) getMachineTypeMultiplier(machineType string) float64 {
	// Simplified multipliers based on machine family
	switch {
	case strings.Contains(machineType, "f1-micro"):
		return 0.3 // Micro instance
	case strings.Contains(machineType, "g1-small"):
		return 0.5 // Small instance
	case strings.Contains(machineType, "n1-standard-1"):
		return 1.0 // Baseline
	case strings.Contains(machineType, "n1-standard-2"):
		return 2.0
	case strings.Contains(machineType, "n1-standard-4"):
		return 4.0
	case strings.Contains(machineType, "n1-standard-8"):
		return 8.0
	case strings.Contains(machineType, "n1-highmem"):
		return 3.0 // Memory optimized
	case strings.Contains(machineType, "n1-highcpu"):
		return 2.5 // CPU optimized
	case strings.Contains(machineType, "a2-"):
		return 30.0 // GPU instances
	default:
		return 1.0
	}
}

// estimateStorageCost provides storage bucket cost
func (e *GCPEstimator) estimateStorageCost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	return &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown: map[string]float64{
			"storage": basePrice,
		},
	}
}
