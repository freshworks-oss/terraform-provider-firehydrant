package provider

import (
	"encoding/json"
	"fmt"
)

// stringValue dereferences an optional SDK string, returning the empty string when
// the API omitted the field or returned it as null.
func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// boolValue dereferences an optional SDK bool, returning false when the API omitted
// the field or returned it as null.
func boolValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

// stringsToList converts an SDK string slice into the []interface{} a Terraform
// TypeList expects. The result is always non-nil so an absent list and an empty
// list produce the same state, avoiding a spurious diff.
func stringsToList(in []string) []interface{} {
	out := make([]interface{}, 0, len(in))
	for _, v := range in {
		out = append(out, v)
	}
	return out
}

// unmarshalLabels takes an SDK labels map (map[string]any and converts it
// into a map[string]string for Terraform compatibility.
// We need to convert any non-string values to strings since Terraform's TypeMap expects strings.
func unmarshalLabels(labelsMap interface{}) (map[string]string, error) {
	if labelsMap == nil {
		return nil, nil
	}

	switch labels := labelsMap.(type) {
	case map[string]any:
		// If the map is empty, return nil (no labels set)
		if len(labels) == 0 {
			return nil, nil
		}

		// Convert all values to strings
		stringMap := make(map[string]string, len(labels))
		for key, value := range labels {
			switch v := value.(type) {
			case string:
				stringMap[key] = v
			case nil:
				stringMap[key] = ""
			default:
				// Convert any non-string values to their JSON representation
				jsonValue, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal label value for key %s: %w", key, err)
				}
				stringMap[key] = string(jsonValue)
			}
		}
		return stringMap, nil

	default:
		// Fallback: try to marshal/unmarshal for older SDK versions with empty structs
		jsonBytes, err := json.Marshal(labelsMap)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal labels: %w", err)
		}

		var rawMap map[string]interface{}
		if err := json.Unmarshal(jsonBytes, &rawMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}

		if len(rawMap) == 0 {
			return nil, nil
		}

		stringMap := make(map[string]string, len(rawMap))
		for key, value := range rawMap {
			switch v := value.(type) {
			case string:
				stringMap[key] = v
			case nil:
				stringMap[key] = ""
			default:
				jsonValue, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal label value for key %s: %w", key, err)
				}
				stringMap[key] = string(jsonValue)
			}
		}
		return stringMap, nil
	}
}
