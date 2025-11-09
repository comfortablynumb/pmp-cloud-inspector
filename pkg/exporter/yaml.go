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
		//nolint:errcheck // Ignore error on close since we can't return it from defer
		encoder.Close()
	}()

	if options.Pretty {
		encoder.SetIndent(2)
	}

	return encoder.Encode(collection)
}
