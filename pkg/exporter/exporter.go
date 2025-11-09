package exporter

import (
	"fmt"
	"io"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Exporter defines the interface for exporting resources
type Exporter interface {
	// Export writes the collection to the writer in the specific format
	Export(collection *resource.Collection, writer io.Writer, options ExportOptions) error

	// Format returns the format name
	Format() string
}

// ExportOptions provides configuration for export
type ExportOptions struct {
	Pretty     bool // Pretty print output
	IncludeRaw bool // Include raw cloud provider data
}

// Registry manages all registered exporters
type Registry struct {
	exporters map[string]Exporter
}

var globalRegistry = &Registry{
	exporters: make(map[string]Exporter),
}

// Register registers a new exporter
func Register(exporter Exporter) {
	globalRegistry.exporters[exporter.Format()] = exporter
}

// Get retrieves an exporter by format
func Get(format string) (Exporter, error) {
	exporter, ok := globalRegistry.exporters[format]
	if !ok {
		return nil, fmt.Errorf("exporter not found for format: %s", format)
	}
	return exporter, nil
}

// List returns all registered exporter formats
func List() []string {
	formats := make([]string, 0, len(globalRegistry.exporters))
	for format := range globalRegistry.exporters {
		formats = append(formats, format)
	}
	return formats
}

// init registers default exporters
func init() {
	Register(&JSONExporter{})
	Register(&YAMLExporter{})
	Register(&DOTExporter{})
}
