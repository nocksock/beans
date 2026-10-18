package bean

import (
	"testing"
	"time"
)

func TestSortBy(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	evenEarlier := now.Add(-2 * time.Hour)

	orders := SortOrders{
		Statuses:   []string{"in-progress", "todo", "draft", "completed", "scrapped"},
		Priorities: []string{"critical", "high", "normal", "low", "deferred"},
		Types:      []string{"milestone", "epic", "bug", "feature", "task"},
		Sizes:      []string{"xs", "sm", "md", "lg", "xl"},
	}

	tests := []struct {
		name  string
		beans []*Bean
		field string
		desc  bool
		want  []string
	}{
		{
			name:  "id",
			beans: []*Bean{{ID: "c3"}, {ID: "a1"}, {ID: "b2"}},
			field: "id",
			want:  []string{"a1", "b2", "c3"},
		},
		{
			name:  "id desc",
			beans: []*Bean{{ID: "c3"}, {ID: "a1"}, {ID: "b2"}},
			field: "id",
			desc:  true,
			want:  []string{"c3", "b2", "a1"},
		},
		{
			name: "size, empty counts as md, ties by id",
			beans: []*Bean{
				{ID: "xl", Size: "xl"}, {ID: "none"}, {ID: "xs", Size: "xs"},
				{ID: "md", Size: "md"}, {ID: "lg", Size: "lg"},
			},
			field: "size",
			want:  []string{"xs", "md", "none", "lg", "xl"},
		},
		{
			name: "size desc, ties still by id",
			beans: []*Bean{
				{ID: "xs", Size: "xs"}, {ID: "none"}, {ID: "md", Size: "md"}, {ID: "xl", Size: "xl"},
			},
			field: "size",
			desc:  true,
			want:  []string{"xl", "md", "none", "xs"},
		},
		{
			name: "created asc is oldest first",
			beans: []*Bean{
				{ID: "new", CreatedAt: &now}, {ID: "old", CreatedAt: &evenEarlier}, {ID: "mid", CreatedAt: &earlier},
			},
			field: "created",
			want:  []string{"old", "mid", "new"},
		},
		{
			name: "created desc is newest first, nil last",
			beans: []*Bean{
				{ID: "nil", CreatedAt: nil}, {ID: "old", CreatedAt: &evenEarlier}, {ID: "new", CreatedAt: &now},
			},
			field: "created",
			desc:  true,
			want:  []string{"new", "old", "nil"},
		},
		{
			name: "updated desc",
			beans: []*Bean{
				{ID: "old", UpdatedAt: &evenEarlier}, {ID: "new", UpdatedAt: &now}, {ID: "mid", UpdatedAt: &earlier},
			},
			field: "updated",
			desc:  true,
			want:  []string{"new", "mid", "old"},
		},
		{
			name: "status by configured order",
			beans: []*Bean{
				{ID: "c1", Status: "completed"}, {ID: "t1", Status: "todo"},
				{ID: "i1", Status: "in-progress"}, {ID: "t2", Status: "todo"},
			},
			field: "status",
			want:  []string{"i1", "t1", "t2", "c1"},
		},
		{
			name: "priority, empty counts as normal",
			beans: []*Bean{
				{ID: "low", Priority: "low"}, {ID: "none"}, {ID: "crit", Priority: "critical"},
			},
			field: "priority",
			want:  []string{"crit", "none", "low"},
		},
		{
			name: "type by configured order",
			beans: []*Bean{
				{ID: "t", Type: "task"}, {ID: "b", Type: "bug"}, {ID: "m", Type: "milestone"},
			},
			field: "type",
			want:  []string{"m", "b", "t"},
		},
		{
			name: "title case-insensitive",
			beans: []*Bean{
				{ID: "1", Title: "beta"}, {ID: "2", Title: "Alpha"}, {ID: "3", Title: "gamma"},
			},
			field: "title",
			want:  []string{"2", "1", "3"},
		},
		{
			name: "empty field uses default order",
			beans: []*Bean{
				{ID: "completed-bug", Status: "completed", Type: "bug"},
				{ID: "todo-task", Status: "todo", Type: "task"},
				{ID: "todo-bug", Status: "todo", Type: "bug"},
			},
			field: "",
			want:  []string{"todo-bug", "todo-task", "completed-bug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortBy(tt.beans, tt.field, tt.desc, orders)
			for i, id := range tt.want {
				if tt.beans[i].ID != id {
					got := make([]string, len(tt.beans))
					for j, b := range tt.beans {
						got[j] = b.ID
					}
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestDefaultSortDesc(t *testing.T) {
	tests := map[string]bool{
		"created": true, "updated": true,
		"status": false, "priority": false, "type": false, "size": false, "title": false, "id": false,
	}
	for field, want := range tests {
		if got := DefaultSortDesc(field); got != want {
			t.Errorf("DefaultSortDesc(%q) = %v, want %v", field, got, want)
		}
	}
}
