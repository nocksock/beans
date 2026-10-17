package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTree creates files relative to root; keys are slash-separated paths.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadLayer(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		wantCascade bool
		wantKeys    []string
	}{
		{name: "empty file", content: "", wantKeys: nil},
		{name: "no cascade", content: "beans:\n  prefix: x-\n", wantKeys: []string{"beans"}},
		{name: "cascade false", content: "cascade: false\n", wantKeys: []string{"cascade"}},
		{name: "cascade true", content: "cascade: true\nbeans:\n  prefix: x-\n", wantCascade: true, wantKeys: []string{"cascade", "beans"}},
		{name: "invalid yaml", content: "beans: [\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ConfigFileName)
			writeTree(t, dir, map[string]string{ConfigFileName: tt.content})

			l, err := readLayer(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("readLayer() error = %v", err)
			}
			if l.dir != dir {
				t.Errorf("dir = %q, want %q", l.dir, dir)
			}
			if l.cascades() != tt.wantCascade {
				t.Errorf("cascades() = %v, want %v", l.cascades(), tt.wantCascade)
			}
			if len(l.data) != len(tt.wantKeys) {
				t.Errorf("data = %v, want keys %v", l.data, tt.wantKeys)
			}
			for _, k := range tt.wantKeys {
				if _, ok := l.data[k]; !ok {
					t.Errorf("missing key %q", k)
				}
			}
		})
	}
}

func TestFindChain(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		// the example from beans-jzty
		"projects/.beans.yml":      "beans:\n  prefix: projects-\n",
		"projects/a/.beans.yml":    "cascade: true\n",
		"projects/a/aa/.beans.yml": "beans:\n  prefix: aa-\n",
		"projects/b/.beans.yml":    "beans:\n  prefix: b-\n",
		// transitive chain
		"t/.beans.yml":          "cascade: true\n",
		"t/mid/.beans.yml":      "cascade: true\n",
		"t/mid/leaf/.beans.yml": "",
		// chain broken in the middle
		"x/.beans.yml":          "cascade: true\n",
		"x/mid/.beans.yml":      "",
		"x/mid/leaf/.beans.yml": "",
		// no config in between: skips directories without a config file
		"g/.beans.yml":           "cascade: true\n",
		"g/no/config/.beans.yml": "",
	})

	tests := []struct {
		leaf string
		want []string // dirs root-first, relative to root
	}{
		{leaf: "projects", want: []string{"projects"}},
		{leaf: "projects/a", want: []string{"projects/a"}},
		{leaf: "projects/a/aa", want: []string{"projects/a", "projects/a/aa"}},
		{leaf: "projects/b", want: []string{"projects/b"}},
		{leaf: "t/mid/leaf", want: []string{"t", "t/mid", "t/mid/leaf"}},
		{leaf: "x/mid/leaf", want: []string{"x/mid/leaf"}},
		{leaf: "x/mid", want: []string{"x", "x/mid"}},
		{leaf: "g/no/config", want: []string{"g", "g/no/config"}},
	}

	for _, tt := range tests {
		t.Run(tt.leaf, func(t *testing.T) {
			leafPath := filepath.Join(root, filepath.FromSlash(tt.leaf), ConfigFileName)
			layers, err := findChain(leafPath)
			if err != nil {
				t.Fatalf("findChain() error = %v", err)
			}
			var got []string
			for _, l := range layers {
				rel, _ := filepath.Rel(root, l.dir)
				got = append(got, filepath.ToSlash(rel))
			}
			if len(got) != len(tt.want) {
				t.Fatalf("chain = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("chain = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestOriginDir(t *testing.T) {
	layers := []layer{
		{dir: "/root", data: map[string]any{
			"beans":     map[string]any{"path": ".beans", "prefix": "r-"},
			"session":   map[string]any{"check": "c", "launch": "l"},
			"launchers": []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}},
		}},
		{dir: "/root/pkg", data: map[string]any{
			"beans":     map[string]any{"prefix": "p-"},
			"session":   map[string]any{"launch": "l2"},
			"launchers": []any{map[string]any{"name": "b"}},
		}},
	}

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"inherited key", originDir(layers, "beans", "path"), "/root"},
		{"overridden key", originDir(layers, "beans", "prefix"), "/root/pkg"},
		{"missing key", originDir(layers, "beans", "id_length"), ""},
		{"path through scalar", originDir(layers, "beans", "prefix", "x"), ""},
		{"inherited session cmd", originDir(layers, "session", "check"), "/root"},
		{"overridden session cmd", originDir(layers, "session", "launch"), "/root/pkg"},
		{"inherited launcher", originDirOfNamed(layers, "launchers", "a"), "/root"},
		{"overridden launcher", originDirOfNamed(layers, "launchers", "b"), "/root/pkg"},
		{"missing launcher", originDirOfNamed(layers, "launchers", "z"), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}
