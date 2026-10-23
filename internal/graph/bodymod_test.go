package graph

import (
	"context"
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph/model"
)

func TestMutationUpdateBeanBodyMod(t *testing.T) {
	const original = "## Tasks\n\n- [ ] one\n- [ ] two\n- [ ] dup\n- [ ] dup"

	tests := []struct {
		name    string
		body    *string
		mod     *model.BodyModification
		want    string
		wantErr string
	}{
		{
			name: "single replacement",
			mod:  &model.BodyModification{Replace: []*model.ReplaceOperation{{Old: "- [ ] one", New: "- [x] one"}}},
			want: "## Tasks\n\n- [x] one\n- [ ] two\n- [ ] dup\n- [ ] dup",
		},
		{
			name: "replacements run in order on the previous result",
			mod: &model.BodyModification{Replace: []*model.ReplaceOperation{
				{Old: "- [ ] one", New: "- [ ] uno"},
				{Old: "uno", New: "eins"},
			}},
			want: "## Tasks\n\n- [ ] eins\n- [ ] two\n- [ ] dup\n- [ ] dup",
		},
		{
			name: "empty new deletes the match",
			mod:  &model.BodyModification{Replace: []*model.ReplaceOperation{{Old: "\n- [ ] two", New: ""}}},
			want: "## Tasks\n\n- [ ] one\n- [ ] dup\n- [ ] dup",
		},
		{
			name: "append adds a blank line separator",
			mod:  &model.BodyModification{Append: strPtr("## Notes")},
			want: original + "\n\n## Notes",
		},
		{
			name: "replace then append",
			mod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{{Old: "- [ ] two", New: "- [x] two"}},
				Append:  strPtr("done"),
			},
			want: "## Tasks\n\n- [ ] one\n- [x] two\n- [ ] dup\n- [ ] dup\n\ndone",
		},
		{
			name:    "old text not found",
			mod:     &model.BodyModification{Replace: []*model.ReplaceOperation{{Old: "missing", New: "x"}}},
			wantErr: "text not found",
		},
		{
			name:    "old text found more than once",
			mod:     &model.BodyModification{Replace: []*model.ReplaceOperation{{Old: "- [ ] dup", New: "x"}}},
			wantErr: "found 2 times",
		},
		{
			name:    "empty old text",
			mod:     &model.BodyModification{Replace: []*model.ReplaceOperation{{Old: "", New: "x"}}},
			wantErr: "cannot be empty",
		},
		{
			name: "failing later replacement saves nothing",
			mod: &model.BodyModification{
				Replace: []*model.ReplaceOperation{{Old: "- [ ] one", New: "- [x] one"}, {Old: "missing", New: "x"}},
				Append:  strPtr("never"),
			},
			wantErr: "replacement 1 failed",
		},
		{
			name:    "body and bodyMod together",
			body:    strPtr("new"),
			mod:     &model.BodyModification{Append: strPtr("x")},
			wantErr: "cannot specify both body and bodyMod",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, core := setupTestResolver(t)
			if err := core.Create(&bean.Bean{ID: "mod-1", Title: "Mod", Status: "todo", Body: original}); err != nil {
				t.Fatalf("create: %v", err)
			}

			status := "in-progress"
			got, err := resolver.Mutation().UpdateBean(context.Background(), "mod-1",
				model.UpdateBeanInput{Status: &status, Body: tt.body, BodyMod: tt.mod})

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("UpdateBean() error = %v, want it to contain %q", err, tt.wantErr)
				}
				stored, _ := core.Get("mod-1")
				if stored.Body != original || stored.Status != "todo" {
					t.Errorf("failed update changed the bean: status %q, body %q", stored.Status, stored.Body)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateBean() error = %v", err)
			}
			if got.Body != tt.want {
				t.Errorf("Body = %q, want %q", got.Body, tt.want)
			}
			if got.Status != status {
				t.Errorf("Status = %q, want %q", got.Status, status)
			}
		})
	}
}
