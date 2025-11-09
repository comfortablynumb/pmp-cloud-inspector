package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	_ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/aws"    // Register AWS provider
	_ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/github" // Register GitHub provider
	// Uncomment after running 'go mod tidy' to download dependencies:
	// _ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/gcp"    // Register GCP provider
	// _ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/gitlab" // Register GitLab provider
	// _ "github.com/comfortablynumb/pmp-cloud-inspector/pkg/provider/jfrog"  // Register JFrog provider
)

var rootCmd = &cobra.Command{
	Use:   "pmp-cloud-inspector",
	Short: "Cloud resource inspector and exporter",
	Long: `A CLI tool to inspect cloud resources across multiple providers (AWS, GCP, etc.)
and export them to various formats (JSON, YAML, DOT).

The tool reads a YAML configuration file that specifies which cloud providers,
accounts, and resource types to inspect. It then discovers relationships between
resources and exports them in the desired format.`,
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	rootCmd.AddCommand(uiCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
