package cmd

import (
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestRenderBeanPrompt(t *testing.T) {
	testCore, cleanup := setupQueryTestCore(t)
	defer cleanup()

	parent := &bean.Bean{ID: "p-1", Slug: "parent", Title: "Parent Epic", Status: "todo", Type: "epic"}
	main := &bean.Bean{
		ID:       "m-1",
		Slug:     "main",
		Title:    "Main Task",
		Status:   "todo",
		Type:     "feature",
		Priority: "high",
		Tags:     []string{"cli", "agents"},
		Parent:   "p-1",
		Blocking: []string{"b-1"},
		Body:     "Do the thing.\n\n- [ ] step one",
	}
	blocked := &bean.Bean{ID: "b-1", Slug: "blocked", Title: "Blocked Task", Status: "draft", Type: "task"}
	blocker := &bean.Bean{ID: "x-1", Slug: "blocker", Title: "Blocker Task", Status: "in-progress", Type: "task", Blocking: []string{"m-1"}}
	child := &bean.Bean{ID: "c-1", Slug: "child", Title: "Child Task", Status: "completed", Type: "task", Parent: "m-1"}
	lonely := &bean.Bean{ID: "l-1", Slug: "lonely", Title: "Lonely Task", Status: "todo", Type: "task"}

	for _, b := range []*bean.Bean{parent, main, blocked, blocker, child, lonely} {
		if err := testCore.Create(b); err != nil {
			t.Fatalf("failed to create bean %s: %v", b.ID, err)
		}
	}

	tests := []struct {
		name        string
		bean        *bean.Bean
		extra       string
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:  "bean with all relationship kinds",
			bean:  main,
			extra: "",
			wantContain: []string{
				"m-1",
				"Main Task",
				"feature",
				"high",
				"cli, agents",
				"Do the thing.",
				"- [ ] step one",
				"p-1: (parent) [todo] Parent Epic",
				"b-1: (blocking) [draft] Blocked Task",
				"x-1: (blocked by) [in-progress] Blocker Task",
				"c-1: (child) [completed] Child Task",
				"beans show <id>",
				"beans update m-1 --status in-progress",
				"beans update m-1 --status completed",
			},
			wantAbsent: []string{"Additional instructions"},
		},
		{
			name:        "bean without relationships omits related section",
			bean:        lonely,
			wantContain: []string{"l-1", "Lonely Task", "beans update l-1 --status completed"},
			wantAbsent:  []string{"Related beans", "(parent)", "(child)"},
		},
		{
			name:        "extra text from config is appended",
			bean:        lonely,
			extra:       "Always run mise test before committing.",
			wantContain: []string{"Additional instructions", "Always run mise test before committing."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderBeanPrompt(tt.bean, tt.extra)
			if err != nil {
				t.Fatalf("renderBeanPrompt() error = %v", err)
			}
			for _, want := range tt.wantContain {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q\n---\n%s", want, got)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("output unexpectedly contains %q\n---\n%s", absent, got)
				}
			}
		})
	}
}
