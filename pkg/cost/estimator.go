package cost

import (
	"time"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Estimator provides cost estimation for cloud resources
type Estimator interface {
	EstimateCost(res *resource.Resource) (*resource.ResourceCost, error)
}

// EstimatorRegistry holds estimators for different providers
type EstimatorRegistry struct {
	estimators map[string]Estimator
}

// NewEstimatorRegistry creates a new estimator registry
func NewEstimatorRegistry() *EstimatorRegistry {
	return &EstimatorRegistry{
		estimators: make(map[string]Estimator),
	}
}

// Register registers an estimator for a provider
func (r *EstimatorRegistry) Register(provider string, estimator Estimator) {
	r.estimators[provider] = estimator
}

// EstimateCost estimates the cost for a resource using the appropriate estimator
func (r *EstimatorRegistry) EstimateCost(res *resource.Resource) (*resource.ResourceCost, error) {
	estimator, ok := r.estimators[res.Provider]
	if !ok {
		// No estimator for this provider - return nil (no cost data)
		return nil, nil
	}

	return estimator.EstimateCost(res)
}

// EstimateCollection estimates costs for all resources in a collection
func (r *EstimatorRegistry) EstimateCollection(collection *resource.Collection) error {
	now := time.Now()

	for _, res := range collection.Resources {
		cost, err := r.EstimateCost(res)
		if err != nil {
			// Log error but continue with other resources
			continue
		}

		if cost != nil {
			cost.LastUpdated = now
			res.Cost = cost
		}
	}

	return nil
}
