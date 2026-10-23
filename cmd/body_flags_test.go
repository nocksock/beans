package cmd

import (
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph/model"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// runWithFlags sets flags on a command, runs it, and resets every flag afterwards
// so the package-level flag variables don't leak into other tests.
func runWithFlags(t *testing.T, cmd *cobra.Command, args []string, flags map[string]string) error {
	t.Helper()
	oldCfg := cfg
	cfg = config.Default()
	t.Cleanup(func() {
		cfg = oldCfg
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if sv, ok := f.Value.(pflag.SliceValue); ok {
				_ = sv.Replace(nil)
			} else {
				_ = f.Value.Set(f.DefValue)
			}
			f.Changed = false
		})
	})
	for name, value := range flags {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatalf("setting --%s: %v", name, err)
		}
	}
	if err := cmd.ValidateFlagGroups(); err != nil {
		return err
	}
	return cmd.RunE(cmd, args)
}

func TestResolveContentUnescapes(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"plain", "hello", "hello"},
		{"newline", `a\nb`, "a\nb"},
		{"tab and backslash", `a\tb\\c`, "a\tb\\c"},
		{"unknown escape kept", `a\rb`, `a\rb`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveContent(tt.value, "")
			if err != nil {
				t.Fatalf("resolveContent() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildBodyModification(t *testing.T) {
	tests := []struct {
		name        string
		replace     bool
		old, new    string
		appendSet   bool
		appendValue string
		wantReplace []model.ReplaceOperation
		wantAppend  *string
	}{
		{
			name:    "replace only, escapes interpreted",
			replace: true, old: `- [ ] a\n- [ ] b`, new: `- [x] a\n- [x] b`,
			wantReplace: []model.ReplaceOperation{{Old: "- [ ] a\n- [ ] b", New: "- [x] a\n- [x] b"}},
		},
		{
			name:    "replace with empty new deletes",
			replace: true, old: "gone",
			wantReplace: []model.ReplaceOperation{{Old: "gone", New: ""}},
		},
		{
			name:      "append only, escapes interpreted",
			appendSet: true, appendValue: `## Notes\n\nhi`,
			wantAppend: strPtr("## Notes\n\nhi"),
		},
		{
			name:    "replace and append",
			replace: true, old: "a", new: "b",
			appendSet: true, appendValue: "c",
			wantReplace: []model.ReplaceOperation{{Old: "a", New: "b"}},
			wantAppend:  strPtr("c"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildBodyModification(tt.replace, tt.old, tt.new, tt.appendSet, tt.appendValue)
			if err != nil {
				t.Fatalf("buildBodyModification() error = %v", err)
			}
			if len(got.Replace) != len(tt.wantReplace) {
				t.Fatalf("Replace = %v, want %v", got.Replace, tt.wantReplace)
			}
			for i, op := range got.Replace {
				if *op != tt.wantReplace[i] {
					t.Errorf("Replace[%d] = %+v, want %+v", i, *op, tt.wantReplace[i])
				}
			}
			if (got.Append == nil) != (tt.wantAppend == nil) ||
				(got.Append != nil && *got.Append != *tt.wantAppend) {
				t.Errorf("Append = %v, want %v", got.Append, tt.wantAppend)
			}
		})
	}
}

func TestUpdateBodyFlags(t *testing.T) {
	const original = "## Tasks\n\n- [ ] one\n- [ ] dup\n- [ ] dup"

	tests := []struct {
		name    string
		flags   map[string]string
		want    string
		wantErr string
	}{
		{
			name:  "append",
			flags: map[string]string{"body-append": `## Notes\n\nhi`},
			want:  original + "\n\n## Notes\n\nhi",
		},
		{
			name:  "replace",
			flags: map[string]string{"body-replace-old": "- [ ] one", "body-replace-new": "- [x] one"},
			want:  "## Tasks\n\n- [x] one\n- [ ] dup\n- [ ] dup",
		},
		{
			name: "replace and append together",
			flags: map[string]string{
				"body-replace-old": `## Tasks\n\n`, "body-replace-new": "",
				"body-append": "end",
			},
			want: "- [ ] one\n- [ ] dup\n- [ ] dup\n\nend",
		},
		{
			name:  "full body replacement unescapes",
			flags: map[string]string{"body": `new\nbody`},
			want:  "new\nbody",
		},
		{
			name:    "replace text not found",
			flags:   map[string]string{"body-replace-old": "missing", "body-replace-new": "x"},
			wantErr: "text not found",
		},
		{
			name:    "replace text found twice",
			flags:   map[string]string{"body-replace-old": "- [ ] dup", "body-replace-new": "x"},
			wantErr: "found 2 times",
		},
		{
			name:    "replace-old without replace-new",
			flags:   map[string]string{"body-replace-old": "one"},
			wantErr: "must all be set",
		},
		{
			name:    "body with body-append",
			flags:   map[string]string{"body": "x", "body-append": "y"},
			wantErr: "none of the others can be",
		},
		{
			name:    "body-file with body-replace",
			flags:   map[string]string{"body-file": "f.md", "body-replace-old": "a", "body-replace-new": "b"},
			wantErr: "none of the others can be",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCore, cleanup := setupQueryTestCore(t)
			defer cleanup()
			if err := testCore.Create(&bean.Bean{ID: "upd-1", Title: "Upd", Status: "todo", Body: original}); err != nil {
				t.Fatalf("create: %v", err)
			}

			err := runWithFlags(t, updateCmd, []string{"upd-1"}, tt.flags)

			stored, getErr := testCore.Get("upd-1")
			if getErr != nil {
				t.Fatalf("get: %v", getErr)
			}
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("update error = %v, want it to contain %q", err, tt.wantErr)
				}
				if stored.Body != original {
					t.Errorf("failed update changed the body to %q", stored.Body)
				}
				return
			}
			if err != nil {
				t.Fatalf("update error = %v", err)
			}
			if stored.Body != tt.want {
				t.Errorf("Body = %q, want %q", stored.Body, tt.want)
			}
		})
	}
}

func TestCreateFlags(t *testing.T) {
	tests := []struct {
		name       string
		flags      map[string]string
		wantPrefix string
		wantBody   string
	}{
		{"configured prefix", map[string]string{}, "cfg-", ""},
		{"explicit prefix", map[string]string{"prefix": "api-"}, "api-", ""},
		{"empty prefix", map[string]string{"prefix": ""}, "", ""},
		{"body unescapes", map[string]string{"body": `a\nb`}, "cfg-", "a\nb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCore, cleanup := setupQueryTestCore(t)
			defer cleanup()
			testCore.Config().Beans.Prefix = "cfg-"
			testCore.Config().Beans.IDLength = 4

			if err := runWithFlags(t, createCmd, []string{"New", "bean"}, tt.flags); err != nil {
				t.Fatalf("create error = %v", err)
			}

			all := testCore.All()
			if len(all) != 1 {
				t.Fatalf("got %d beans, want 1", len(all))
			}
			b := all[0]
			if !strings.HasPrefix(b.ID, tt.wantPrefix) || len(b.ID) != len(tt.wantPrefix)+4 {
				t.Errorf("ID = %q, want prefix %q and 4 random chars", b.ID, tt.wantPrefix)
			}
			if b.Body != tt.wantBody {
				t.Errorf("Body = %q, want %q", b.Body, tt.wantBody)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
