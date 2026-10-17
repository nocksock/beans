package config

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const customStatusesConfig = `beans:
  prefix: test-
statuses:
  in-progress:
    - name: review
      color: magenta
      description: Awaiting review
    - name: implementing
  completed:
    - name: merged
`

func loadConfigString(t *testing.T, content string) (*Config, error) {
	t.Helper()
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{ConfigFileName: content})
	return Load(filepath.Join(dir, ConfigFileName))
}

func mustLoadConfigString(t *testing.T, content string) *Config {
	t.Helper()
	cfg, err := loadConfigString(t, content)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return cfg
}

func TestCustomStatuses(t *testing.T) {
	cfg := mustLoadConfigString(t, customStatusesConfig)

	t.Run("sorted after their parent", func(t *testing.T) {
		got := cfg.StatusNames()
		want := []string{"in-progress", "review", "implementing", "todo", "draft", "completed", "merged", "scrapped"}
		for _, s := range DefaultStatuses {
			if !slices.Contains(want, s.Name) {
				// Builtins added later (e.g. archived) go after the ones we know about
				want = append(want, s.Name)
			}
		}
		if !slices.Equal(got, want) {
			t.Errorf("StatusNames() = %v, want %v", got, want)
		}
	})

	t.Run("valid", func(t *testing.T) {
		for _, s := range []string{"review", "implementing", "merged", "todo"} {
			if !cfg.IsValidStatus(s) {
				t.Errorf("IsValidStatus(%q) = false, want true", s)
			}
		}
		if cfg.IsValidStatus("nope") {
			t.Error("IsValidStatus(\"nope\") = true, want false")
		}
	})

	t.Run("listed", func(t *testing.T) {
		if !strings.Contains(cfg.StatusList(), "in-progress, review, implementing") {
			t.Errorf("StatusList() = %q", cfg.StatusList())
		}
	})

	t.Run("explicit fields", func(t *testing.T) {
		s := cfg.GetStatus("review")
		if s == nil {
			t.Fatal("GetStatus(\"review\") = nil")
		}
		if s.Color != "magenta" || s.Description != "Awaiting review" || s.Parent != "in-progress" {
			t.Errorf("GetStatus(\"review\") = %+v", *s)
		}
	})

	t.Run("inherited fields", func(t *testing.T) {
		s := cfg.GetStatus("merged")
		if s == nil {
			t.Fatal("GetStatus(\"merged\") = nil")
		}
		if s.Color != "gray" {
			t.Errorf("merged color = %q, want gray (from completed)", s.Color)
		}
		if !s.Archive || !cfg.IsArchiveStatus("merged") {
			t.Error("merged should inherit archive from completed")
		}
		if cfg.IsArchiveStatus("review") {
			t.Error("review should not be an archive status")
		}
	})

	t.Run("builtins are untouched", func(t *testing.T) {
		if got := cfg.GetStatus("in-progress"); got == nil || got.Parent != "" {
			t.Errorf("GetStatus(\"in-progress\") = %+v, want builtin without parent", got)
		}
		if DefaultStatuses[0].Parent != "" {
			t.Error("DefaultStatuses mutated")
		}
	})
}

func TestStatusCategory(t *testing.T) {
	cfg := mustLoadConfigString(t, customStatusesConfig)

	tests := []struct {
		status string
		want   string
	}{
		{"review", "in-progress"},
		{"merged", "completed"},
		{"in-progress", "in-progress"},
		{"todo", "todo"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := cfg.StatusCategory(tt.status); got != tt.want {
				t.Errorf("StatusCategory(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestExpandStatuses(t *testing.T) {
	cfg := mustLoadConfigString(t, customStatusesConfig)

	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"builtin includes substatuses", []string{"in-progress"}, []string{"in-progress", "review", "implementing"}},
		{"custom is exact", []string{"review"}, []string{"review"}},
		{"builtin without substatuses", []string{"todo"}, []string{"todo"}},
		{"several", []string{"completed", "todo"}, []string{"completed", "merged", "todo"}},
		{"no duplicates", []string{"review", "in-progress"}, []string{"review", "in-progress", "implementing"}},
		{"unknown kept", []string{"nope"}, []string{"nope"}},
		{"empty", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.ExpandStatuses(tt.in); !slices.Equal(got, tt.want) {
				t.Errorf("ExpandStatuses(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCustomStatusesDefaultConfig(t *testing.T) {
	cfg := Default()
	if len(cfg.Statuses()) != len(DefaultStatuses) {
		t.Errorf("Statuses() = %d entries, want %d", len(cfg.Statuses()), len(DefaultStatuses))
	}
	if got := cfg.ExpandStatuses([]string{"in-progress"}); !slices.Equal(got, []string{"in-progress"}) {
		t.Errorf("ExpandStatuses() = %v", got)
	}
}

func TestCustomStatusesValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  string
		wantErr string
	}{
		{
			name:    "unknown parent",
			config:  "statuses:\n  review:\n    - name: foo\n",
			wantErr: "'review' is not a builtin status",
		},
		{
			name:    "shadows builtin",
			config:  "statuses:\n  in-progress:\n    - name: todo\n",
			wantErr: "'todo' is a builtin status",
		},
		{
			name:    "duplicate",
			config:  "statuses:\n  in-progress:\n    - name: review\n  todo:\n    - name: review\n",
			wantErr: "'review' is defined more than once",
		},
		{
			name:    "missing name",
			config:  "statuses:\n  in-progress:\n    - color: red\n",
			wantErr: "needs a name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadConfigString(t, tt.config)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCustomStatusesCascade(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".beans.yml": `cascade: true
statuses:
  in-progress:
    - name: review
      color: magenta
  completed:
    - name: merged
`,
		"pkg/.beans.yml": `statuses:
  in-progress:
    - name: review
      color: red
    - name: testing
`,
	})

	cfg, err := LoadFromDirectory(filepath.Join(root, "pkg"))
	if err != nil {
		t.Fatalf("LoadFromDirectory() error = %v", err)
	}

	if got := cfg.ExpandStatuses([]string{"in-progress"}); !slices.Equal(got, []string{"in-progress", "review", "testing"}) {
		t.Errorf("in-progress substatuses = %v", got)
	}
	if s := cfg.GetStatus("review"); s == nil || s.Color != "red" {
		t.Errorf("review = %+v, want leaf color red", s)
	}
	if !cfg.IsValidStatus("merged") {
		t.Error("merged should be inherited from the root config")
	}
}
