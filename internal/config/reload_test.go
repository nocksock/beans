package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, ConfigFileName)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReload(t *testing.T) {
	tests := []struct {
		name       string
		initial    string // config file content at load time; "" = no file
		updated    string // config file content at reload time; "" = no file
		wantPrefix string
		wantErr    bool
	}{
		{name: "picks up changes", initial: "beans:\n  prefix: old-\n", updated: "beans:\n  prefix: new-\n", wantPrefix: "new-"},
		{name: "picks up newly created file", updated: "beans:\n  prefix: new-\n", wantPrefix: "new-"},
		{name: "falls back to defaults when file is removed", initial: "beans:\n  prefix: old-\n", wantPrefix: ""},
		{name: "invalid file is an error", initial: "beans:\n  prefix: old-\n", updated: "beans: [", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ConfigFileName)
			if tt.initial != "" {
				writeConfig(t, dir, tt.initial)
			}
			cfg, err := LoadFromDirectory(dir)
			if err != nil {
				t.Fatal(err)
			}

			if tt.updated != "" {
				writeConfig(t, dir, tt.updated)
			} else {
				os.Remove(path)
			}

			got, err := cfg.Reload()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Beans.Prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", got.Beans.Prefix, tt.wantPrefix)
			}
			if got.ConfigDir() != dir {
				t.Errorf("ConfigDir() = %q, want %q", got.ConfigDir(), dir)
			}
		})
	}
}

func TestWatch(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "project")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}

	changed := make(chan struct{}, 10)
	stop, err := Watch(dir, func() { changed <- struct{}{} })
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	expect := func(what string, want bool) {
		t.Helper()
		select {
		case <-changed:
			if !want {
				t.Fatalf("%s: unexpected change notification", what)
			}
		case <-time.After(500 * time.Millisecond):
			if want {
				t.Fatalf("%s: no change notification", what)
			}
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "other.yml"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	expect("unrelated file", false)

	writeConfig(t, dir, "beans:\n  prefix: a-\n")
	expect("config in start dir", true)

	writeConfig(t, parent, "cascade: true\n")
	expect("config in parent dir", true)
}
