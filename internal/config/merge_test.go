package config

import (
	"reflect"
	"testing"
)

func TestMergeData(t *testing.T) {
	tests := []struct {
		name string
		base map[string]any
		over map[string]any
		want map[string]any
	}{
		{
			name: "nil base",
			base: nil,
			over: map[string]any{"a": 1},
			want: map[string]any{"a": 1},
		},
		{
			name: "nil over",
			base: map[string]any{"a": 1},
			over: nil,
			want: map[string]any{"a": 1},
		},
		{
			name: "scalar override",
			base: map[string]any{"a": 1, "b": "keep"},
			over: map[string]any{"a": 2},
			want: map[string]any{"a": 2, "b": "keep"},
		},
		{
			name: "nested maps merge recursively",
			base: map[string]any{"beans": map[string]any{"prefix": "root-", "id_length": 4}},
			over: map[string]any{"beans": map[string]any{"prefix": "pkg-"}},
			want: map[string]any{"beans": map[string]any{"prefix": "pkg-", "id_length": 4}},
		},
		{
			name: "plain lists are replaced",
			base: map[string]any{"tags": []any{"a", "b"}},
			over: map[string]any{"tags": []any{"c"}},
			want: map[string]any{"tags": []any{"c"}},
		},
		{
			name: "keyed lists merge by name",
			base: map[string]any{"launchers": []any{
				map[string]any{"name": "a", "exec": "root-a"},
				map[string]any{"name": "b", "exec": "root-b", "multiple": true},
			}},
			over: map[string]any{"launchers": []any{
				map[string]any{"name": "c", "exec": "pkg-c"},
				map[string]any{"name": "b", "exec": "pkg-b"},
			}},
			want: map[string]any{"launchers": []any{
				map[string]any{"name": "a", "exec": "root-a"},
				map[string]any{"name": "b", "exec": "pkg-b"},
				map[string]any{"name": "c", "exec": "pkg-c"},
			}},
		},
		{
			name: "list with unnamed items is replaced",
			base: map[string]any{"l": []any{map[string]any{"name": "a"}}},
			over: map[string]any{"l": []any{map[string]any{"exec": "x"}}},
			want: map[string]any{"l": []any{map[string]any{"exec": "x"}}},
		},
		{
			name: "type mismatch: over wins",
			base: map[string]any{"session": map[string]any{"check": "x"}},
			over: map[string]any{"session": "nope"},
			want: map[string]any{"session": "nope"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeData(tt.base, tt.over)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mergeData() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMergeDataDoesNotMutateInputs(t *testing.T) {
	base := map[string]any{"beans": map[string]any{"prefix": "root-"}}
	over := map[string]any{"beans": map[string]any{"prefix": "pkg-"}}

	mergeData(base, over)

	if got := base["beans"].(map[string]any)["prefix"]; got != "root-" {
		t.Errorf("base was mutated: prefix = %v", got)
	}
}
