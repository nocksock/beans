package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrCascading is returned when writing a setting to a config file that takes
// part in cascading, where it's unclear which file the setting belongs in.
var ErrCascading = errors.New("can't change settings of a cascading config yet")

// splitKey splits a dotted setting key like "beans.prefix" into its parts.
func splitKey(key string) ([]string, error) {
	keys := strings.Split(key, ".")
	for _, k := range keys {
		if k == "" {
			return nil, fmt.Errorf("invalid setting key %q", key)
		}
	}
	return keys, nil
}

// Get returns the effective value of the setting at the dotted key, e.g.
// "beans.prefix". List items can be addressed by index or by name
// ("launchers.claude.exec"). An empty key returns the whole config.
// Settings that are valid but unset return their zero value.
func (c *Config) Get(key string) (any, error) {
	v := reflect.ValueOf(*c)
	if key == "" {
		return v.Interface(), nil
	}

	keys, err := splitKey(key)
	if err != nil {
		return nil, err
	}
	for i, k := range keys {
		if v, err = child(v, k); err != nil {
			return nil, fmt.Errorf("%s: %w", strings.Join(keys[:i+1], "."), err)
		}
	}
	return deref(v).Interface(), nil
}

// child returns the value below v at key k.
func child(v reflect.Value, k string) (reflect.Value, error) {
	v = deref(v)
	switch v.Kind() {
	case reflect.Struct:
		if index, ok := yamlField(v.Type(), k); ok {
			return v.FieldByIndex(index), nil
		}
	case reflect.Map:
		if item := v.MapIndex(reflect.ValueOf(k).Convert(v.Type().Key())); item.IsValid() {
			return item, nil
		}
		return reflect.Value{}, errors.New("not set")
	case reflect.Slice:
		if item, ok := sliceItem(v, k); ok {
			return item, nil
		}
		return reflect.Value{}, errors.New("no such item")
	default:
		return reflect.Value{}, errors.New("not a nested setting")
	}
	return reflect.Value{}, errors.New("unknown setting")
}

// deref follows pointers, treating nil as the zero value of the pointed-to type.
func deref(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return reflect.Zero(v.Type().Elem())
		}
		v = v.Elem()
	}
	return v
}

// yamlField finds the exported struct field serialized under name, looking
// into inlined structs, and returns its index path.
func yamlField(t reflect.Type, name string) ([]int, bool) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag, opts, _ := strings.Cut(f.Tag.Get("yaml"), ",")
		if opts == "inline" {
			if index, ok := yamlField(f.Type, name); ok {
				return append([]int{i}, index...), true
			}
			continue
		}
		if tag == name {
			return []int{i}, true
		}
	}
	return nil, false
}

// sliceItem finds a list item by index or, for structs, by their Name field.
func sliceItem(v reflect.Value, k string) (reflect.Value, bool) {
	if i, err := strconv.Atoi(k); err == nil {
		if i < 0 || i >= v.Len() {
			return reflect.Value{}, false
		}
		return v.Index(i), true
	}
	for i := 0; i < v.Len(); i++ {
		item := deref(v.Index(i))
		if item.Kind() != reflect.Struct {
			continue
		}
		if name := item.FieldByName("Name"); name.IsValid() && name.Kind() == reflect.String && name.String() == k {
			return item, true
		}
	}
	return reflect.Value{}, false
}

// SetSetting sets the setting at the dotted key in the config file at path,
// keeping the rest of the file, including comments, intact. The value is
// parsed as YAML, so "4" becomes a number and "true" a boolean.
// Only map keys can be set; list items can't be addressed.
// Returns ErrCascading if the file inherits from or passes settings down to
// other config files.
func SetSetting(path, key, value string) error {
	keys, err := splitKey(key)
	if err != nil {
		return err
	}

	chain, err := findChain(path)
	if err != nil {
		return err
	}
	if len(chain) > 1 || chain[0].cascades() {
		return ErrCascading
	}

	val, err := parseValue(value)
	if err != nil {
		return err
	}
	if err := checkKeyPath(keys); err != nil {
		return err
	}
	if err := checkValueType(keys, val); err != nil {
		return fmt.Errorf("invalid value for %s: %w", key, err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if doc.Kind == 0 {
		// Empty file
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	if err := setNode(doc.Content[0], keys, val); err != nil {
		return fmt.Errorf("setting %s: %w", key, err)
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, out.Bytes(), 0644)
}

// parseValue parses a command line value as a YAML value node.
func parseValue(value string) (*yaml.Node, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(value), &doc); err != nil {
		return nil, fmt.Errorf("invalid value %q: %w", value, err)
	}
	if doc.Kind == 0 {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: ""}, nil
	}
	return doc.Content[0], nil
}

// checkValueType checks that val fits the type of the setting at keys, by
// decoding the setting on its own into a config file.
func checkValueType(keys []string, val *yaml.Node) error {
	fragment := &yaml.Node{Kind: yaml.MappingNode}
	if err := setNode(fragment, keys, val); err != nil {
		return err
	}

	var file settingsFile
	err := fragment.Decode(&file)
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		return errors.New(strings.Join(typeErr.Errors, "; "))
	}
	return err
}

// settingsFile is everything that can be set in a config file.
type settingsFile struct {
	Config  `yaml:",inline"`
	Cascade bool `yaml:"cascade"`
}

// checkKeyPath checks that keys lead through maps to a known setting.
func checkKeyPath(keys []string) error {
	t := reflect.TypeOf(settingsFile{})
	for i, k := range keys {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Struct:
			index, ok := yamlField(t, k)
			if !ok {
				return fmt.Errorf("%s: unknown setting", strings.Join(keys[:i+1], "."))
			}
			t = t.FieldByIndex(index).Type
		case reflect.Map:
			t = t.Elem()
		case reflect.Slice:
			return fmt.Errorf("%s: list items can't be set", strings.Join(keys[:i], "."))
		default:
			return fmt.Errorf("%s: not a nested setting", strings.Join(keys[:i], "."))
		}
	}
	return nil
}

// setNode sets keys to val below the mapping node m, creating missing maps.
func setNode(m *yaml.Node, keys []string, val *yaml.Node) error {
	for i, k := range keys {
		if m.Kind != yaml.MappingNode {
			return fmt.Errorf("%s is not a map", strings.Join(keys[:i], "."))
		}
		last := i == len(keys)-1

		existing := mappingValue(m, k)
		switch {
		case existing != nil && last:
			replaceNode(existing, val)
		case existing != nil:
			if existing.Tag == "!!null" {
				// An empty section like "tui:" becomes a map
				replaceNode(existing, &yaml.Node{Kind: yaml.MappingNode})
			}
			m = existing
		case last:
			m.Content = append(m.Content, keyNode(k), val)
		default:
			next := &yaml.Node{Kind: yaml.MappingNode}
			m.Content = append(m.Content, keyNode(k), next)
			m = next
		}
	}
	return nil
}

// mappingValue returns the value node for key k in mapping m, or nil.
func mappingValue(m *yaml.Node, k string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == k {
			return m.Content[i+1]
		}
	}
	return nil
}

func keyNode(k string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}
}

// replaceNode overwrites dst with src, keeping dst's comments.
func replaceNode(dst, src *yaml.Node) {
	head, line, foot := dst.HeadComment, dst.LineComment, dst.FootComment
	*dst = *src
	dst.HeadComment, dst.LineComment, dst.FootComment = head, line, foot
}
