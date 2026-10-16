package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

func TestPreviewView(t *testing.T) {
	parent := &bean.Bean{ID: "beans-prnt", Title: "Parent Bean"}
	tests := []struct {
		name    string
		bean    *bean.Bean
		ctx     previewContext
		want    []string
		notWant []string
	}{
		{
			name: "empty",
			want: []string{"No bean selected"},
		},
		{
			name: "full bean",
			bean: &bean.Bean{
				ID: "beans-test", Title: "Test Bean", Status: "todo", Type: "feature",
				Priority: "high", Tags: []string{"frontend", "design"}, Body: "## Summary\n\nThis is the body.",
				Blocking: []string{"beans-blkd"},
			},
			ctx:  previewContext{parent: parent, childCount: 3},
			want: []string{"beans-test", "Test Bean", "todo", "feature", "high", "frontend", "design", "Summary", "This is the body", "Parent Bean", "Children", "3", "beans-blkd"},
		},
		{
			name:    "normal priority is hidden",
			bean:    &bean.Bean{ID: "beans-test", Title: "T", Status: "todo", Type: "task", Priority: "normal"},
			notWant: []string{"normal"},
		},
		{
			name: "empty body",
			bean: &bean.Bean{ID: "beans-test", Title: "T", Status: "todo", Type: "task"},
			want: []string{"No description"},
		},
		{
			name: "missing parent shows ID",
			bean: &bean.Bean{ID: "beans-test", Title: "T", Status: "todo", Type: "task", Parent: "beans-gone"},
			want: []string{"Parent", "beans-gone"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newPreviewModel(config.Default())
			m.setBean(tt.bean, tt.ctx)
			view := ansi.Strip(m.View(70, 30))
			for _, s := range tt.want {
				if !strings.Contains(view, s) {
					t.Errorf("preview should contain %q:\n%s", s, view)
				}
			}
			for _, s := range tt.notWant {
				if strings.Contains(view, s) {
					t.Errorf("preview should not contain %q:\n%s", s, view)
				}
			}
		})
	}
}

func TestPreviewViewExactSize(t *testing.T) {
	long := &bean.Bean{
		ID: "beans-long", Title: strings.Repeat("Very long title ", 10), Status: "todo", Type: "task",
		Body: strings.Repeat("A paragraph of body text that goes on and on.\n\n", 50),
	}
	sizes := []struct{ w, h int }{{80, 40}, {50, 10}, {40, 5}}
	for _, b := range []*bean.Bean{nil, long} {
		for _, sz := range sizes {
			m := newPreviewModel(config.Default())
			m.setBean(b, previewContext{})
			view := m.View(sz.w, sz.h)
			if got := lipgloss.Height(view); got != sz.h {
				t.Errorf("%dx%d: height = %d, want %d", sz.w, sz.h, got, sz.h)
			}
			if got := lipgloss.Width(view); got != sz.w {
				t.Errorf("%dx%d: width = %d, want %d", sz.w, sz.h, got, sz.w)
			}
		}
	}
}
