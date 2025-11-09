package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/config"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/exporter"
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider"
	_ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/aws" // Register AWS provider
	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

var (
	configFile string
	outputFile string
	format     string
	pretty     bool
	includeRaw bool
)

var rootCmd = &cobra.Command{
	Use:   "pmp-cloud-inspector",
	Short: "Cloud resource inspector and exporter",
	Long: `A CLI tool to inspect cloud resources across multiple providers (AWS, GCP, etc.)
and export them to various formats (JSON, YAML, DOT).

The tool reads a YAML configuration file that specifies which cloud providers,
accounts, and resource types to inspect. It then discovers relationships between
resources and exports them in the desired format.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Path to configuration file")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (defaults to stdout)")
	rootCmd.Flags().StringVarP(&format, "format", "f", "", "Output format: json, yaml, dot (overrides config)")
	rootCmd.Flags().BoolVarP(&pretty, "pretty", "p", true, "Pretty print output")
	rootCmd.Flags().BoolVar(&includeRaw, "include-raw", false, "Include raw cloud provider data")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Loaded configuration from %s\n", configFile)

	// Collect resources from all configured providers
	allResources := resource.NewCollection()

	for _, providerCfg := range cfg.Providers {
		fmt.Fprintf(os.Stderr, "Initializing provider: %s\n", providerCfg.Name)

		// Get provider factory
		registry := provider.GetRegistry()
		p, err := registry.Create(providerCfg.Name)
		if err != nil {
			return fmt.Errorf("failed to create provider %s: %w", providerCfg.Name, err)
		}

		// Initialize provider
		if err := p.Initialize(ctx, providerCfg); err != nil {
			return fmt.Errorf("failed to initialize provider %s: %w", providerCfg.Name, err)
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

		collection, err := p.CollectResources(ctx, resourceTypes)
		if err != nil {
			return fmt.Errorf("failed to collect resources from %s: %w", providerCfg.Name, err)
		}

		fmt.Fprintf(os.Stderr, "Collected %d resources from %s\n", len(collection.Resources), providerCfg.Name)

		// Discover relationships if enabled
		if cfg.Resources.Relationships {
			fmt.Fprintf(os.Stderr, "Discovering relationships...\n")
			if err := p.DiscoverRelationships(ctx, collection); err != nil {
				return fmt.Errorf("failed to discover relationships: %w", err)
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
		writer, err = os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer writer.Close()
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
