package config

import (
	"path/filepath"
	"slices"
	"testing"
)

// scopeTree is the example monorepo from beans-tbbz.
var scopeTree = map[string]string{
	".beans.yml":                  "cascade: true\nbeans:\n  path: .beans\n  prefix: repo-\n",
	"packages/api/.beans.yml":     "cascade: true\nbeans:\n  prefix: api-\n",
	"packages/api/v2/.beans.yml":  "beans:\n  prefix: api-v2-\n",
	"packages/web/.beans.yml":     "beans:\n  prefix: web-\n",
	"packages/web/src/.keep":      "",
	"packages/shared/.beans.yml":  "beans:\n  id_length: 6\n",
	"packages/tools/.beans.yml":   "beans:\n  prefix: \"\"\n",
	"packages/mobile/.beans.yml":  "beans:\n  prefix: mob-\n  path: .beans\n",
	"vendor/lib/.beans.yml":       "beans:\n  prefix: lib-\n  path: ../../.beans\n",
	"vendor/lib/sub/.beans.yml":   "beans:\n  prefix: libsub-\n",
	"node_modules/x/.beans.yml":   "beans:\n  prefix: nm-\n",
	".hidden/.beans.yml":          "beans:\n  prefix: hidden-\n",
	"standalone/.beans.yml":       "beans:\n  prefix: solo-\n",
	"standalone/inner/.beans.yml": "beans:\n  prefix: inner-\n",
}

func loadScopeTree(t *testing.T, root, dir string) *Config {
	t.Helper()
	cfg, err := LoadFromDirectory(filepath.Join(root, filepath.FromSlash(dir)))
	if err != nil {
		t.Fatalf("LoadFromDirectory(%s) error = %v", dir, err)
	}
	return cfg
}

func TestScope(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, scopeTree)

	tests := []struct {
		dir  string
		want string
	}{
		{".", ""},
		{"packages/api", "api-"},
		{"packages/api/v2", "api-v2-"},
		{"packages/web/src", "web-"},
		{"packages/shared", "repo-"},
		{"packages/tools", ""},
		{"packages/mobile", "mob-"},
		{"vendor/lib", "lib-"},
		{"standalone", "solo-"},
		{"standalone/inner", ""}, // standalone doesn't cascade
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			if got := loadScopeTree(t, root, tt.dir).Scope(); got != tt.want {
				t.Errorf("Scope() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("non-cascading root", func(t *testing.T) {
		plain := t.TempDir()
		writeTree(t, plain, map[string]string{
			".beans.yml":     "beans:\n  prefix: plain-\n",
			"sub/.beans.yml": "beans:\n  prefix: sub-\n",
		})
		for _, dir := range []string{".", "sub"} {
			if got := loadScopeTree(t, plain, dir).Scope(); got != "" {
				t.Errorf("Scope(%s) = %q, want empty", dir, got)
			}
		}
		if got := loadScopeTree(t, plain, ".").AvailablePrefixes(); !slices.Equal(got, []string{"plain-"}) {
			t.Errorf("AvailablePrefixes() = %q, want [plain-]", got)
		}
	})

	t.Run("no config", func(t *testing.T) {
		cfg, err := LoadFromDirectory(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.Scope(); got != "" {
			t.Errorf("Scope() = %q, want empty", got)
		}
	})
}

func TestAvailablePrefixes(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, scopeTree)

	tests := []struct {
		dir  string
		want []string
	}{
		// mobile has its own beans dir; standalone/inner and vendor/lib/sub
		// don't inherit; hidden dirs and node_modules aren't searched
		{".", []string{"repo-", "api-", "api-v2-", "web-", "lib-", "solo-"}},
		{"packages/api", []string{"api-", "api-v2-"}},
		{"packages/api/v2", []string{"api-v2-"}},
		{"packages/web/src", []string{"web-"}},
		{"packages/shared", []string{"repo-"}},
		{"packages/tools", []string{""}},
		{"standalone", []string{"solo-"}},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			got := loadScopeTree(t, root, tt.dir).AvailablePrefixes()
			slices.Sort(got[1:])
			want := slices.Clone(tt.want)
			slices.Sort(want[1:])
			if !slices.Equal(got, want) {
				t.Errorf("AvailablePrefixes() = %q, want %q", got, want)
			}
		})
	}
}
