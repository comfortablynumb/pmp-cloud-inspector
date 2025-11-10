package cost

import (
	"strings"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// AzureEstimator provides cost estimation for Azure resources
type AzureEstimator struct {
	// Simplified pricing data (USD per month)
	// In a production system, this would integrate with Azure Pricing API
	pricing map[resource.ResourceType]float64
}

// NewAzureEstimator creates a new Azure cost estimator
func NewAzureEstimator() *AzureEstimator {
	return &AzureEstimator{
		pricing: map[resource.ResourceType]float64{
			// Virtual Machines - Average Standard_D2s_v3
			resource.TypeAzureVM: 70.08,

			// Virtual Network - Minimal cost
			resource.TypeAzureVNet: 0.00,

			// Storage Account - Average General Purpose v2
			resource.TypeAzureStorageAccount: 20.00,

			// App Service - Average B1 plan
			resource.TypeAzureAppService: 13.14,

			// SQL Database - Average Basic tier
			resource.TypeAzureSQLDatabase: 4.99,

			// Key Vault - Base cost + operations
			resource.TypeAzureKeyVault: 5.00,
		},
	}
}

// EstimateCost estimates the cost for an Azure resource
func (e *AzureEstimator) EstimateCost(res *resource.Resource) (*resource.ResourceCost, error) {
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

	// Adjust pricing based on SKU or other properties
	switch res.Type {
	case resource.TypeAzureVM:
		cost = e.estimateVMCost(res, basePrice)
	case resource.TypeAzureStorageAccount:
		cost = e.estimateStorageCost(res, basePrice)
	case resource.TypeAzureAppService:
		cost = e.estimateAppServiceCost(res, basePrice)
	}

	return cost, nil
}

// estimateVMCost provides more accurate VM cost based on size
func (e *AzureEstimator) estimateVMCost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	cost := &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown:       make(map[string]float64),
	}

	// Check if VM is deallocated or stopped
	if state, ok := res.Properties["provisioning_state"].(string); ok {
		if state == "Deallocated" || state == "Stopped" {
			cost.MonthlyEstimate = 0
			cost.Breakdown["compute"] = 0
			return cost
		}
	}

	// Adjust based on VM size
	if vmSize, ok := res.Properties["vm_size"].(string); ok {
		multiplier := e.getVMSizeMultiplier(vmSize)
		cost.MonthlyEstimate = basePrice * multiplier
		cost.Breakdown["compute"] = cost.MonthlyEstimate
	}

	return cost
}

// getVMSizeMultiplier returns a cost multiplier based on VM size
func (e *AzureEstimator) getVMSizeMultiplier(vmSize string) float64 {
	// Simplified multipliers based on VM series
	switch {
	case strings.Contains(vmSize, "Standard_B"):
		return 0.3 // Burstable VMs
	case strings.Contains(vmSize, "Standard_D2"):
		return 1.0 // Baseline - D2
	case strings.Contains(vmSize, "Standard_D4"):
		return 2.0 // D4
	case strings.Contains(vmSize, "Standard_D8"):
		return 4.0 // D8
	case strings.Contains(vmSize, "Standard_D16"):
		return 8.0 // D16
	case strings.Contains(vmSize, "Standard_E"):
		return 3.0 // Memory optimized
	case strings.Contains(vmSize, "Standard_F"):
		return 2.5 // Compute optimized
	case strings.Contains(vmSize, "Standard_NC"):
		return 20.0 // GPU VMs
	default:
		return 1.0
	}
}

// estimateStorageCost provides storage account cost
func (e *AzureEstimator) estimateStorageCost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	return &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown: map[string]float64{
			"storage": basePrice,
		},
	}
}

// estimateAppServiceCost provides App Service cost based on plan
func (e *AzureEstimator) estimateAppServiceCost(res *resource.Resource, basePrice float64) *resource.ResourceCost {
	cost := &resource.ResourceCost{
		MonthlyEstimate: basePrice,
		Currency:        "USD",
		Breakdown: map[string]float64{
			"app_service_plan": basePrice,
		},
	}

	// Could be enhanced with actual plan tier from properties
	return cost
}
