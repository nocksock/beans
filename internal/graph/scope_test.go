package graph

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/graph/model"
)

func TestQueryBeansIDPrefix(t *testing.T) {
	resolver, core := setupTestResolver(t)
	for _, id := range []string{"repo-a1b2", "api-c3d4", "api-v2-e5f6", "web-g7h8"} {
		createTestBean(t, core, id, id, "todo")
	}

	tests := []struct {
		name   string
		prefix *string
		want   []string
	}{
		{"nil", nil, []string{"api-c3d4", "api-v2-e5f6", "repo-a1b2", "web-g7h8"}},
		{"empty", strPtr(""), []string{"api-c3d4", "api-v2-e5f6", "repo-a1b2", "web-g7h8"}},
		{"includes nested prefixes", strPtr("api-"), []string{"api-c3d4", "api-v2-e5f6"}},
		{"nested", strPtr("api-v2-"), []string{"api-v2-e5f6"}},
		{"no match", strPtr("mob-"), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Query().Beans(context.Background(), &model.BeanFilter{IDPrefix: tt.prefix})
			if err != nil {
				t.Fatalf("Beans() error = %v", err)
			}
			var ids []string
			for _, b := range got {
				ids = append(ids, b.ID)
			}
			slices.Sort(ids)
			if !slices.Equal(ids, tt.want) {
				t.Errorf("Beans() IDs = %v, want %v", ids, tt.want)
			}
		})
	}
}

func TestMutationCreateBeanPrefix(t *testing.T) {
	resolver, core := setupTestResolver(t)
	core.Config().Beans.Prefix = "repo-"
	core.Config().Beans.IDLength = 6

	tests := []struct {
		name       string
		prefix     *string
		wantPrefix string
	}{
		{"default prefix", nil, "repo-"},
		{"explicit prefix", strPtr("api-"), "api-"},
		{"empty prefix", strPtr(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolver.Mutation().CreateBean(context.Background(), model.CreateBeanInput{
				Title:  strPtr("New bean"),
				Prefix: tt.prefix,
			})
			if err != nil {
				t.Fatalf("CreateBean() error = %v", err)
			}
			if !strings.HasPrefix(got.ID, tt.wantPrefix) || len(got.ID) != len(tt.wantPrefix)+6 {
				t.Errorf("ID = %q, want prefix %q and 6 random chars", got.ID, tt.wantPrefix)
			}
		})
	}
}
