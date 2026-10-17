package config

// mergeData merges over into base and returns the result. Neither input is mutated.
//
// Rules:
//   - maps merge recursively
//   - lists whose items are all maps with a "name" key merge by name: an item in
//     over replaces the same-named item in base, new names are appended
//   - anything else (scalars, other lists, type mismatches): over wins
func mergeData(base, over map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(over))
	for k, v := range base {
		result[k] = v
	}
	for k, overVal := range over {
		result[k] = mergeValue(result[k], overVal)
	}
	return result
}

func mergeValue(baseVal, overVal any) any {
	switch over := overVal.(type) {
	case map[string]any:
		if base, ok := baseVal.(map[string]any); ok {
			return mergeData(base, over)
		}
	case []any:
		if base, ok := baseVal.([]any); ok && isNamedList(base) && isNamedList(over) {
			return mergeNamedLists(base, over)
		}
	}
	return overVal
}

// isNamedList reports whether every item is a map with a string "name" key.
func isNamedList(items []any) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if itemName(item) == "" {
			return false
		}
	}
	return true
}

func itemName(item any) string {
	m, ok := item.(map[string]any)
	if !ok {
		return ""
	}
	name, _ := m["name"].(string)
	return name
}

// mergeNamedLists replaces same-named items in place and appends new ones,
// keeping base order first.
func mergeNamedLists(base, over []any) []any {
	result := make([]any, len(base), len(base)+len(over))
	copy(result, base)

	index := make(map[string]int, len(base))
	for i, item := range result {
		index[itemName(item)] = i
	}

	for _, item := range over {
		name := itemName(item)
		if i, ok := index[name]; ok {
			result[i] = item
			continue
		}
		index[name] = len(result)
		result = append(result, item)
	}
	return result
}
