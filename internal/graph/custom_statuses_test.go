package graph

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"testing"

	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph/model"
)

func setupCustomStatusResolver(t *testing.T) (*Resolver, *beancore.Core) {
	t.Helper()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, config.ConfigFileName)
	configYAML := `statuses:
  in-progress:
    - name: review
  completed:
    - name: merged
`
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	beansDir := cfg.ResolveBeansPath()
	if err := os.MkdirAll(beansDir, 0755); err != nil {
		t.Fatal(err)
	}
	core := beancore.New(beansDir, cfg)
	if err := core.Load(); err != nil {
		t.Fatalf("failed to load core: %v", err)
	}

	createTestBean(t, core, "s-wip", "Working", "in-progress")
	createTestBean(t, core, "s-review", "In review", "review")
	createTestBean(t, core, "s-todo", "Todo", "todo")
	createTestBean(t, core, "s-done", "Done", "completed")
	createTestBean(t, core, "s-merged", "Merged", "merged")

	return &Resolver{Core: core}, core
}

func TestCustomStatusFilters(t *testing.T) {
	resolver, _ := setupCustomStatusResolver(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		filter *model.BeanFilter
		want   []string
	}{
		{"builtin includes substatus", &model.BeanFilter{Status: []string{"in-progress"}}, []string{"s-review", "s-wip"}},
		{"custom status is exact", &model.BeanFilter{Status: []string{"review"}}, []string{"s-review"}},
		{"exclude builtin excludes substatus", &model.BeanFilter{ExcludeStatus: []string{"completed"}}, []string{"s-review", "s-todo", "s-wip"}},
		{"exclude custom status is exact", &model.BeanFilter{ExcludeStatus: []string{"merged"}}, []string{"s-done", "s-review", "s-todo", "s-wip"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Query().Beans(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Beans() error = %v", err)
			}
			ids := make([]string, len(got))
			for i, b := range got {
				ids[i] = b.ID
			}
			sort.Strings(ids)
			if !slices.Equal(ids, tt.want) {
				t.Errorf("Beans() = %v, want %v", ids, tt.want)
			}
		})
	}
}

func TestStatusCategoryResolver(t *testing.T) {
	resolver, core := setupCustomStatusResolver(t)
	ctx := context.Background()
	br := resolver.Bean()

	tests := []struct {
		id   string
		want string
	}{
		{"s-review", "in-progress"},
		{"s-merged", "completed"},
		{"s-todo", "todo"},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			b, err := core.Get(tt.id)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			got, err := br.StatusCategory(ctx, b)
			if err != nil {
				t.Fatalf("StatusCategory() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("StatusCategory() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCustomArchiveStatusDoesNotBlock(t *testing.T) {
	resolver, core := setupCustomStatusResolver(t)
	ctx := context.Background()

	blocked := createTestBean(t, core, "s-blocked", "Blocked", "todo")
	blocked.BlockedBy = []string{"s-merged"}
	if err := core.Update(blocked); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	isBlocked := true
	got, err := resolver.Query().Beans(ctx, &model.BeanFilter{IsBlocked: &isBlocked})
	if err != nil {
		t.Fatalf("Beans() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a bean blocked only by a merged bean should not count as blocked, got %d", len(got))
	}
}
