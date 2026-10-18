package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

func TestSortBeans(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	evenEarlier := now.Add(-2 * time.Hour)

	// Statuses are now hardcoded, so we just use default config
	testCfg := config.Default()

	t.Run("sort by id", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c3"},
			{ID: "a1"},
			{ID: "b2"},
		}
		sortBeans(beans, "id", testCfg)

		if beans[0].ID != "a1" || beans[1].ID != "b2" || beans[2].ID != "c3" {
			t.Errorf("sort by id: got [%s, %s, %s], want [a1, b2, c3]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by size", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "xl", Size: "xl"},
			{ID: "none"},
			{ID: "xs", Size: "xs"},
			{ID: "md", Size: "md"},
			{ID: "lg", Size: "lg"},
		}
		sortBeans(beans, "size", testCfg)

		// Smallest first; empty size counts as md and ties sort by ID
		want := []string{"xs", "md", "none", "lg", "xl"}
		for i, id := range want {
			if beans[i].ID != id {
				t.Errorf("sort by size: position %d = %s, want %s", i, beans[i].ID, id)
			}
		}
	})

	t.Run("sort by created", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "old", CreatedAt: &evenEarlier},
			{ID: "new", CreatedAt: &now},
			{ID: "mid", CreatedAt: &earlier},
		}
		sortBeans(beans, "created", testCfg)

		// Should be newest first
		if beans[0].ID != "new" || beans[1].ID != "mid" || beans[2].ID != "old" {
			t.Errorf("sort by created: got [%s, %s, %s], want [new, mid, old]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by created with nil", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "nil1", CreatedAt: nil},
			{ID: "has", CreatedAt: &now},
			{ID: "nil2", CreatedAt: nil},
		}
		sortBeans(beans, "created", testCfg)

		// Non-nil should come first, then nil sorted by ID
		if beans[0].ID != "has" {
			t.Errorf("sort by created with nil: first should be \"has\", got %q", beans[0].ID)
		}
	})

	t.Run("sort by updated", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "old", UpdatedAt: &evenEarlier},
			{ID: "new", UpdatedAt: &now},
			{ID: "mid", UpdatedAt: &earlier},
		}
		sortBeans(beans, "updated", testCfg)

		// Should be newest first
		if beans[0].ID != "new" || beans[1].ID != "mid" || beans[2].ID != "old" {
			t.Errorf("sort by updated: got [%s, %s, %s], want [new, mid, old]",
				beans[0].ID, beans[1].ID, beans[2].ID)
		}
	})

	t.Run("sort by status", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "c1", Status: "completed"},
			{ID: "t1", Status: "todo"},
			{ID: "i1", Status: "in-progress"},
			{ID: "t2", Status: "todo"},
		}
		sortBeans(beans, "status", testCfg)

		// Should be ordered by status config order (in-progress, todo, draft, completed, scrapped), then by ID within same status
		expected := []string{"i1", "t1", "t2", "c1"}
		for i, want := range expected {
			if beans[i].ID != want {
				t.Errorf("sort by status[%d]: got %q, want %q", i, beans[i].ID, want)
			}
		}
	})

	t.Run("default sort (archive status then type)", func(t *testing.T) {
		beans := []*bean.Bean{
			{ID: "completed-bug", Status: "completed", Type: "bug"},
			{ID: "todo-feature", Status: "todo", Type: "feature"},
			{ID: "todo-task", Status: "todo", Type: "task"},
			{ID: "completed-task", Status: "completed", Type: "task"},
			{ID: "todo-bug", Status: "todo", Type: "bug"},
		}
		sortBeans(beans, "", testCfg)

		// Should be: non-archive first (sorted by type order from DefaultTypes: milestone, epic, bug, feature, task),
		// then archive (sorted by type)
		// DefaultTypes order: milestone, epic, bug, feature, task
		expected := []string{"todo-bug", "todo-feature", "todo-task", "completed-bug", "completed-task"}
		for i, want := range expected {
			if beans[i].ID != want {
				t.Errorf("default sort[%d]: got %q, want %q", i, beans[i].ID, want)
			}
		}
	})
}

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
