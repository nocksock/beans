package config

import "path/filepath"

// findChain returns the layers that make up the config at leafPath, root-first:
// the leaf itself plus each ancestor config, walking upward while the next
// config found has cascade enabled.
func findChain(leafPath string) ([]layer, error) {
	leaf, err := readLayer(leafPath)
	if err != nil {
		return nil, err
	}

	chain := []layer{leaf}
	dir := leaf.dir
	for {
		parentPath, err := findConfigAbove(dir)
		if err != nil {
			return nil, err
		}
		if parentPath == "" {
			break
		}

		parent, err := readLayer(parentPath)
		if err != nil {
			return nil, err
		}
		if !parent.cascades() {
			break
		}

		chain = append([]layer{parent}, chain...)
		dir = parent.dir
	}
	return chain, nil
}

// findConfigAbove finds the nearest config file strictly above dir.
func findConfigAbove(dir string) (string, error) {
	parent := filepath.Dir(dir)
	if parent == dir {
		return "", nil
	}
	return FindConfig(parent)
}
