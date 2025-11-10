package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/exporter"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

var (
	configFile  string
	outputFile  string
	format      string
	pretty      bool
	includeRaw  bool
	concurrency int
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
