package tui

import (
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func testSections() []keybindSection {
	return []keybindSection{
		{"Global", []key.Binding{bind("quit", "q"), bind("help", "?")}},
		{"Beans", []key.Binding{
			bind("status", "s"),
			bind("filter by tag", "g t"),
			bind("unbound"), // no keys: skipped
			bind("back", "esc", "backspace"),
		}},
	}
}

func TestKeybindRows(t *testing.T) {
	got := keybindRows(testSections())
	want := []keybindRow{
		{section: "Global"},
		{keys: "q", desc: "quit", key: "q"},
		{keys: "?", desc: "help", key: "?"},
		{section: "Beans"},
		{keys: "s", desc: "status", key: "s"},
		{keys: "g t", desc: "filter by tag", key: "g t"},
		{keys: "esc/bksp", desc: "back", key: "esc"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("keybindRows() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestKeybindRowsSkipsEmptySections(t *testing.T) {
	got := keybindRows([]keybindSection{{"Empty", []key.Binding{bind("unbound")}}})
	if len(got) != 0 {
		t.Errorf("keybindRows() = %+v, want no rows", got)
	}
}

func TestFilterRows(t *testing.T) {
	rows := keybindRows(testSections())
	tests := []struct {
		query string
		want  []string // descs and section headers in order
	}{
		{"", []string{"Global", "quit", "help", "Beans", "status", "filter by tag", "back"}},
		{"STAT", []string{"Beans", "status"}},
		{"g t", []string{"Beans", "filter by tag"}},
		{"bksp", []string{"Beans", "back"}},
		{"q", []string{"Global", "quit"}},
		{"nothing here", nil},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			var got []string
			for _, r := range filterRows(rows, tt.query) {
				if r.header() {
					got = append(got, r.section)
				} else {
					got = append(got, r.desc)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterRows(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func newTestKeybinds() keybindsModel {
	return newKeybindsModel(testSections(), 80, 24)
}

func selectedDesc(m keybindsModel) string {
	r, ok := m.selected()
	if !ok {
		return ""
	}
	return r.desc
}

func TestKeybindsCursorSkipsHeaders(t *testing.T) {
	m := newTestKeybinds()
	if got := selectedDesc(m); got != "quit" {
		t.Fatalf("initial selection = %q, want quit", got)
	}
	steps := []struct {
		key  string
		want string
	}{
		{"j", "help"},
		{"j", "status"}, // skips the Beans header
		{"j", "filter by tag"},
		{"j", "back"},
		{"j", "back"}, // clamps at the end
		{"k", "filter by tag"},
		{"k", "status"},
		{"k", "help"}, // skips the header going up
		{"k", "quit"},
		{"k", "quit"}, // clamps at the start
	}
	for _, s := range steps {
		m, _ = m.Update(keyMsg(s.key))
		if got := selectedDesc(m); got != s.want {
			t.Errorf("after %q: selection = %q, want %q", s.key, got, s.want)
		}
	}
}

func runKey(t *testing.T, m keybindsModel) string {
	t.Helper()
	_, cmd := m.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("enter: expected a command")
	}
	msg, ok := cmd().(runKeybindMsg)
	if !ok {
		t.Fatalf("enter: got %T, want runKeybindMsg", cmd())
	}
	return msg.key
}

func TestKeybindsEnterRunsSelected(t *testing.T) {
	m := newTestKeybinds()
	m, _ = m.Update(keyMsg("j"))
	m, _ = m.Update(keyMsg("j"))
	if got := runKey(t, m); got != "s" {
		t.Errorf("run key = %q, want s", got)
	}
}

func TestKeybindsEnterRunsChord(t *testing.T) {
	m := newTestKeybinds()
	for range 3 {
		m, _ = m.Update(keyMsg("j"))
	}
	if got := runKey(t, m); got != "g t" {
		t.Errorf("run key = %q, want \"g t\"", got)
	}
}

func typeKeybindsFilter(m keybindsModel, text string) keybindsModel {
	for _, r := range text {
		m, _ = m.Update(keyMsg(string(r)))
	}
	return m
}

func TestKeybindsFilter(t *testing.T) {
	m := newTestKeybinds()
	m, _ = m.Update(keyMsg("/"))
	if !m.typing() {
		t.Fatal("after /: typing() = false, want true")
	}
	m = typeKeybindsFilter(m, "tag")
	if got := selectedDesc(m); got != "filter by tag" {
		t.Errorf("selection while filtering = %q, want filter by tag", got)
	}
	if got := runKey(t, m); got != "g t" {
		t.Errorf("enter while filtering runs %q, want \"g t\"", got)
	}
}

func TestKeybindsFilterTypesJAndQuestionMark(t *testing.T) {
	m := newTestKeybinds()
	m, _ = m.Update(keyMsg("/"))
	m = typeKeybindsFilter(m, "j?")
	if got := m.filter.Value(); got != "j?" {
		t.Errorf("filter value = %q, want j?", got)
	}
	if _, ok := m.selected(); ok {
		t.Error("expected no selection for a filter matching nothing")
	}
}

func TestKeybindsArrowsMoveWhileFiltering(t *testing.T) {
	m := newTestKeybinds()
	m, _ = m.Update(keyMsg("/"))
	m = typeKeybindsFilter(m, "ta") // status, filter by tag
	m, _ = m.Update(keyMsg("down"))
	if got := selectedDesc(m); got != "filter by tag" {
		t.Errorf("after down while filtering: %q, want filter by tag", got)
	}
}

func TestKeybindsEscClearsFilterThenCloses(t *testing.T) {
	m := newTestKeybinds()
	m, _ = m.Update(keyMsg("/"))
	m = typeKeybindsFilter(m, "tag")
	m, cmd := m.Update(keyMsg("esc"))
	if cmd != nil {
		t.Fatal("esc while filtering should not close")
	}
	if m.typing() || m.filter.Value() != "" {
		t.Errorf("esc should clear the filter and leave the input: typing=%v value=%q", m.typing(), m.filter.Value())
	}
	if got := selectedDesc(m); got != "quit" {
		t.Errorf("selection after clearing = %q, want quit", got)
	}
	_, cmd = m.Update(keyMsg("esc"))
	if cmd == nil {
		t.Fatal("second esc should close")
	}
	if _, ok := cmd().(closeHelpMsg); !ok {
		t.Errorf("second esc: got %T, want closeHelpMsg", cmd())
	}
}

func TestKeybindsViewShowsSectionsAndKeys(t *testing.T) {
	m := newTestKeybinds()
	v := m.View()
	for _, want := range []string{"Global", "Beans", "esc/bksp", "filter by tag", "g t"} {
		if !strings.Contains(v, want) {
			t.Errorf("View() lacks %q:\n%s", want, v)
		}
	}
}

func TestKeybindsScrollsToCursor(t *testing.T) {
	var bindings []key.Binding
	for _, k := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		bindings = append(bindings, bind("action "+k, k))
	}
	m := newKeybindsModel([]keybindSection{{"Many", bindings}}, 80, 12)
	m, _ = m.Update(keyMsg("G"))
	if got := selectedDesc(m); got != "action j" {
		t.Fatalf("after G: %q, want action j", got)
	}
	if !strings.Contains(m.View(), "action j") {
		t.Errorf("View() should show the selected row:\n%s", m.View())
	}
}

var _ tea.Msg = runKeybindMsg{}

func TestAppKeybindsDialogRunsBinding(t *testing.T) {
	a, _ := newTestApp(t)
	a.Update(keyMsg("?"))
	if a.state != viewHelpOverlay {
		t.Fatalf("after ?: state = %v, want viewHelpOverlay", a.state)
	}
	v := a.View()
	for _, want := range []string{"Global", "Layout", "Beans"} {
		if !strings.Contains(v, want) {
			t.Errorf("dialog lacks section %q", want)
		}
	}
	// Filter down to the status binding and run it
	a.Update(keyMsg("/"))
	for _, r := range "status" {
		a.Update(keyMsg(string(r)))
	}
	_, cmd := a.Update(keyMsg("enter"))
	if cmd == nil {
		t.Fatal("enter: expected a command")
	}
	// The run message replays "s", whose command opens the picker
	_, cmd = a.Update(cmd())
	if cmd == nil {
		t.Fatal("replaying s: expected the status picker command")
	}
	a.Update(cmd())
	if a.state != viewStatusPicker {
		t.Errorf("after running the status binding: state = %v, want viewStatusPicker", a.state)
	}
}

func TestAppKeybindsDialogRunsChord(t *testing.T) {
	a, _ := newTestApp(t)
	a.Update(keyMsg("?"))
	_, cmd := a.Update(runKeybindMsgKey(t, a, "filter by tag"))
	if a.state != viewList {
		t.Fatalf("after running a chord: state = %v, want viewList", a.state)
	}
	if cmd == nil {
		t.Fatal("expected the list to react to the replayed chord")
	}
	if _, ok := cmd().(openTagPickerMsg); !ok {
		t.Errorf("replayed chord produced %T, want openTagPickerMsg", cmd())
	}
}

// runKeybindMsgKey returns the run message for the dialog row with desc.
func runKeybindMsgKey(t *testing.T, a *App, desc string) runKeybindMsg {
	t.Helper()
	for _, r := range a.keybinds.rows {
		if r.desc == desc {
			return runKeybindMsg{key: r.key}
		}
	}
	t.Fatalf("no dialog row %q", desc)
	return runKeybindMsg{}
}

func TestAppKeybindsDialogSectionFollowsFocusedPane(t *testing.T) {
	a, _ := newTestApp(t)
	a.Update(keyMsg("4")) // focus the log pane
	a.Update(keyMsg("?"))
	if v := a.View(); !strings.Contains(v, "Log") || !strings.Contains(v, "oldest") {
		t.Errorf("dialog should list the log pane's bindings:\n%s", v)
	}
}

func TestAppQuitKeyTypesInKeybindsFilter(t *testing.T) {
	a, _ := newTestApp(t)
	a.Update(keyMsg("?"))
	a.Update(keyMsg("/"))
	if _, cmd := a.Update(keyMsg("q")); isQuit(cmd) {
		t.Error("q in the keybinds filter should type, not quit")
	}
	if got := a.keybinds.filter.Value(); got != "q" {
		t.Errorf("filter value = %q, want q", got)
	}
}

func manyKeybinds(height int) keybindsModel {
	var bindings []key.Binding
	for _, k := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		bindings = append(bindings, bind("action "+k, k))
	}
	return newKeybindsModel([]keybindSection{{"Many", bindings}}, 80, height)
}

func TestKeybindsCursorNeverUnderEllipsis(t *testing.T) {
	m := manyKeybinds(12)
	for i := 0; i < 9; i++ {
		m, _ = m.Update(keyMsg("j"))
		want := selectedDesc(m)
		if v := m.View(); !strings.Contains(v, want) {
			t.Fatalf("after %d downs the selected row %q is not shown:\n%s", i+1, want, v)
		}
	}
}

func TestKeybindsFitsTheScreen(t *testing.T) {
	m := manyKeybinds(16)
	if h := lipgloss.Height(m.View()); h > 14 {
		t.Errorf("dialog height = %d, want at most 14 on a 16-row screen", h)
	}
}

func TestKeybindsKeyColumnHasGap(t *testing.T) {
	m := newKeybindsModel([]keybindSection{{"S", []key.Binding{bind("next page", "right", "pgdown", "f", "d")}}}, 80, 24)
	if v := ansi.Strip(m.View()); !strings.Contains(v, "→/pgdown/f/d next page") {
		t.Errorf("key label should be followed by a space:\n%s", v)
	}
}
