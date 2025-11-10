package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// Filter is the interface that all filters must implement
type Filter interface {
	// Apply applies the filter to a resource and returns true if the resource matches
	Apply(res *resource.Resource) bool
	// Description returns a human-readable description of what the filter does
	Description() string
}

// CompositeFilter combines multiple filters with AND or OR logic
type CompositeFilter struct {
	Filters []Filter
	Logic   LogicOperator
}

// LogicOperator defines how multiple filters are combined
type LogicOperator string

const (
	LogicAND LogicOperator = "AND"
	LogicOR  LogicOperator = "OR"
)

// Apply applies all filters according to the logic operator
func (f *CompositeFilter) Apply(res *resource.Resource) bool {
	if len(f.Filters) == 0 {
		return true
	}

	if f.Logic == LogicOR {
		for _, filter := range f.Filters {
			if filter.Apply(res) {
				return true
			}
		}
		return false
	}

	// Default to AND logic
	for _, filter := range f.Filters {
		if !filter.Apply(res) {
			return false
		}
	}
	return true
}

func (f *CompositeFilter) Description() string {
	if len(f.Filters) == 0 {
		return "no filters"
	}

	descriptions := make([]string, len(f.Filters))
	for i, filter := range f.Filters {
		descriptions[i] = filter.Description()
	}

	return fmt.Sprintf("(%s)", strings.Join(descriptions, fmt.Sprintf(" %s ", f.Logic)))
}

// TagFilter filters resources by tags
type TagFilter struct {
	Key      string
	Value    string
	Operator TagOperator
}

// TagOperator defines how tag values are compared
type TagOperator string

const (
	TagExists    TagOperator = "exists"     // Tag key exists
	TagNotExists TagOperator = "not-exists" // Tag key does not exist
	TagEquals    TagOperator = "equals"     // Tag value equals exactly
	TagContains  TagOperator = "contains"   // Tag value contains substring
	TagMatches   TagOperator = "matches"    // Tag value matches regex
)

func (f *TagFilter) Apply(res *resource.Resource) bool {
	if res.Tags == nil {
		return f.Operator == TagNotExists
	}

	value, exists := res.Tags[f.Key]

	switch f.Operator {
	case TagExists:
		return exists
	case TagNotExists:
		return !exists
	case TagEquals:
		return exists && value == f.Value
	case TagContains:
		return exists && strings.Contains(value, f.Value)
	case TagMatches:
		if !exists {
			return false
		}
		matched, err := regexp.MatchString(f.Value, value)
		return err == nil && matched
	}

	return false
}

func (f *TagFilter) Description() string {
	switch f.Operator {
	case TagExists:
		return fmt.Sprintf("tag[%s] exists", f.Key)
	case TagNotExists:
		return fmt.Sprintf("tag[%s] not exists", f.Key)
	case TagEquals:
		return fmt.Sprintf("tag[%s] = %s", f.Key, f.Value)
	case TagContains:
		return fmt.Sprintf("tag[%s] contains %s", f.Key, f.Value)
	case TagMatches:
		return fmt.Sprintf("tag[%s] matches /%s/", f.Key, f.Value)
	}
	return "unknown tag filter"
}

// RegexFilter filters resources by regex pattern matching
type RegexFilter struct {
	Field   string // "name", "id", "type", "provider", "region"
	Pattern *regexp.Regexp
}

func NewRegexFilter(field, pattern string) (*RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern: %w", err)
	}
	return &RegexFilter{
		Field:   field,
		Pattern: re,
	}, nil
}

func (f *RegexFilter) Apply(res *resource.Resource) bool {
	var value string
	switch f.Field {
	case "name":
		value = res.Name
	case "id":
		value = res.ID
	case "type":
		value = string(res.Type)
	case "provider":
		value = res.Provider
	case "region":
		value = res.Region
	case "account":
		value = res.Account
	default:
		// Try to get from properties
		if prop, ok := res.Properties[f.Field]; ok {
			value = fmt.Sprintf("%v", prop)
		}
	}

	return f.Pattern.MatchString(value)
}

func (f *RegexFilter) Description() string {
	return fmt.Sprintf("%s matches /%s/", f.Field, f.Pattern.String())
}

// DateRangeFilter filters resources by creation/update date
type DateRangeFilter struct {
	Field string // "created", "updated"
	From  *time.Time
	To    *time.Time
}

func (f *DateRangeFilter) Apply(res *resource.Resource) bool {
	var date *time.Time

	switch f.Field {
	case "created":
		date = res.CreatedAt
	case "updated":
		date = res.UpdatedAt
	default:
		return false
	}

	if date == nil {
		return false
	}

	if f.From != nil && date.Before(*f.From) {
		return false
	}

	if f.To != nil && date.After(*f.To) {
		return false
	}

	return true
}

func (f *DateRangeFilter) Description() string {
	var parts []string
	if f.From != nil {
		parts = append(parts, fmt.Sprintf("%s >= %s", f.Field, f.From.Format(time.RFC3339)))
	}
	if f.To != nil {
		parts = append(parts, fmt.Sprintf("%s <= %s", f.Field, f.To.Format(time.RFC3339)))
	}
	return strings.Join(parts, " AND ")
}

// StateFilter filters resources by state/status
type StateFilter struct {
	States []string // List of acceptable states
}

func (f *StateFilter) Apply(res *resource.Resource) bool {
	if len(f.States) == 0 {
		return true
	}

	// Try to find state in various property names
	stateKeys := []string{"state", "status", "provisioning_state", "lifecycle_state"}

	for _, key := range stateKeys {
		if state, ok := res.Properties[key]; ok {
			stateStr := strings.ToLower(fmt.Sprintf("%v", state))
			for _, allowedState := range f.States {
				if strings.ToLower(allowedState) == stateStr {
					return true
				}
			}
		}
	}

	return false
}

func (f *StateFilter) Description() string {
	return fmt.Sprintf("state in [%s]", strings.Join(f.States, ", "))
}

// PropertyFilter filters resources by arbitrary property values
type PropertyFilter struct {
	Path     string // JSONPath-like path to property (e.g., "vm_size", "properties.enabled")
	Operator CompareOperator
	Value    interface{}
}

// CompareOperator defines how values are compared
type CompareOperator string

const (
	OpEquals             CompareOperator = "="
	OpNotEquals          CompareOperator = "!="
	OpGreaterThan        CompareOperator = ">"
	OpGreaterThanOrEqual CompareOperator = ">="
	OpLessThan           CompareOperator = "<"
	OpLessThanOrEqual    CompareOperator = "<="
	OpContains           CompareOperator = "contains"
	OpStartsWith         CompareOperator = "starts-with"
	OpEndsWith           CompareOperator = "ends-with"
)

func (f *PropertyFilter) Apply(res *resource.Resource) bool {
	value := f.getPropertyValue(res)
	if value == nil {
		return false
	}

	return f.compare(value, f.Value)
}

func (f *PropertyFilter) getPropertyValue(res *resource.Resource) interface{} {
	// Handle simple property paths
	parts := strings.Split(f.Path, ".")

	var current interface{} = res.Properties
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}

	return current
}

func (f *PropertyFilter) compare(actual, expected interface{}) bool {
	switch f.Operator {
	case OpEquals:
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case OpNotEquals:
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case OpContains:
		return strings.Contains(strings.ToLower(fmt.Sprintf("%v", actual)), strings.ToLower(fmt.Sprintf("%v", expected)))
	case OpStartsWith:
		return strings.HasPrefix(strings.ToLower(fmt.Sprintf("%v", actual)), strings.ToLower(fmt.Sprintf("%v", expected)))
	case OpEndsWith:
		return strings.HasSuffix(strings.ToLower(fmt.Sprintf("%v", actual)), strings.ToLower(fmt.Sprintf("%v", expected)))
	case OpGreaterThan, OpGreaterThanOrEqual, OpLessThan, OpLessThanOrEqual:
		return f.compareNumeric(actual, expected)
	}

	return false
}

func (f *PropertyFilter) compareNumeric(actual, expected interface{}) bool {
	actualNum, err1 := toFloat64(actual)
	expectedNum, err2 := toFloat64(expected)

	if err1 != nil || err2 != nil {
		return false
	}

	switch f.Operator {
	case OpGreaterThan:
		return actualNum > expectedNum
	case OpGreaterThanOrEqual:
		return actualNum >= expectedNum
	case OpLessThan:
		return actualNum < expectedNum
	case OpLessThanOrEqual:
		return actualNum <= expectedNum
	}

	return false
}

func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func (f *PropertyFilter) Description() string {
	return fmt.Sprintf("%s %s %v", f.Path, f.Operator, f.Value)
}

// CostFilter filters resources by cost (when cost data is available)
type CostFilter struct {
	MinCost *float64
	MaxCost *float64
}

func (f *CostFilter) Apply(res *resource.Resource) bool {
	// Try to get cost from properties
	costValue, ok := res.Properties["cost"]
	if !ok {
		costValue, ok = res.Properties["monthly_cost"]
	}
	if !ok {
		// No cost data, exclude by default
		return false
	}

	cost, err := toFloat64(costValue)
	if err != nil {
		return false
	}

	if f.MinCost != nil && cost < *f.MinCost {
		return false
	}

	if f.MaxCost != nil && cost > *f.MaxCost {
		return false
	}

	return true
}

func (f *CostFilter) Description() string {
	var parts []string
	if f.MinCost != nil {
		parts = append(parts, fmt.Sprintf("cost >= %.2f", *f.MinCost))
	}
	if f.MaxCost != nil {
		parts = append(parts, fmt.Sprintf("cost <= %.2f", *f.MaxCost))
	}
	if len(parts) == 0 {
		return "cost filter (any)"
	}
	return strings.Join(parts, " AND ")
}

// TypeFilter filters resources by type
type TypeFilter struct {
	Types []resource.ResourceType
}

func (f *TypeFilter) Apply(res *resource.Resource) bool {
	if len(f.Types) == 0 {
		return true
	}

	for _, t := range f.Types {
		if res.Type == t {
			return true
		}
	}
	return false
}

func (f *TypeFilter) Description() string {
	if len(f.Types) == 0 {
		return "any type"
	}
	types := make([]string, len(f.Types))
	for i, t := range f.Types {
		types[i] = string(t)
	}
	return fmt.Sprintf("type in [%s]", strings.Join(types, ", "))
}

// ProviderFilter filters resources by provider
type ProviderFilter struct {
	Providers []string
}

func (f *ProviderFilter) Apply(res *resource.Resource) bool {
	if len(f.Providers) == 0 {
		return true
	}

	for _, p := range f.Providers {
		if res.Provider == p {
			return true
		}
	}
	return false
}

func (f *ProviderFilter) Description() string {
	if len(f.Providers) == 0 {
		return "any provider"
	}
	return fmt.Sprintf("provider in [%s]", strings.Join(f.Providers, ", "))
}

// ApplyFilters applies all filters to a collection and returns a new filtered collection
func ApplyFilters(collection *resource.Collection, filters ...Filter) *resource.Collection {
	if len(filters) == 0 {
		return collection
	}

	composite := &CompositeFilter{
		Filters: filters,
		Logic:   LogicAND,
	}

	filtered := resource.NewCollection()
	filtered.Metadata = collection.Metadata

	for _, res := range collection.Resources {
		if composite.Apply(res) {
			filtered.Add(res)
		}
	}

	return filtered
}
