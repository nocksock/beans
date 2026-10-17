package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigGet(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		ConfigFileName: `beans:
  prefix: app-
launchers:
  - name: claude
    exec: claude "$BEANS_ID"
keybinding:
  list:
    changeStatus: s
`,
	})
	cfg, err := LoadFromDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		key     string
		want    any
		wantErr bool
	}{
		{key: "beans.prefix", want: "app-"},
		{key: "beans.id_length", want: 4},           // default
		{key: "beans.default_status", want: "todo"}, // default
		{key: "tui.disable_launcher_warning", want: false},
		{key: "prompt.extra", want: ""}, // nil pointer
		{key: "launchers.0.exec", want: `claude "$BEANS_ID"`},
		{key: "launchers.claude.name", want: "claude"},
		{key: "keybinding.list.changeStatus", want: KeyList{"s"}},
		{key: "beans", want: cfg.Beans},
		{key: "foo.bar", wantErr: true},
		{key: "beans.nope", wantErr: true},
		{key: "beans.prefix.deeper", wantErr: true},
		{key: "launchers.5", wantErr: true},
		{key: "launchers.missing", wantErr: true},
		{key: "keybinding.detail", wantErr: true},
		{key: "beans..prefix", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := cfg.Get(tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Get(%q) = %v, want error", tt.key, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get(%q) error = %v", tt.key, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get(%q) = %#v, want %#v", tt.key, got, tt.want)
			}
		})
	}

	t.Run("empty key returns whole config", func(t *testing.T) {
		got, err := cfg.Get("")
		if err != nil {
			t.Fatal(err)
		}
		if c, ok := got.(Config); !ok || c.Beans.Prefix != "app-" {
			t.Errorf("Get(\"\") = %#v", got)
		}
	})
}

func TestSetSetting(t *testing.T) {
	tests := []struct {
		name    string
		initial string
		key     string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:    "replaces value and keeps comments",
			initial: "# project config\nbeans:\n  # the id prefix\n  prefix: old- # trailing\n  id_length: 4\n",
			key:     "beans.prefix",
			value:   "new-",
			want:    "# project config\nbeans:\n  # the id prefix\n  prefix: new- # trailing\n  id_length: 4\n",
		},
		{
			name:    "parses numbers",
			initial: "beans:\n  prefix: x-\n",
			key:     "beans.id_length",
			value:   "6",
			want:    "beans:\n  prefix: x-\n  id_length: 6\n",
		},
		{
			name:    "creates missing sections",
			initial: "beans:\n  prefix: x-\n",
			key:     "tui.disable_launcher_warning",
			value:   "true",
			want:    "beans:\n  prefix: x-\ntui:\n  disable_launcher_warning: true\n",
		},
		{
			name:    "fills empty section",
			initial: "tui:\n",
			key:     "tui.disable_launcher_warning",
			value:   "true",
			want:    "tui:\n  disable_launcher_warning: true\n",
		},
		{
			name:    "empty file",
			initial: "",
			key:     "beans.prefix",
			value:   "x-",
			want:    "beans:\n  prefix: x-\n",
		},
		{
			name:    "map keys below free-form maps",
			initial: "beans:\n  prefix: x-\n",
			key:     "keybinding.list.changeStatus",
			value:   "[s, S]",
			want:    "beans:\n  prefix: x-\nkeybinding:\n  list:\n    changeStatus: [s, S]\n",
		},
		{
			name:    "cascade can be enabled",
			initial: "beans:\n  prefix: x-\n",
			key:     "cascade",
			value:   "true",
			want:    "beans:\n  prefix: x-\ncascade: true\n",
		},
		{
			name:    "unknown setting",
			initial: "beans:\n  prefix: x-\n",
			key:     "foo.bar",
			value:   "1",
			wantErr: true,
		},
		{
			name:    "wrong type",
			initial: "beans:\n  prefix: x-\n",
			key:     "beans.id_length",
			value:   "lots",
			wantErr: true,
		},
		{
			name:    "can't descend into scalars",
			initial: "beans:\n  prefix: x-\n",
			key:     "beans.prefix.deeper",
			value:   "1",
			wantErr: true,
		},
		{
			name:    "can't address list items",
			initial: "launchers:\n  - name: a\n    exec: a\n",
			key:     "launchers.a.exec",
			value:   "b",
			wantErr: true,
		},
		{
			name:    "cascading file",
			initial: "cascade: true\nbeans:\n  prefix: x-\n",
			key:     "beans.prefix",
			value:   "y-",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ConfigFileName)
			if err := os.WriteFile(path, []byte(tt.initial), 0644); err != nil {
				t.Fatal(err)
			}

			err := SetSetting(path, tt.key, tt.value)
			got, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				if string(got) != tt.initial {
					t.Errorf("file changed on error:\n%s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SetSetting() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("file =\n%s\nwant\n%s", got, tt.want)
			}
			if _, err := Load(path); err != nil {
				t.Errorf("result doesn't load: %v", err)
			}
		})
	}
}

func TestCheckKeyPath(t *testing.T) {
	tests := []struct {
		key     string
		wantErr string
	}{
		{key: "beans.prefix"},
		{key: "cascade"},
		{key: "prompt.extra"},
		{key: "keybinding.anything.goes"},
		{key: "foo.bar", wantErr: "foo: unknown setting"},
		{key: "beans.nope", wantErr: "beans.nope: unknown setting"},
		{key: "beans.prefix.deeper", wantErr: "beans.prefix: not a nested setting"},
		{key: "launchers.claude", wantErr: "launchers: list items can't be set"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			keys, _ := splitKey(tt.key)
			err := checkKeyPath(keys)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("checkKeyPath(%q) error = %v", tt.key, err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("checkKeyPath(%q) error = %v, want %q", tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestSetSettingInheritingFromCascade(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		ConfigFileName:          "cascade: true\n",
		"pkg/" + ConfigFileName: "beans:\n  prefix: pkg-\n",
	})

	err := SetSetting(filepath.Join(root, "pkg", ConfigFileName), "beans.prefix", "x-")
	if !errors.Is(err, ErrCascading) {
		t.Errorf("error = %v, want ErrCascading", err)
	}
}
