package exporter

import (
	"io"

	"gopkg.in/yaml.v3"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// YAMLExporter exports resources in YAML format
type YAMLExporter struct{}

// Format returns the format name
func (e *YAMLExporter) Format() string {
	return "yaml"
}

// Export exports the collection to YAML
func (e *YAMLExporter) Export(collection *resource.Collection, writer io.Writer, options ExportOptions) error {
	// Filter out raw data if not requested
	if !options.IncludeRaw {
		collection = filterRawData(collection)
	}

	encoder := yaml.NewEncoder(writer)
	defer func() {
		if err := encoder.Close(); err != nil {
			// Log error but don't override return value
			// since we're in a defer
		}
	}()

	if options.Pretty {
		encoder.SetIndent(2)
	}

	return encoder.Encode(collection)
}
