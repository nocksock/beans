package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// KeyDisabled is the binding value that unbinds an action.
const KeyDisabled = "<disabled>"

// Keybindings maps a TUI context (e.g. "list") to its action overrides
// (e.g. "changeStatus" -> ["s"]). Which contexts and actions exist is up to
// the TUI, which validates them.
type Keybindings map[string]map[string]KeyList

// KeyList is the keys bound to one action. In YAML it is a single key or a
// list of keys; KeyDisabled or an empty list unbinds the action (an empty,
// non-nil KeyList).
type KeyList []string

// UnmarshalYAML accepts a scalar or a sequence of scalars.
func (k *KeyList) UnmarshalYAML(node *yaml.Node) error {
	var keys []string
	switch node.Kind {
	case yaml.ScalarNode:
		keys = []string{node.Value}
	case yaml.SequenceNode:
		if err := node.Decode(&keys); err != nil {
			return err
		}
	default:
		return fmt.Errorf("a key binding must be a key or a list of keys")
	}

	*k = KeyList{}
	for _, key := range keys {
		switch key {
		case KeyDisabled:
			// Unbinds the action; any other keys listed with it are ignored
			*k = KeyList{}
			return nil
		case "":
			return fmt.Errorf("empty key in key binding")
		}
		*k = append(*k, key)
	}
	return nil
}
