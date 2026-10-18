package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hmans/beans/internal/config"
)

func TestHiddenStatuses(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), config.ConfigFileName)
	if err := os.WriteFile(configPath, []byte("statuses:\n  completed:\n    - name: merged\n    - name: released\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		status  []string
		include []string
		want    []string
		wantErr bool
	}{
		{
			name: "hides done statuses and their substatuses by default",
			want: []string{"completed", "merged", "released", "scrapped", "archived"},
		},
		{
			name:    "include builtin includes its substatuses",
			include: []string{"completed"},
			want:    []string{"scrapped", "archived"},
		},
		{
			name:    "include all",
			include: []string{"completed", "scrapped", "archived"},
			want:    []string{},
		},
		{
			name:    "include archived",
			include: []string{"archived"},
			want:    []string{"completed", "merged", "released", "scrapped"},
		},
		{
			name:    "include substatus only",
			include: []string{"merged"},
			want:    []string{"completed", "released", "scrapped", "archived"},
		},
		{
			name:   "explicit status filter shows hidden status and its substatuses",
			status: []string{"completed"},
			want:   []string{"scrapped", "archived"},
		},
		{
			name:   "explicit status filter for a substatus shows it",
			status: []string{"merged"},
			want:   []string{"completed", "released", "scrapped", "archived"},
		},
		{
			name:    "invalid include value",
			include: []string{"todo"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hiddenStatuses(cfg, tt.status, tt.include)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListIDPrefix(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	files := map[string]string{
		".beans.yml":     "cascade: true\nbeans:\n  prefix: repo-\n",
		"api/.beans.yml": "beans:\n  prefix: api-\n",
	}
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name string
		dir  string
		all  bool
		want string // "" means no prefix filter
	}{
		{"cascade root is unscoped", ".", false, ""},
		{"nested config is scoped", "api", false, "api-"},
		{"--all disables scope", "api", true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.LoadFromDirectory(filepath.Join(root, tt.dir))
			if err != nil {
				t.Fatal(err)
			}
			got := listIDPrefix(cfg, tt.all)
			if tt.want == "" {
				if got != nil {
					t.Errorf("listIDPrefix() = %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != tt.want {
				t.Errorf("listIDPrefix() = %v, want %q", got, tt.want)
			}
		})
	}
}
