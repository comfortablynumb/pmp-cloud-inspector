package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// ParseTagFilter parses tag filter expressions
// Examples:
//   - "Environment" -> tag exists
//   - "Environment=production" -> tag equals
//   - "Environment~prod" -> tag contains
//   - "Environment=/prod.*/" -> tag matches regex
func ParseTagFilter(expr string) (*TagFilter, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty tag filter expression")
	}

	// Check for operators
	if strings.Contains(expr, "=/") && strings.HasSuffix(expr, "/") {
		// Regex match: Environment=/prod.*/
		parts := strings.SplitN(expr, "=/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag regex filter: %s", expr)
		}
		pattern := strings.TrimSuffix(parts[1], "/")
		return &TagFilter{
			Key:      parts[0],
			Value:    pattern,
			Operator: TagMatches,
		}, nil
	}

	if strings.Contains(expr, "!=") {
		// Not exists (if no value) or not equals
		parts := strings.SplitN(expr, "!=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag filter: %s", expr)
		}
		if parts[1] == "" {
			return &TagFilter{
				Key:      parts[0],
				Operator: TagNotExists,
			}, nil
		}
		// Note: not-equals is handled by negating equals
		return nil, fmt.Errorf("tag != operator not directly supported, use regex instead")
	}

	if strings.Contains(expr, "~") {
		// Contains: Environment~prod
		parts := strings.SplitN(expr, "~", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag contains filter: %s", expr)
		}
		return &TagFilter{
			Key:      parts[0],
			Value:    parts[1],
			Operator: TagContains,
		}, nil
	}

	if strings.Contains(expr, "=") {
		// Equals: Environment=production
		parts := strings.SplitN(expr, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid tag equals filter: %s", expr)
		}
		return &TagFilter{
			Key:      parts[0],
			Value:    parts[1],
			Operator: TagEquals,
		}, nil
	}

	// Just key name means tag exists
	return &TagFilter{
		Key:      expr,
		Operator: TagExists,
	}, nil
}

// ParseRegexFilter parses regex filter expressions
// Examples:
//   - "name:/prod-.*/" -> match name
//   - "id:/^i-[0-9a-f]+$/" -> match id
func ParseRegexFilter(expr string) (*RegexFilter, error) {
	parts := strings.SplitN(expr, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid regex filter format, expected field:/pattern/: %s", expr)
	}

	pattern := parts[1]
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
		pattern = strings.TrimSuffix(pattern, "/")
	}

	return NewRegexFilter(parts[0], pattern)
}

// ParseDateRangeFilter parses date range filter expressions
// Examples:
//   - "created:2024-01-01..2024-12-31"
//   - "created:>2024-01-01"
//   - "updated:<2024-12-31"
func ParseDateRangeFilter(expr string) (*DateRangeFilter, error) {
	parts := strings.SplitN(expr, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid date filter format, expected field:range: %s", expr)
	}

	field := parts[0]
	rangeExpr := parts[1]

	filter := &DateRangeFilter{Field: field}

	// Handle range: 2024-01-01..2024-12-31
	if strings.Contains(rangeExpr, "..") {
		dates := strings.Split(rangeExpr, "..")
		if len(dates) != 2 {
			return nil, fmt.Errorf("invalid date range: %s", rangeExpr)
		}

		from, err := parseDate(dates[0])
		if err != nil {
			return nil, fmt.Errorf("invalid from date: %w", err)
		}
		filter.From = &from

		to, err := parseDate(dates[1])
		if err != nil {
			return nil, fmt.Errorf("invalid to date: %w", err)
		}
		filter.To = &to

		return filter, nil
	}

	// Handle comparisons
	if strings.HasPrefix(rangeExpr, ">") {
		dateStr := strings.TrimPrefix(rangeExpr, ">")
		from, err := parseDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date: %w", err)
		}
		filter.From = &from
		return filter, nil
	}

	if strings.HasPrefix(rangeExpr, "<") {
		dateStr := strings.TrimPrefix(rangeExpr, "<")
		to, err := parseDate(dateStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date: %w", err)
		}
		filter.To = &to
		return filter, nil
	}

	// Single date means exactly that day
	date, err := parseDate(rangeExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	filter.From = &date
	endOfDay := date.Add(24 * time.Hour)
	filter.To = &endOfDay
	return filter, nil
}

func parseDate(s string) (time.Time, error) {
	// Try different formats
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// ParseStateFilter parses state filter expressions
// Examples:
//   - "running,stopped"
//   - "active"
func ParseStateFilter(expr string) (*StateFilter, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty state filter")
	}

	states := strings.Split(expr, ",")
	for i, state := range states {
		states[i] = strings.TrimSpace(state)
	}

	return &StateFilter{States: states}, nil
}

// ParsePropertyFilter parses property filter expressions
// Examples:
//   - "vm_size=Standard_D2s_v3"
//   - "enabled=true"
//   - "logins_count>100"
func ParsePropertyFilter(expr string) (*PropertyFilter, error) {
	// Determine operator
	operators := []struct {
		op     CompareOperator
		symbol string
	}{
		{OpGreaterThanOrEqual, ">="},
		{OpLessThanOrEqual, "<="},
		{OpNotEquals, "!="},
		{OpEquals, "="},
		{OpGreaterThan, ">"},
		{OpLessThan, "<"},
		{OpContains, "~"},
		{OpStartsWith, "^="},
		{OpEndsWith, "$="},
	}

	for _, opDef := range operators {
		if strings.Contains(expr, opDef.symbol) {
			parts := strings.SplitN(expr, opDef.symbol, 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid property filter: %s", expr)
			}

			return &PropertyFilter{
				Path:     parts[0],
				Operator: opDef.op,
				Value:    parseValue(parts[1]),
			}, nil
		}
	}

	return nil, fmt.Errorf("no operator found in property filter: %s", expr)
}

func parseValue(s string) interface{} {
	// Try to parse as number
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}

	// Try to parse as bool
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}

	// Return as string
	return s
}

// ParseCostFilter parses cost filter expressions
// Examples:
//   - "100..500" -> between 100 and 500
//   - ">100" -> greater than 100
//   - "<500" -> less than 500
func ParseCostFilter(expr string) (*CostFilter, error) {
	filter := &CostFilter{}

	// Handle range: 100..500
	if strings.Contains(expr, "..") {
		parts := strings.Split(expr, "..")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid cost range: %s", expr)
		}

		min, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid min cost: %w", err)
		}
		filter.MinCost = &min

		max, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid max cost: %w", err)
		}
		filter.MaxCost = &max

		return filter, nil
	}

	// Handle comparisons
	if strings.HasPrefix(expr, ">") {
		costStr := strings.TrimPrefix(expr, ">")
		min, err := strconv.ParseFloat(costStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid cost: %w", err)
		}
		filter.MinCost = &min
		return filter, nil
	}

	if strings.HasPrefix(expr, "<") {
		costStr := strings.TrimPrefix(expr, "<")
		max, err := strconv.ParseFloat(costStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid cost: %w", err)
		}
		filter.MaxCost = &max
		return filter, nil
	}

	return nil, fmt.Errorf("invalid cost filter format: %s", expr)
}

// ParseTypeFilter parses type filter expression
func ParseTypeFilter(types []string) *TypeFilter {
	resourceTypes := make([]resource.ResourceType, len(types))
	for i, t := range types {
		resourceTypes[i] = resource.ResourceType(t)
	}
	return &TypeFilter{Types: resourceTypes}
}

// ParseProviderFilter parses provider filter expression
func ParseProviderFilter(providers []string) *ProviderFilter {
	return &ProviderFilter{Providers: providers}
}

// ParseJSONPathQuery parses a JSONPath-like query
// This is a simplified implementation supporting basic path traversal
// Examples:
//   - "$.properties.vm_size" -> get vm_size from properties
//   - "$.tags.Environment" -> get Environment tag
func ParseJSONPathQuery(query string, collection *resource.Collection) ([]*resource.Resource, error) {
	// Remove leading $.
	query = strings.TrimPrefix(query, "$.")

	if query == "" {
		return collection.Resources, nil
	}

	// For now, use this as a property filter
	// More advanced JSONPath would require a dedicated library
	re, err := regexp.Compile(".*")
	if err != nil {
		return nil, err
	}

	filter := &RegexFilter{
		Field:   query,
		Pattern: re,
	}

	result := []*resource.Resource{}
	for _, res := range collection.Resources {
		if filter.Apply(res) {
			result = append(result, res)
		}
	}

	return result, nil
}
