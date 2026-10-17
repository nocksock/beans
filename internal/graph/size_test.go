package graph

import (
	"context"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph/model"
)

func TestQueryBeansWithSize(t *testing.T) {
	resolver, core := setupTestResolver(t)
	ctx := context.Background()

	// Empty size counts as "md"
	for _, b := range []*bean.Bean{
		{ID: "size-xs", Title: "Tiny", Status: "todo", Size: "xs"},
		{ID: "size-md", Title: "Medium explicit", Status: "todo", Size: "md"},
		{ID: "size-none", Title: "Medium implicit", Status: "todo"},
		{ID: "size-xl", Title: "Huge", Status: "todo", Size: "xl"},
	} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	tests := []struct {
		name   string
		filter *model.BeanFilter
		want   []string
	}{
		{"md includes empty size", &model.BeanFilter{Size: []string{"md"}}, []string{"size-md", "size-none"}},
		{"multiple sizes", &model.BeanFilter{Size: []string{"xs", "xl"}}, []string{"size-xs", "size-xl"}},
		{"exclude md excludes empty size", &model.BeanFilter{ExcludeSize: []string{"md"}}, []string{"size-xs", "size-xl"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Query().Beans(ctx, tt.filter)
			if err != nil {
				t.Fatalf("Beans() error = %v", err)
			}
			ids := make(map[string]bool, len(got))
			for _, b := range got {
				ids[b.ID] = true
			}
			if len(got) != len(tt.want) {
				t.Errorf("Beans() = %v, want %v", ids, tt.want)
			}
			for _, id := range tt.want {
				if !ids[id] {
					t.Errorf("Beans() missing %q, got %v", id, ids)
				}
			}
		})
	}
}

func TestBeanSizeResolver(t *testing.T) {
	resolver, _ := setupTestResolver(t)
	ctx := context.Background()

	tests := []struct {
		size string
		want string
	}{
		{"", "md"},
		{"xs", "xs"},
		{"xl", "xl"},
	}
	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			got, err := resolver.Bean().Size(ctx, &bean.Bean{Size: tt.size})
			if err != nil {
				t.Fatalf("Size() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Size() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMutationSize(t *testing.T) {
	resolver, _ := setupTestResolver(t)
	ctx := context.Background()
	mr := resolver.Mutation()

	size := "lg"
	created, err := mr.CreateBean(ctx, model.CreateBeanInput{Title: strPtr("Sized"), Size: &size})
	if err != nil {
		t.Fatalf("CreateBean() error = %v", err)
	}
	if created.Size != "lg" {
		t.Errorf("CreateBean().Size = %q, want %q", created.Size, "lg")
	}

	unsized, err := mr.CreateBean(ctx, model.CreateBeanInput{Title: strPtr("Unsized")})
	if err != nil {
		t.Fatalf("CreateBean() error = %v", err)
	}
	if unsized.Size != "" {
		t.Errorf("CreateBean() without size stored %q, want empty", unsized.Size)
	}

	clear := ""
	updated, err := mr.UpdateBean(ctx, created.ID, model.UpdateBeanInput{Size: &clear})
	if err != nil {
		t.Fatalf("UpdateBean() error = %v", err)
	}
	if updated.Size != "" {
		t.Errorf("UpdateBean() clear size = %q, want empty", updated.Size)
	}

	invalid := "huge"
	if _, err := mr.CreateBean(ctx, model.CreateBeanInput{Title: strPtr("Bad"), Size: &invalid}); err == nil {
		t.Error("CreateBean() with invalid size: expected error")
	}
	if _, err := mr.UpdateBean(ctx, created.ID, model.UpdateBeanInput{Size: &invalid}); err == nil {
		t.Error("UpdateBean() with invalid size: expected error")
	}
}
