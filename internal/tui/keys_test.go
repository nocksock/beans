package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
)

// TestMain isolates tests from the developer's own user config.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "beans-tui-test")
	if err != nil {
		panic(err)
	}
	os.Setenv("XDG_CONFIG_HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// resetKeymap restores the default key map after the test.
func resetKeymap(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { keymap = DefaultKeyMap() })
}

func runes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestDefaultKeyMapHasNoConflicts(t *testing.T) {
	_, warnings, err := NewKeyMap(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) > 0 {
		t.Errorf("default key map has conflicts:\n%s", strings.Join(warnings, "\n"))
	}
}

func TestKeyMapShape(t *testing.T) {
	// Every context must be a struct of key.Bindings, so config names resolve
	root := reflect.TypeOf(KeyMap{})
	bindingType := reflect.TypeOf(key.Binding{})
	for i := 0; i < root.NumField(); i++ {
		ctx := root.Field(i)
		if ctx.Type.Kind() != reflect.Struct {
			t.Fatalf("%s is not a struct", ctx.Name)
		}
		for j := 0; j < ctx.Type.NumField(); j++ {
			if f := ctx.Type.Field(j); f.Type != bindingType {
				t.Errorf("%s.%s is %s, want key.Binding", ctx.Name, f.Name, f.Type)
			}
		}
	}
}

func TestNewKeyMap(t *testing.T) {
	tests := []struct {
		name      string
		overrides config.Keybindings
		check     func(t *testing.T, km KeyMap)
		wantErr   string
		wantWarns []string
	}{
		{
			name:      "override replaces keys",
			overrides: config.Keybindings{"list": {"changeStatus": {"x", "ctrl+s"}}},
			check: func(t *testing.T, km KeyMap) {
				if got := km.List.ChangeStatus.Keys(); !slices.Equal(got, []string{"x", "ctrl+s"}) {
					t.Errorf("keys = %v", got)
				}
				if got := km.List.ChangeStatus.Help(); got.Key != "x/ctrl+s" || got.Desc != "status" {
					t.Errorf("help = %+v", got)
				}
				if got := km.Detail.ChangeStatus.Keys(); !slices.Equal(got, []string{"s"}) {
					t.Errorf("detail keys = %v, want untouched", got)
				}
			},
		},
		{
			name:      "empty list unbinds",
			overrides: config.Keybindings{"global": {"quit": {}}},
			check: func(t *testing.T, km KeyMap) {
				if len(km.Global.Quit.Keys()) != 0 || key.Matches(runes("q"), km.Global.Quit) {
					t.Errorf("quit should be unbound, keys = %v", km.Global.Quit.Keys())
				}
			},
		},
		{
			name:      "space is the space bar",
			overrides: config.Keybindings{"list": {"toggleSelect": {"space"}, "filterByTag": {"g space"}}},
			check: func(t *testing.T, km KeyMap) {
				if !key.Matches(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, km.List.ToggleSelect) {
					t.Error("space bar should match")
				}
				if got := km.List.FilterByTag.Keys(); !slices.Equal(got, []string{"g  "}) {
					t.Errorf("chord keys = %q", got)
				}
			},
		},
		{
			name:      "unknown context",
			overrides: config.Keybindings{"lists": {"create": {"c"}}},
			wantErr:   `unknown keybinding context "lists"`,
		},
		{
			name:      "unknown action",
			overrides: config.Keybindings{"list": {"ChangeStatus": {"c"}}},
			wantErr:   `unknown keybinding action "ChangeStatus" in "list"`,
		},
		{
			name:      "conflict within a context",
			overrides: config.Keybindings{"list": {"create": {"s"}}},
			wantWarns: []string{`list: "s" is bound to create, changeStatus`},
		},
		{
			name:      "conflict with a global binding",
			overrides: config.Keybindings{"detail": {"edit": {"q"}}},
			wantWarns: []string{`detail: "q" is bound to global.quit, edit`},
		},
		{
			name:      "same key in different contexts is fine",
			overrides: config.Keybindings{"picker": {"confirm": {"s"}}},
		},
		{
			name:      "key shadowing a chord",
			overrides: config.Keybindings{"list": {"create": {"g"}}},
			wantWarns: []string{`list: "g" (create) shadows chord "g t"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			km, warnings, err := NewKeyMap(tt.overrides)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(warnings, tt.wantWarns) {
				t.Errorf("warnings = %q, want %q", warnings, tt.wantWarns)
			}
			if tt.check != nil {
				tt.check(t, km)
			}
		})
	}
}

func TestKeyLabel(t *testing.T) {
	tests := []struct {
		keys []string
		want string
	}{
		{[]string{"esc", "backspace"}, "esc/bksp"},
		{[]string{" "}, "space"},
		{[]string{"up", "k"}, "↑/k"},
		{[]string{"g t"}, "g t"},
		{nil, ""},
	}
	for _, tt := range tests {
		if got := keyLabel(tt.keys); got != tt.want {
			t.Errorf("keyLabel(%q) = %q, want %q", tt.keys, got, tt.want)
		}
	}
}

// newKeymapTestApp returns a test app using the given key binding overrides.
func newKeymapTestApp(t *testing.T, overrides config.Keybindings) *App {
	t.Helper()
	resetKeymap(t)
	app, _ := newTestApp(t)
	app.config.Keybinding = overrides
	if _, err := applyKeymap(app.config); err != nil {
		t.Fatal(err)
	}
	app.list.list.KeyMap = mainListKeyMap()
	return app
}

func TestReboundKeys(t *testing.T) {
	tests := []struct {
		name      string
		overrides config.Keybindings
		keys      []tea.KeyMsg
		wantState viewState
		wantQuit  bool
	}{
		{
			name:      "rebound key opens the status picker",
			overrides: config.Keybindings{"list": {"changeStatus": {"x"}}},
			keys:      []tea.KeyMsg{runes("x")},
			wantState: viewStatusPicker,
		},
		{
			name:      "old key no longer works",
			overrides: config.Keybindings{"list": {"changeStatus": {"x"}}},
			keys:      []tea.KeyMsg{runes("s")},
			wantState: viewList,
		},
		{
			name:      "unbound action does nothing",
			overrides: config.Keybindings{"list": {"changeType": {}}},
			keys:      []tea.KeyMsg{runes("t")},
			wantState: viewList,
		},
		{
			name:      "rebound quit",
			overrides: config.Keybindings{"global": {"quit": {"ctrl+q"}}},
			keys:      []tea.KeyMsg{{Type: tea.KeyCtrlQ}},
			wantQuit:  true,
		},
		{
			name:      "old quit key no longer quits",
			overrides: config.Keybindings{"global": {"quit": {"ctrl+q"}}},
			keys:      []tea.KeyMsg{runes("q")},
			wantState: viewList,
		},
		{
			name:      "rebound chord",
			overrides: config.Keybindings{"list": {"changePriority": {"g p"}}},
			keys:      []tea.KeyMsg{runes("g"), runes("p")},
			wantState: viewPriorityPicker,
		},
		{
			name:      "chord prefix alone does nothing",
			overrides: config.Keybindings{"list": {"changePriority": {"g p"}}},
			keys:      []tea.KeyMsg{runes("g")},
			wantState: viewList,
		},
		{
			name:      "unknown chord is swallowed",
			overrides: config.Keybindings{"list": {"changePriority": {"g p"}}},
			keys:      []tea.KeyMsg{runes("g"), runes("s")},
			wantState: viewList,
		},
		{
			name:      "rebound picker cancel",
			overrides: config.Keybindings{"list": {"changeStatus": {"x"}}, "picker": {"cancel": {"x"}}},
			keys:      []tea.KeyMsg{runes("x"), runes("x")},
			wantState: viewList,
		},
		{
			name:      "rebound edit opens the edit form",
			overrides: config.Keybindings{"list": {"edit": {"ctrl+o"}}},
			keys:      []tea.KeyMsg{{Type: tea.KeyCtrlO}},
			wantState: viewCreateModal,
		},
		{
			name:     "back at the top level quits",
			keys:     []tea.KeyMsg{{Type: tea.KeyEsc}},
			wantQuit: true,
		},
		{
			name:      "rebound help",
			overrides: config.Keybindings{"global": {"help": {"H"}}},
			keys:      []tea.KeyMsg{runes("H")},
			wantState: viewHelpOverlay,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := newKeymapTestApp(t, tt.overrides)
			var quit bool
			for _, k := range tt.keys {
				_, cmd := app.Update(k)
				if isQuit(cmd) {
					quit = true
					break
				}
				// One level is enough to open/close views; deeper would run
				// text input blink timers
				if cmd != nil {
					if msg := cmd(); msg != nil {
						app.Update(msg)
					}
				}
			}
			if quit != tt.wantQuit {
				t.Fatalf("quit = %v, want %v", quit, tt.wantQuit)
			}
			if !tt.wantQuit && app.state != tt.wantState {
				t.Errorf("state = %v, want %v", app.state, tt.wantState)
			}
		})
	}
}

func TestDefaultFilterByTagChord(t *testing.T) {
	resetKeymap(t)
	app, b := newTestApp(t)
	b.Tags = []string{"x"}
	if err := app.core.Update(b); err != nil {
		t.Fatal(err)
	}
	app.Update(app.list.loadBeans())

	app.Update(runes("g"))
	_, cmd := app.Update(runes("t"))
	runCmd(app, cmd, 0)
	if app.state != viewTagPicker {
		t.Errorf("state = %v, want tag picker", app.state)
	}
}

func TestFooterShowsReboundKeys(t *testing.T) {
	app := newKeymapTestApp(t, config.Keybindings{
		"list":   {"changeStatus": {"x"}, "archive": {}},
		"global": {"quit": {"ctrl+q"}},
	})
	footer := app.list.Footer()
	for _, want := range []string{"x", "status", "ctrl+q", "quit"} {
		if !strings.Contains(footer, want) {
			t.Errorf("footer should contain %q:\n%s", want, footer)
		}
	}
	if strings.Contains(footer, "Y archive") {
		t.Errorf("footer should not show the unbound archive action:\n%s", footer)
	}
}

func TestReloadAppliesKeybindings(t *testing.T) {
	resetKeymap(t)
	app, _ := newTestApp(t)
	path := filepath.Join(app.config.ConfigDir(), config.ConfigFileName)

	write := func(content string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		app.Update(reloadConfigMsg{})
	}

	write("keybinding:\n  list:\n    changeStatus: x\n")
	if !key.Matches(runes("x"), keymap.List.ChangeStatus) {
		t.Fatal("reload should apply the new binding")
	}
	if !strings.HasPrefix(app.list.notice, "Config reloaded") || app.list.noticeIsErr {
		t.Errorf("notice = %q", app.list.notice)
	}

	write("keybinding:\n  list:\n    nope: x\n")
	if !key.Matches(runes("x"), keymap.List.ChangeStatus) {
		t.Error("an invalid keybinding should keep the current key map")
	}
	if !strings.Contains(app.list.notice, "Config not reloaded") || !app.list.noticeIsErr {
		t.Errorf("notice = %q", app.list.notice)
	}
}

func TestRunRejectsInvalidKeybinding(t *testing.T) {
	resetKeymap(t)
	dir := t.TempDir()
	cfg := config.Default()
	cfg.SetConfigDir(dir)
	cfg.Keybinding = config.Keybindings{"list": {"nope": {"x"}}}
	core := beancore.New(dir, cfg)
	if err := Run(core, cfg); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("Run() error = %v, want unknown action error", err)
	}
}

func TestReboundToggleArchived(t *testing.T) {
	app := newKeymapTestApp(t, config.Keybindings{"list": {"toggleArchived": {"ctrl+a"}}})
	app.Update(runes("A"))
	if app.list.showArchived {
		t.Error("the default key should no longer toggle archived beans")
	}
	app.Update(tea.KeyMsg{Type: tea.KeyCtrlA})
	if !app.list.showArchived {
		t.Error("the rebound key should toggle archived beans")
	}
}

func TestReboundToggleLog(t *testing.T) {
	app := newKeymapTestApp(t, config.Keybindings{"list": {"toggleLog": {"ctrl+l"}}})
	app.Update(runes("L"))
	if app.log.visible {
		t.Error("the default key should no longer toggle the log panel")
	}
	app.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	if !app.log.visible {
		t.Error("the rebound key should toggle the log panel")
	}
}
