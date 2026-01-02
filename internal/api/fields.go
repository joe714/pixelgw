package api

import (
	"strings"
)

// ParseFields parses the fields query parameter.
// Returns nil if param is nil or empty (use defaults).
// Returns ["*"] if param is "*" (include all fields).
// Otherwise returns the list of requested field names.
func ParseFields(param *string) []string {
	if param == nil || *param == "" {
		return nil
	}
	if *param == "*" {
		return []string{"*"}
	}
	fields := strings.Split(*param, ",")
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}

// ShouldInclude checks if a field should be included in the response.
// fields: the parsed fields list (nil for defaults, ["*"] for all, or specific field names)
// fieldName: the field to check
// defaultInclude: whether to include this field by default when fields is nil
func ShouldInclude(fields []string, fieldName string, defaultInclude bool) bool {
	// No fields specified - use default behavior
	if fields == nil {
		return defaultInclude
	}
	// "*" means include all fields
	for _, f := range fields {
		if f == "*" {
			return true
		}
	}
	// Check if field is in the list
	for _, f := range fields {
		if f == fieldName {
			return true
		}
	}
	return false
}
