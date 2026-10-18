package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCascadingParent(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".beans.yml":       cascadingRoot,
		"plain/.beans.yml": "beans:\n  prefix: plain-\n",
		"plain/sub/.keep":  "",
		"pkg/.keep":        "",
		"pkg/deeper/.keep": "",
		"standalone/.keep": "",
	})
	// A standalone tree without any config above it
	alone := t.TempDir()

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{"directly below cascading root", filepath.Join(root, "pkg"), filepath.Join(root, ConfigFileName)},
		{"further below cascading root", filepath.Join(root, "pkg", "deeper"), filepath.Join(root, ConfigFileName)},
		{"below a non-cascading config", filepath.Join(root, "plain", "sub"), ""},
		{"no config above", alone, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CascadingParent(tt.dir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("CascadingParent(%s) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestSaveSettings(t *testing.T) {
	dir := t.TempDir()
	err := SaveSettings(dir, map[string]any{
		"beans.prefix":    "api-",
		"beans.id_length": 6,
		"prompt.extra":    "be nice",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	want := "beans:\n  id_length: 6\n  prefix: api-\nprompt:\n  extra: be nice\n"
	if string(raw) != want {
		t.Errorf("file = %q, want %q", raw, want)
	}
}

func TestSaveSettingsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSettings(dir, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ConfigFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "{}\n" {
		t.Errorf("file = %q, want %q", raw, "{}\n")
	}
}
