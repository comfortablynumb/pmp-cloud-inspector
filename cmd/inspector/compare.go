package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/spf13/cobra"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

var (
	baseFile    string
	compareFile string
	outputType  string
)

var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare two or more cloud resource exports and show drifts",
	Long: `Compare cloud resource exports to identify changes between different points in time.
The command shows added, removed, and modified resources between exports.

Examples:
  # Compare two exports
  pmp-cloud-inspector compare -b export1.json -c export2.json

  # Compare with detailed output
  pmp-cloud-inspector compare -b export1.json -c export2.json -t detailed`,
	RunE: runCompare,
}

func init() {
	compareCmd.Flags().StringVarP(&baseFile, "base", "b", "", "Base export file (older snapshot)")
	compareCmd.Flags().StringVarP(&compareFile, "compare", "c", "", "Compare export file (newer snapshot)")
	compareCmd.Flags().StringVarP(&outputType, "type", "t", "summary", "Output type: summary, detailed, json")
	if err := compareCmd.MarkFlagRequired("base"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to mark base flag as required: %v\n", err)
	}
	if err := compareCmd.MarkFlagRequired("compare"); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to mark compare flag as required: %v\n", err)
	}
}

// DriftReport represents the differences between two exports
type DriftReport struct {
	BaseTimestamp    time.Time            `json:"base_timestamp"`
	CompareTimestamp time.Time            `json:"compare_timestamp"`
	Added            []*resource.Resource `json:"added"`
	Removed          []*resource.Resource `json:"removed"`
	Modified         []ResourceDiff       `json:"modified"`
	Summary          DriftSummary         `json:"summary"`
}

// ResourceDiff represents changes to a resource
type ResourceDiff struct {
	ResourceID   string                    `json:"resource_id"`
	ResourceType resource.ResourceType     `json:"resource_type"`
	Name         string                    `json:"name"`
	Changes      map[string]PropertyChange `json:"changes"`
	BaseResource *resource.Resource        `json:"base_resource,omitempty"`
	NewResource  *resource.Resource        `json:"new_resource,omitempty"`
}

// PropertyChange represents a change in a resource property
type PropertyChange struct {
	Old interface{} `json:"old"`
	New interface{} `json:"new"`
}

// DriftSummary provides summary statistics
type DriftSummary struct {
	TotalAdded     int `json:"total_added"`
	TotalRemoved   int `json:"total_removed"`
	TotalModified  int `json:"total_modified"`
	TotalUnchanged int `json:"total_unchanged"`
}

func runCompare(cmd *cobra.Command, args []string) error {
	// Load base export
	baseCollection, err := loadExport(baseFile)
	if err != nil {
		return fmt.Errorf("failed to load base export: %w", err)
	}

	// Load compare export
	compareCollection, err := loadExport(compareFile)
	if err != nil {
		return fmt.Errorf("failed to load compare export: %w", err)
	}

	// Generate drift report
	report := generateDriftReport(baseCollection, compareCollection)

	// Output report based on type
	switch outputType {
	case "json":
		return outputJSON(report)
	case "detailed":
		return outputDetailed(report)
	default:
		return outputSummary(report)
	}
}

func loadExport(filePath string) (*resource.Collection, error) {
	// #nosec G304 - filePath is provided by user as CLI argument, this is expected behavior
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", closeErr)
		}
	}()

	var collection resource.Collection
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&collection); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	return &collection, nil
}

func generateDriftReport(base, compare *resource.Collection) *DriftReport {
	report := &DriftReport{
		BaseTimestamp:    base.Metadata.Timestamp,
		CompareTimestamp: compare.Metadata.Timestamp,
		Added:            make([]*resource.Resource, 0),
		Removed:          make([]*resource.Resource, 0),
		Modified:         make([]ResourceDiff, 0),
	}

	// Create index maps for quick lookup
	baseIndex := make(map[string]*resource.Resource)
	compareIndex := make(map[string]*resource.Resource)

	for _, res := range base.Resources {
		baseIndex[res.ID] = res
	}

	for _, res := range compare.Resources {
		compareIndex[res.ID] = res
	}

	// Find removed resources (in base but not in compare)
	for id, res := range baseIndex {
		if _, exists := compareIndex[id]; !exists {
			report.Removed = append(report.Removed, res)
		}
	}

	// Find added and modified resources
	for id, compareRes := range compareIndex {
		baseRes, exists := baseIndex[id]
		if !exists {
			// Resource added
			report.Added = append(report.Added, compareRes)
		} else {
			// Check if modified
			diff := compareResources(baseRes, compareRes)
			if len(diff.Changes) > 0 {
				report.Modified = append(report.Modified, diff)
			} else {
				report.Summary.TotalUnchanged++
			}
		}
	}

	// Update summary
	report.Summary.TotalAdded = len(report.Added)
	report.Summary.TotalRemoved = len(report.Removed)
	report.Summary.TotalModified = len(report.Modified)

	return report
}

func compareResources(base, compare *resource.Resource) ResourceDiff {
	diff := ResourceDiff{
		ResourceID:   base.ID,
		ResourceType: base.Type,
		Name:         base.Name,
		Changes:      make(map[string]PropertyChange),
		BaseResource: base,
		NewResource:  compare,
	}

	// Compare basic fields
	if base.Name != compare.Name {
		diff.Changes["name"] = PropertyChange{Old: base.Name, New: compare.Name}
	}
	if base.Region != compare.Region {
		diff.Changes["region"] = PropertyChange{Old: base.Region, New: compare.Region}
	}

	// Compare tags
	if !reflect.DeepEqual(base.Tags, compare.Tags) {
		diff.Changes["tags"] = PropertyChange{Old: base.Tags, New: compare.Tags}
	}

	// Compare properties
	for key, baseValue := range base.Properties {
		compareValue, exists := compare.Properties[key]
		if !exists {
			diff.Changes[fmt.Sprintf("properties.%s", key)] = PropertyChange{Old: baseValue, New: nil}
		} else if !reflect.DeepEqual(baseValue, compareValue) {
			diff.Changes[fmt.Sprintf("properties.%s", key)] = PropertyChange{Old: baseValue, New: compareValue}
		}
	}

	// Check for new properties
	for key, compareValue := range compare.Properties {
		if _, exists := base.Properties[key]; !exists {
			diff.Changes[fmt.Sprintf("properties.%s", key)] = PropertyChange{Old: nil, New: compareValue}
		}
	}

	// Compare timestamps if available
	if base.UpdatedAt != nil && compare.UpdatedAt != nil {
		if !base.UpdatedAt.Equal(*compare.UpdatedAt) {
			diff.Changes["updated_at"] = PropertyChange{Old: base.UpdatedAt, New: compare.UpdatedAt}
		}
	}

	return diff
}

func outputJSON(report *DriftReport) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

func outputSummary(report *DriftReport) error {
	fmt.Println("=== Cloud Resource Drift Report ===")
	fmt.Printf("Base snapshot:    %s\n", report.BaseTimestamp.Format(time.RFC3339))
	fmt.Printf("Compare snapshot: %s\n", report.CompareTimestamp.Format(time.RFC3339))
	fmt.Println()

	fmt.Println("Summary:")
	fmt.Printf("  Added:     %d resources\n", report.Summary.TotalAdded)
	fmt.Printf("  Removed:   %d resources\n", report.Summary.TotalRemoved)
	fmt.Printf("  Modified:  %d resources\n", report.Summary.TotalModified)
	fmt.Printf("  Unchanged: %d resources\n", report.Summary.TotalUnchanged)
	fmt.Println()

	if len(report.Added) > 0 {
		fmt.Printf("Added Resources (%d):\n", len(report.Added))
		for _, res := range report.Added {
			fmt.Printf("  + [%s] %s (%s)\n", res.Type, res.Name, res.ID)
		}
		fmt.Println()
	}

	if len(report.Removed) > 0 {
		fmt.Printf("Removed Resources (%d):\n", len(report.Removed))
		for _, res := range report.Removed {
			fmt.Printf("  - [%s] %s (%s)\n", res.Type, res.Name, res.ID)
		}
		fmt.Println()
	}

	if len(report.Modified) > 0 {
		fmt.Printf("Modified Resources (%d):\n", len(report.Modified))
		for _, diff := range report.Modified {
			fmt.Printf("  ~ [%s] %s (%s) - %d changes\n",
				diff.ResourceType, diff.Name, diff.ResourceID, len(diff.Changes))
		}
		fmt.Println()
	}

	return nil
}

func outputDetailed(report *DriftReport) error {
	// First output summary
	if err := outputSummary(report); err != nil {
		return err
	}

	// Then detailed changes
	if len(report.Modified) > 0 {
		fmt.Println("=== Detailed Changes ===")
		for i, diff := range report.Modified {
			fmt.Printf("\n%d. [%s] %s (%s)\n", i+1, diff.ResourceType, diff.Name, diff.ResourceID)
			for field, change := range diff.Changes {
				fmt.Printf("   %s:\n", field)
				fmt.Printf("     - Old: %v\n", formatValue(change.Old))
				fmt.Printf("     + New: %v\n", formatValue(change.New))
			}
		}
	}

	return nil
}

func formatValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}

	// Format complex types as JSON
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}
