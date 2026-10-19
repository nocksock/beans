package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var ctrlG = tea.KeyMsg{Type: tea.KeyCtrlG}

// focusSelector returns a test form focused on the given selector, in
// navigation mode, with a clock the test controls
func focusSelector(f createField, clock *time.Time) createModalModel {
	m := newTestCreateModal()
	m.prefixes = []string{"beans", "other"}
	m.now = func() time.Time { return *clock }
	m.setFocus(f)
	return m
}

func TestCreateModalTypeAhead(t *testing.T) {
	tests := []struct {
		name  string
		field createField
		keys  string
		pause bool // let the type-ahead buffer time out before the last key
		want  string
	}{
		{"a letter selects the first match", createFieldType, "f", false, "feature"},
		{"matching ignores case", createFieldType, "F", false, "feature"},
		{"letters build a prefix", createFieldStatus, "dr", false, "draft"},
		{"repeating a letter moves to the next match", createFieldSize, "xx", false, "xl"},
		{"repeating wraps around", createFieldSize, "xxx", false, "xs"},
		{"a non-matching key is ignored", createFieldType, "fz", false, "feature"},
		{"the buffer resets after a pause", createFieldType, "fe", true, "epic"},
		{"letters bound in navigation mode type ahead after i", createFieldType, "ie", false, "epic"},
		{"works on the prefix", createFieldPrefix, "o", false, "other"},
		{"works on the size", createFieldSize, "x", false, "xs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := time.Now()
			m := focusSelector(tt.field, &clock)
			m, _ = pressCreate(m, ctrlG) // not a chord on its own: ignored
			for i, r := range tt.keys {
				if tt.pause && i == len(tt.keys)-1 {
					clock = clock.Add(2 * typeAheadTimeout)
				}
				m, _ = pressCreate(m, keyMsg(string(r)))
			}
			if got := m.selectedName(tt.field); got != tt.want {
				t.Errorf("selected %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateModalSelectorModes(t *testing.T) {
	clock := time.Now()
	m := focusSelector(createFieldType, &clock)
	if m.typeAhead {
		t.Fatal("a tabbed-to selector should start in navigation mode")
	}

	m, _ = pressCreate(m, keyMsg("i"))
	if !m.typeAhead {
		t.Fatal("i should enter type-ahead mode")
	}
	m, _ = pressCreate(m, keyMsg("l"))
	if m.focus != createFieldType {
		t.Errorf("l in type-ahead mode moved focus to %v", m.focus)
	}

	m, cmd := pressCreate(m, keyMsg("esc"))
	if m.typeAhead || cmd != nil {
		t.Fatalf("first esc should only leave type-ahead mode (typeAhead=%v, cmd=%v)", m.typeAhead, cmd != nil)
	}
	_, cmd = pressCreate(m, keyMsg("esc"))
	if cmd == nil {
		t.Fatal("second esc should cancel the form")
	}
	if _, ok := cmd().(closeCreateModalMsg); !ok {
		t.Error("second esc should close the form")
	}

	m, _ = pressCreate(m, keyMsg("f"))
	if !m.typeAhead || m.selectedName(createFieldType) != "feature" {
		t.Errorf("an unbound letter should enter type-ahead and select (typeAhead=%v, type=%q)", m.typeAhead, m.selectedName(createFieldType))
	}
	m, _ = pressCreate(m, keyMsg("tab"))
	if m.typeAhead {
		t.Error("moving focus should leave type-ahead mode")
	}
}

func TestCreateModalNavigation(t *testing.T) {
	tests := []struct {
		name     string
		prefixes []string
		from     createField
		keys     []string
		want     createField
	}{
		{"l moves right", []string{"beans", "other"}, createFieldType, []string{"l"}, createFieldStatus},
		{"h moves left", []string{"beans", "other"}, createFieldType, []string{"h"}, createFieldPrefix},
		{"h skips a single prefix", []string{"beans"}, createFieldType, []string{"h"}, createFieldType},
		{"l stops at the last column", []string{"beans"}, createFieldSize, []string{"l"}, createFieldSize},
		{"k moves up to the description", []string{"beans"}, createFieldStatus, []string{"k"}, createFieldBody},
		{"j stays on the selector row", []string{"beans"}, createFieldStatus, []string{"j"}, createFieldStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock := time.Now()
			m := focusSelector(tt.from, &clock)
			m.prefixes = tt.prefixes
			for _, k := range tt.keys {
				m, _ = pressCreate(m, keyMsg(k))
			}
			if m.focus != tt.want {
				t.Errorf("focus = %v, want %v", m.focus, tt.want)
			}
		})
	}
}

func TestCreateModalArrowsCycle(t *testing.T) {
	clock := time.Now()
	m := focusSelector(createFieldType, &clock)
	before := m.selectedName(createFieldType)
	m, _ = pressCreate(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.selectedName(createFieldType) == before {
		t.Error("right should change the value")
	}
}

func TestCreateModalJumpChords(t *testing.T) {
	tests := []struct {
		key       string
		prefixes  []string
		want      createField
		typeAhead bool
	}{
		{"n", []string{"beans"}, createFieldTitle, false},
		{"d", []string{"beans"}, createFieldBody, false},
		{"i", []string{"beans", "other"}, createFieldPrefix, true},
		{"i", []string{"beans"}, createFieldBody, false}, // hidden prefix: focus stays
		{"t", []string{"beans"}, createFieldType, true},
		{"s", []string{"beans"}, createFieldStatus, true},
		{"P", []string{"beans"}, createFieldPriority, true},
		{"S", []string{"beans"}, createFieldSize, true},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			app, _ := newTestApp(t)
			app.Update(openCreateModalMsg{})
			app.createModal.prefixes = tt.prefixes
			app.Update(keyMsg("tab")) // start in the description
			app.Update(ctrlG)
			app.Update(keyMsg(tt.key))
			m := app.createModal
			if m.focus != tt.want || m.typeAhead != tt.typeAhead {
				t.Errorf("focus = %v (typeAhead %v), want %v (typeAhead %v)", m.focus, m.typeAhead, tt.want, tt.typeAhead)
			}
			if got := m.body.Value(); got != "" {
				t.Errorf("chord leaked into the description: %q", got)
			}
		})
	}
}

func TestCreateModalChordsWhileTypingTitle(t *testing.T) {
	app, _ := newTestApp(t)
	app.Update(openCreateModalMsg{})
	for _, k := range []tea.KeyMsg{keyMsg("t"), ctrlG, keyMsg("x"), keyMsg("s"), ctrlG, keyMsg("t"), keyMsg("f")} {
		app.Update(k)
	}
	m := app.createModal
	if got := m.textInput.Value(); got != "ts" {
		t.Errorf("title = %q, want %q (letters typed, unknown chord dropped)", got, "ts")
	}
	if m.focus != createFieldType || m.selectedName(createFieldType) != "feature" {
		t.Errorf("focus = %v, type = %q; want type/feature", m.focus, m.selectedName(createFieldType))
	}
}

func TestCreateModalDefaultKeysDontConflict(t *testing.T) {
	for _, w := range DefaultKeyMap().conflicts() {
		if strings.HasPrefix(w, "createModal:") {
			t.Error(w)
		}
	}
}
