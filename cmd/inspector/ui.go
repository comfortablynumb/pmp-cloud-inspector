package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/ui"
)

var (
	port int
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the web UI server",
	Long: `Start a web server that provides a user interface for viewing cloud resources.

The UI allows you to upload exported JSON or YAML files and view the resources
in a beautiful, interactive interface built with Tailwind CSS and jQuery.`,
	RunE: runUI,
}

func init() {
	uiCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
}

func runUI(cmd *cobra.Command, args []string) error {
	fmt.Printf("Starting PMP Cloud Inspector UI on port %d...\n", port)
	fmt.Printf("Open your browser at http://localhost:%d\n", port)

	server := ui.NewServer(port)
	return server.Start()
}
