package exporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// DOTExporter exports resources in GraphViz DOT format
type DOTExporter struct{}

// Format returns the format name
func (e *DOTExporter) Format() string {
	return "dot"
}

// Export exports the collection to DOT format
func (e *DOTExporter) Export(collection *resource.Collection, writer io.Writer, options ExportOptions) error {
	// Write DOT header
	if _, err := fmt.Fprintf(writer, "digraph cloud_resources {\n"); err != nil {
		return err
	}

	// Set graph properties
	if _, err := fmt.Fprintf(writer, "  rankdir=LR;\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(writer, "  node [shape=box, style=rounded];\n\n"); err != nil {
		return err
	}

	// Write nodes
	for _, res := range collection.Resources {
		if err := e.writeNode(writer, res); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(writer, "\n"); err != nil {
		return err
	}

	// Write edges (relationships)
	for _, res := range collection.Resources {
		if err := e.writeEdges(writer, res); err != nil {
			return err
		}
	}

	// Write DOT footer
	if _, err := fmt.Fprintf(writer, "}\n"); err != nil {
		return err
	}

	return nil
}

// writeNode writes a single node to the DOT output
func (e *DOTExporter) writeNode(writer io.Writer, res *resource.Resource) error {
	nodeID := e.sanitizeID(res.ID)
	label := e.formatLabel(res)
	color := e.getColorForType(res.Type)

	_, err := fmt.Fprintf(writer, "  %s [label=\"%s\", fillcolor=\"%s\", style=\"filled,rounded\"];\n",
		nodeID, label, color)
	return err
}

// writeEdges writes all edges for a resource
func (e *DOTExporter) writeEdges(writer io.Writer, res *resource.Resource) error {
	nodeID := e.sanitizeID(res.ID)

	for _, rel := range res.Relationships {
		targetID := e.sanitizeID(rel.TargetID)
		label := string(rel.Type)

		if _, err := fmt.Fprintf(writer, "  %s -> %s [label=\"%s\"];\n",
			nodeID, targetID, label); err != nil {
			return err
		}
	}

	return nil
}

// sanitizeID makes an ID safe for DOT format
func (e *DOTExporter) sanitizeID(id string) string {
	// Replace characters that are not valid in DOT identifiers
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "-", "_")
	id = strings.ReplaceAll(id, ".", "_")
	return fmt.Sprintf("\"%s\"", id)
}

// formatLabel creates a label for a node
func (e *DOTExporter) formatLabel(res *resource.Resource) string {
	label := fmt.Sprintf("%s\\n%s", res.Name, res.Type)

	if res.Region != "" {
		label += fmt.Sprintf("\\n%s", res.Region)
	}

	return label
}

// getColorForType returns a color based on resource type
func (e *DOTExporter) getColorForType(resourceType resource.ResourceType) string {
	colors := map[resource.ResourceType]string{
		resource.TypeAWSIAMUser:       "#FFE4B5",
		resource.TypeAWSIAMRole:       "#FFD700",
		resource.TypeAWSAccount:       "#87CEEB",
		resource.TypeAWSVPC:           "#98FB98",
		resource.TypeAWSSubnet:        "#90EE90",
		resource.TypeAWSSecurityGroup: "#FFA07A",
		resource.TypeAWSECR:           "#DDA0DD",
	}

	if color, ok := colors[resourceType]; ok {
		return color
	}

	return "#E0E0E0" // Default gray
}
