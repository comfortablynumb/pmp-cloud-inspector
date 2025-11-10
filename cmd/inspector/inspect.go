package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/cost"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/exporter"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/filter"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

var (
	configFile    string
	outputFile    string
	format        string
	pretty        bool
	includeRaw    bool
	concurrency   int
	estimateCosts bool

	// Filter flags
	filterTags       []string
	filterRegex      []string
	filterDateRange  []string
	filterStates     string
	filterProperties []string
	filterCost       string
	filterTypes      []string
	filterProviders  []string
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect cloud resources and export to various formats",
	Long: `Inspect cloud resources across multiple providers and export them to JSON, YAML, or DOT format.

The inspect command reads a YAML configuration file that specifies which cloud providers,
accounts, and resource types to inspect. It then discovers relationships between
resources and exports them in the desired format.`,
	RunE: runInspect,
}

func init() {
	inspectCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to configuration file")
	inspectCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (defaults to stdout)")
	inspectCmd.Flags().StringVarP(&format, "format", "f", "", "Output format: json, yaml, dot (overrides config)")
	inspectCmd.Flags().BoolVarP(&pretty, "pretty", "p", true, "Pretty print output")
	inspectCmd.Flags().BoolVar(&includeRaw, "include-raw", false, "Include raw cloud provider data")
	inspectCmd.Flags().IntVar(&concurrency, "concurrent", 4, "Number of concurrent goroutines for parallel resource collection")
	inspectCmd.Flags().BoolVar(&estimateCosts, "estimate-costs", false, "Estimate monthly costs for resources")

	// Filter flags
	inspectCmd.Flags().StringSliceVar(&filterTags, "filter-tag", nil, "Filter by tags (e.g., Environment=prod, Name~test, Owner)")
	inspectCmd.Flags().StringSliceVar(&filterRegex, "filter-regex", nil, "Filter by regex (e.g., name:/prod-.*/, id:/^i-/)")
	inspectCmd.Flags().StringSliceVar(&filterDateRange, "filter-date", nil, "Filter by date range (e.g., created:>2024-01-01, updated:2024-01..2024-12)")
	inspectCmd.Flags().StringVar(&filterStates, "filter-state", "", "Filter by resource states (comma-separated, e.g., running,active)")
	inspectCmd.Flags().StringSliceVar(&filterProperties, "filter-property", nil, "Filter by property (e.g., vm_size=Standard_D2s_v3, enabled=true, cost>100)")
	inspectCmd.Flags().StringVar(&filterCost, "filter-cost", "", "Filter by cost (e.g., 100..500, >100, <500)")
	inspectCmd.Flags().StringSliceVar(&filterTypes, "filter-type", nil, "Filter by resource types (e.g., aws:ec2:instance)")
	inspectCmd.Flags().StringSliceVar(&filterProviders, "filter-provider", nil, "Filter by providers (e.g., aws, azure, gcp)")
}

// contextKey is a type for context keys to avoid collisions
type contextKey string

const concurrencyKey contextKey = "concurrency"

func runInspect(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Add concurrency setting to context
	ctx = context.WithValue(ctx, concurrencyKey, concurrency)

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if validateErr := cfg.Validate(); validateErr != nil {
		return fmt.Errorf("invalid config: %w", validateErr)
	}

	fmt.Fprintf(os.Stderr, "Loaded configuration from %s\n", configFile)

	// Collect resources from all configured providers
	allResources := resource.NewCollection()

	for _, providerCfg := range cfg.Providers {
		fmt.Fprintf(os.Stderr, "Initializing provider: %s\n", providerCfg.Name)

		// Get provider factory
		registry := provider.GetRegistry()
		p, createErr := registry.Create(providerCfg.Name)
		if createErr != nil {
			return fmt.Errorf("failed to create provider %s: %w", providerCfg.Name, createErr)
		}

		// Initialize provider
		if initErr := p.Initialize(ctx, providerCfg); initErr != nil {
			return fmt.Errorf("failed to initialize provider %s: %w", providerCfg.Name, initErr)
		}

		// Collect resources
		fmt.Fprintf(os.Stderr, "Collecting resources from %s...\n", providerCfg.Name)

		var resourceTypes []resource.ResourceType
		if !cfg.Resources.IncludeAll && len(cfg.Resources.Types) > 0 {
			// Convert string types to ResourceType
			for _, typeStr := range cfg.Resources.Types {
				resourceTypes = append(resourceTypes, resource.ResourceType(typeStr))
			}
		}

		collection, collectErr := p.CollectResources(ctx, resourceTypes)
		if collectErr != nil {
			return fmt.Errorf("failed to collect resources from %s: %w", providerCfg.Name, collectErr)
		}

		fmt.Fprintf(os.Stderr, "Collected %d resources from %s\n", len(collection.Resources), providerCfg.Name)

		// Discover relationships if enabled
		if cfg.Resources.Relationships {
			fmt.Fprintf(os.Stderr, "Discovering relationships...\n")
			if relErr := p.DiscoverRelationships(ctx, collection); relErr != nil {
				return fmt.Errorf("failed to discover relationships: %w", relErr)
			}
		}

		// Merge into all resources
		for _, res := range collection.Resources {
			allResources.Add(res)
		}
	}

	fmt.Fprintf(os.Stderr, "Total resources collected: %d\n", len(allResources.Resources))

	// Estimate costs if enabled
	if estimateCosts {
		fmt.Fprintf(os.Stderr, "Estimating costs...\n")
		if costErr := estimateResourceCosts(allResources); costErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to estimate costs: %v\n", costErr)
		} else if allResources.Metadata.TotalCost != nil {
			fmt.Fprintf(os.Stderr, "Estimated total monthly cost: $%.2f %s\n",
				allResources.Metadata.TotalCost.Total,
				allResources.Metadata.TotalCost.Currency)
		}
	}

	// Apply filters if any
	filters, err := buildFilters()
	if err != nil {
		return fmt.Errorf("failed to build filters: %w", err)
	}

	if len(filters) > 0 {
		fmt.Fprintf(os.Stderr, "Applying filters...\n")
		allResources = filter.ApplyFilters(allResources, filters...)
		fmt.Fprintf(os.Stderr, "Filtered to %d resources\n", len(allResources.Resources))
	}

	// Determine output format
	outputFormat := format
	if outputFormat == "" {
		outputFormat = cfg.Export.Format
	}
	if outputFormat == "" {
		outputFormat = "json" // default
	}

	// Get exporter
	exp, err := exporter.Get(outputFormat)
	if err != nil {
		return fmt.Errorf("failed to get exporter: %w", err)
	}

	// Determine output writer
	var writer *os.File
	if outputFile != "" {
		// #nosec G304 - outputFile is provided by user as CLI argument, this is expected behavior
		writer, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if closeErr := writer.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close output file: %v\n", closeErr)
			}
		}()
		fmt.Fprintf(os.Stderr, "Writing output to %s in %s format...\n", outputFile, outputFormat)
	} else {
		writer = os.Stdout
		fmt.Fprintf(os.Stderr, "Writing output to stdout in %s format...\n", outputFormat)
	}

	// Export
	exportOptions := exporter.ExportOptions{
		Pretty:     pretty,
		IncludeRaw: includeRaw,
	}

	if err := exp.Export(allResources, writer, exportOptions); err != nil {
		return fmt.Errorf("failed to export resources: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Export completed successfully!\n")

	return nil
}

// buildFilters constructs filters from command-line flags
func buildFilters() ([]filter.Filter, error) {
	var filters []filter.Filter

	// Tag filters
	for _, tagExpr := range filterTags {
		f, err := filter.ParseTagFilter(tagExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid tag filter '%s': %w", tagExpr, err)
		}
		filters = append(filters, f)
	}

	// Regex filters
	for _, regexExpr := range filterRegex {
		f, err := filter.ParseRegexFilter(regexExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid regex filter '%s': %w", regexExpr, err)
		}
		filters = append(filters, f)
	}

	// Date range filters
	for _, dateExpr := range filterDateRange {
		f, err := filter.ParseDateRangeFilter(dateExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid date filter '%s': %w", dateExpr, err)
		}
		filters = append(filters, f)
	}

	// State filter
	if filterStates != "" {
		f, err := filter.ParseStateFilter(filterStates)
		if err != nil {
			return nil, fmt.Errorf("invalid state filter '%s': %w", filterStates, err)
		}
		filters = append(filters, f)
	}

	// Property filters
	for _, propExpr := range filterProperties {
		f, err := filter.ParsePropertyFilter(propExpr)
		if err != nil {
			return nil, fmt.Errorf("invalid property filter '%s': %w", propExpr, err)
		}
		filters = append(filters, f)
	}

	// Cost filter
	if filterCost != "" {
		f, err := filter.ParseCostFilter(filterCost)
		if err != nil {
			return nil, fmt.Errorf("invalid cost filter '%s': %w", filterCost, err)
		}
		filters = append(filters, f)
	}

	// Type filter
	if len(filterTypes) > 0 {
		f := filter.ParseTypeFilter(filterTypes)
		filters = append(filters, f)
	}

	// Provider filter
	if len(filterProviders) > 0 {
		f := filter.ParseProviderFilter(filterProviders)
		filters = append(filters, f)
	}

	return filters, nil
}

// estimateResourceCosts estimates costs for all resources in the collection
func estimateResourceCosts(collection *resource.Collection) error {
	// Create cost estimator registry
	registry := cost.NewEstimatorRegistry()

	// Register estimators for each provider
	registry.Register("aws", cost.NewAWSEstimator())
	registry.Register("azure", cost.NewAzureEstimator())
	registry.Register("gcp", cost.NewGCPEstimator())

	// Estimate costs for all resources
	return registry.EstimateCollection(collection)
}
