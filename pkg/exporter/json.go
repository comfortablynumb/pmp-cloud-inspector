package exporter

import (
	"encoding/json"
	"io"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// JSONExporter exports resources in JSON format
type JSONExporter struct{}

// Format returns the format name
func (e *JSONExporter) Format() string {
	return "json"
}

// Export exports the collection to JSON
func (e *JSONExporter) Export(collection *resource.Collection, writer io.Writer, options ExportOptions) error {
	// Filter out raw data if not requested
	if !options.IncludeRaw {
		collection = filterRawData(collection)
	}

	encoder := json.NewEncoder(writer)
	if options.Pretty {
		encoder.SetIndent("", "  ")
	}

	return encoder.Encode(collection)
}

// filterRawData creates a copy of the collection without raw data
func filterRawData(collection *resource.Collection) *resource.Collection {
	filtered := resource.NewCollection()
	filtered.Metadata = collection.Metadata

	for _, res := range collection.Resources {
		// Create a shallow copy
		resCopy := *res
		resCopy.RawData = nil
		filtered.Resources = append(filtered.Resources, &resCopy)
	}

	return filtered
}
