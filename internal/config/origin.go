package config

// originDir returns the directory of the closest layer that sets the given
// key path, or "" if no layer sets it. Layers are ordered root-first.
func originDir(layers []layer, keys ...string) string {
	for i := len(layers) - 1; i >= 0; i-- {
		if hasKeyPath(layers[i].data, keys) {
			return layers[i].dir
		}
	}
	return ""
}

// originDirOfNamed returns the directory of the closest layer whose list at
// listKey contains an item with the given name, or "" if none does.
func originDirOfNamed(layers []layer, listKey, name string) string {
	for i := len(layers) - 1; i >= 0; i-- {
		items, _ := layers[i].data[listKey].([]any)
		for _, item := range items {
			if itemName(item) == name {
				return layers[i].dir
			}
		}
	}
	return ""
}

func hasKeyPath(data map[string]any, keys []string) bool {
	current := data
	for i, k := range keys {
		v, ok := current[k]
		if !ok {
			return false
		}
		if i == len(keys)-1 {
			return true
		}
		if current, ok = v.(map[string]any); !ok {
			return false
		}
	}
	return false
}
