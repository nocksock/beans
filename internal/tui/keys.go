package tui

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/config"
)

// KeyMap holds every rebindable key binding of the TUI, grouped by context.
//
// The names in the keybinding config mirror the Go names in lowerCamel: the
// context is the KeyMap field (List -> "list"), the action is the binding's
// field in that context (ChangeStatus -> "changeStatus").
//
// A binding's keys use bubbletea's key names ("ctrl+c", "enter", "?"), with
// "space" for the space bar. Keys separated by spaces form a chord ("g t").
type KeyMap struct {
	Global             GlobalKeyMap
	Layout             LayoutKeyMap
	List               ListKeyMap
	Context            ContextKeyMap
	DepGraph           DepGraphKeyMap
	Log                LogKeyMap
	Help               HelpKeyMap
	Picker             PickerKeyMap
	TagEditor          TagEditorKeyMap
	CreateModal        CreateModalKeyMap
	ArchiveConfirm     ArchiveConfirmKeyMap
	ConfigureLaunchers ConfigureLaunchersKeyMap
}

// GlobalKeyMap holds bindings available in every pane.
type GlobalKeyMap struct {
	Quit         key.Binding
	ForceQuit    key.Binding // works everywhere
	Help         key.Binding
	ReloadConfig key.Binding
	OpenLog      key.Binding
}

// LayoutKeyMap holds bindings that arrange the panes, available in every pane.
type LayoutKeyMap struct {
	NextPreset     key.Binding
	PrevPreset     key.Binding
	FocusNext      key.Binding
	FocusPrev      key.Binding
	ToggleContext  key.Binding
	ToggleDepGraph key.Binding
	ToggleLog      key.Binding
	FocusList      key.Binding
	FocusContext   key.Binding
	FocusDepGraph  key.Binding
	FocusLog       key.Binding
}

// ListKeyMap holds bindings for the bean list.
type ListKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	PrevPage  key.Binding
	NextPage  key.Binding
	GoToStart key.Binding
	GoToEnd   key.Binding
	Filter    key.Binding

	Open            key.Binding
	Back            key.Binding
	ToggleSelect    key.Binding
	Collapse        key.Binding
	Expand          key.Binding
	ToggleCollapse  key.Binding
	CollapseAll     key.Binding
	FilterByTag     key.Binding
	Create          key.Binding
	CreateChild     key.Binding
	Edit            key.Binding
	OpenEditor      key.Binding
	ChangeStatus    key.Binding
	ChangeType      key.Binding
	ChangePriority  key.Binding
	ChangeSize      key.Binding
	RaisePriority   key.Binding
	LowerPriority   key.Binding
	EditTags        key.Binding
	ChangeParent    key.Binding
	ManageBlocking  key.Binding
	ManageBlockedBy key.Binding
	Launch          key.Binding
	CopyID          key.Binding
	Archive         key.Binding
	ToggleArchived  key.Binding
	ToggleScope     key.Binding
}

// ContextKeyMap holds bindings for the context pane. Its scroll keys, Back
// and OpenEditor apply to a run's output too, the rest only to a bean.
type ContextKeyMap struct {
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GoToStart    key.Binding
	GoToEnd      key.Binding
	Back         key.Binding

	Edit            key.Binding
	OpenEditor      key.Binding
	ChangeStatus    key.Binding
	ChangeType      key.Binding
	ChangePriority  key.Binding
	ChangeSize      key.Binding
	EditTags        key.Binding
	ChangeParent    key.Binding
	ManageBlocking  key.Binding
	ManageBlockedBy key.Binding
	Launch          key.Binding
	CopyID          key.Binding
	Archive         key.Binding
}

// DepGraphKeyMap holds bindings for the dependency graph pane.
type DepGraphKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	Refocus    key.Binding
	ToggleDone key.Binding
	Back       key.Binding
}

// LogKeyMap holds bindings for the log pane.
type LogKeyMap struct {
	Up         key.Binding
	Down       key.Binding
	GoToStart  key.Binding
	GoToEnd    key.Binding
	Open       key.Binding
	OpenEditor key.Binding
	Back       key.Binding
}

// HelpKeyMap holds bindings for the keybinds dialog. Its filter input gets
// all other keys while it has focus.
type HelpKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	GoToStart key.Binding
	GoToEnd   key.Binding
	Filter    key.Binding
	Run       key.Binding
	Close     key.Binding
}

// PickerKeyMap holds bindings shared by all list pickers (status, type,
// priority, parent, blocking, tag filter, launcher).
type PickerKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	PrevPage  key.Binding
	NextPage  key.Binding
	GoToStart key.Binding
	GoToEnd   key.Binding
	Filter    key.Binding

	Confirm key.Binding
	Cancel  key.Binding
	Toggle  key.Binding // blocking picker only
}

// TagEditorKeyMap holds bindings for the tag editor. Its text input gets all
// other keys, so printable keys bound here can't be typed.
type TagEditorKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Toggle key.Binding
	Apply  key.Binding
	Cancel key.Binding
}

// CreateModalKeyMap holds bindings for the create dialog. Its text fields get
// all other keys, so printable keys bound here (except the selector-only
// bindings) can't be typed. The Focus* chords work while typing only when
// they start with a non-printable key like ctrl+g.
//
// Selectors have a navigation mode and a type-ahead mode, where letters
// select the value starting with them. The navigation-only bindings are
// letters that type ahead in type-ahead mode.
type CreateModalKeyMap struct {
	Cancel             key.Binding // leaves type-ahead mode first
	NextField          key.Binding
	PrevField          key.Binding
	Submit             key.Binding
	SubmitAndEdit      key.Binding
	QuickSubmit        key.Binding // everywhere but the description field
	FocusTitle         key.Binding
	FocusBody          key.Binding
	FocusPrefix        key.Binding // enters type-ahead mode
	FocusType          key.Binding // enters type-ahead mode
	FocusStatus        key.Binding // enters type-ahead mode
	FocusPriority      key.Binding // enters type-ahead mode
	FocusSize          key.Binding // enters type-ahead mode
	CyclePrev          key.Binding // selector fields only
	CycleNext          key.Binding // selector fields only
	NavLeft            key.Binding // selector fields in navigation mode only
	NavRight           key.Binding // selector fields in navigation mode only
	NavUp              key.Binding // selector fields in navigation mode only
	NavDown            key.Binding // selector fields in navigation mode only
	TypeAhead          key.Binding // selector fields in navigation mode only
	QuickSubmitAndEdit key.Binding // selector fields in navigation mode only
}

// ArchiveConfirmKeyMap holds bindings for the archive confirmation.
type ArchiveConfirmKeyMap struct {
	Confirm key.Binding
	Cancel  key.Binding
}

// ConfigureLaunchersKeyMap holds bindings for the configure launchers dialog.
type ConfigureLaunchersKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Toggle  key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

// keymap is the active key map. It is rebuilt from the config at startup and
// on config reload (see applyKeymap).
var keymap = DefaultKeyMap()

func bind(desc string, keys ...string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(keyLabel(keys), desc))
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Global: GlobalKeyMap{
			Quit:         bind("quit", "q"),
			ForceQuit:    bind("force quit", "ctrl+c"),
			Help:         bind("keybinds", "?"),
			ReloadConfig: bind("reload config", "R"),
			OpenLog:      bind("latest log output", "O"),
		},
		Layout: LayoutKeyMap{
			NextPreset:     bind("next layout", "+", "="),
			PrevPreset:     bind("previous layout", "-"),
			FocusNext:      bind("next pane", "tab"),
			FocusPrev:      bind("previous pane", "shift+tab"),
			ToggleContext:  bind("toggle details", "v"),
			ToggleDepGraph: bind("toggle dependencies", "D"),
			ToggleLog:      bind("toggle log", "L"),
			FocusList:      bind("focus beans", "1"),
			FocusContext:   bind("focus details", "2"),
			FocusDepGraph:  bind("focus dependencies", "3"),
			FocusLog:       bind("focus log", "4"),
		},
		List: ListKeyMap{
			Up:        bind("up", "up", "k"),
			Down:      bind("down", "down", "j"),
			PrevPage:  bind("prev page", "left", "pgup", "u"),
			NextPage:  bind("next page", "right", "pgdown", "f", "d"),
			GoToStart: bind("go to start", "home"),
			GoToEnd:   bind("go to end", "end", "G"),
			Filter:    bind("filter: text, field:value, sort:field", "/"),

			Open:            bind("open children / details", "enter"),
			Back:            bind("back up a level", "esc", "backspace"),
			ToggleSelect:    bind("select", " "),
			Collapse:        bind("collapse", "h"),
			Expand:          bind("expand", "l"),
			ToggleCollapse:  bind("toggle collapse", "z"),
			CollapseAll:     bind("collapse all", "Z"),
			FilterByTag:     bind("filter by tag", "g t"),
			Create:          bind("create", "c"),
			CreateChild:     bind("create child", "C"),
			Edit:            bind("edit", "e"),
			OpenEditor:      bind("$EDITOR", "E"),
			ChangeStatus:    bind("status", "s"),
			ChangeType:      bind("type", "t"),
			ChangePriority:  bind("priority", "P"),
			ChangeSize:      bind("size", "S"),
			RaisePriority:   bind("raise priority", "<"),
			LowerPriority:   bind("lower priority", ">"),
			EditTags:        bind("tags", "T"),
			ChangeParent:    bind("parent", "p"),
			ManageBlocking:  bind("blocking", "b"),
			ManageBlockedBy: bind("blocked by", "B"),
			Launch:          bind("launch", "!"),
			CopyID:          bind("copy ID", "y"),
			Archive:         bind("archive completed", "Y"),
			ToggleArchived:  bind("show archived", "A"),
			ToggleScope:     bind("show all scopes", "g s"),
		},
		Context: ContextKeyMap{
			ScrollUp:     bind("scroll up", "up", "k"),
			ScrollDown:   bind("scroll down", "down", "j"),
			PageUp:       bind("page up", "pgup"),
			PageDown:     bind("page down", "pgdown", " ", "f"),
			HalfPageUp:   bind("½ page up", "u", "ctrl+u"),
			HalfPageDown: bind("½ page down", "d", "ctrl+d"),
			GoToStart:    bind("top", "home", "g"),
			GoToEnd:      bind("bottom", "end", "G"),
			Back:         bind("back", "esc", "backspace"),

			Edit:            bind("edit", "e"),
			OpenEditor:      bind("$EDITOR", "E"),
			ChangeStatus:    bind("status", "s"),
			ChangeType:      bind("type", "t"),
			ChangePriority:  bind("priority", "P"),
			ChangeSize:      bind("size", "S"),
			EditTags:        bind("tags", "T"),
			ChangeParent:    bind("parent", "p"),
			ManageBlocking:  bind("blocking", "b"),
			ManageBlockedBy: bind("blocked by", "B"),
			Launch:          bind("launch", "!"),
			CopyID:          bind("copy ID", "y"),
			Archive:         bind("archive", "Y"),
		},
		DepGraph: DepGraphKeyMap{
			Up:         bind("up", "up", "k"),
			Down:       bind("down", "down", "j"),
			Refocus:    bind("root graph here & open", "enter"),
			ToggleDone: bind("show done", "A"),
			Back:       bind("back", "esc", "backspace"),
		},
		Log: LogKeyMap{
			Up:         bind("up", "up", "k"),
			Down:       bind("down", "down", "j"),
			GoToStart:  bind("oldest", "home", "g"),
			GoToEnd:    bind("latest", "end", "G"),
			Open:       bind("show output", "enter"),
			OpenEditor: bind("$EDITOR", "E"),
			Back:       bind("back", "esc", "backspace"),
		},
		Help: HelpKeyMap{
			Up:        bind("up", "up", "k"),
			Down:      bind("down", "down", "j"),
			GoToStart: bind("go to start", "home", "g"),
			GoToEnd:   bind("go to end", "end", "G"),
			Filter:    bind("filter", "/"),
			Run:       bind("run", "enter"),
			Close:     bind("close", "?", "esc"),
		},
		Picker: PickerKeyMap{
			Up:        bind("up", "up", "k"),
			Down:      bind("down", "down", "j"),
			PrevPage:  bind("prev page", "left", "h", "pgup", "b", "u"),
			NextPage:  bind("next page", "right", "l", "pgdown", "f", "d"),
			GoToStart: bind("go to start", "home", "g"),
			GoToEnd:   bind("go to end", "end", "G"),
			Filter:    bind("filter", "/"),

			Confirm: bind("select", "enter"),
			Cancel:  bind("cancel", "esc", "backspace"),
			Toggle:  bind("toggle", " "),
		},
		TagEditor: TagEditorKeyMap{
			Up:     bind("up", "up", "ctrl+p"),
			Down:   bind("down", "down", "ctrl+n"),
			Toggle: bind("toggle", " ", "tab"),
			Apply:  bind("apply", "enter"),
			Cancel: bind("cancel", "esc"),
		},
		CreateModal: CreateModalKeyMap{
			Cancel:             bind("cancel", "esc"),
			NextField:          bind("next", "tab"),
			PrevField:          bind("previous", "shift+tab"),
			Submit:             bind("create", "ctrl+s"),
			SubmitAndEdit:      bind("create & edit", "ctrl+e"),
			QuickSubmit:        bind("create", "enter"),
			FocusTitle:         bind("go to title", "ctrl+g n"),
			FocusBody:          bind("go to description", "ctrl+g d"),
			FocusPrefix:        bind("go to prefix", "ctrl+g i"),
			FocusType:          bind("go to type", "ctrl+g t"),
			FocusStatus:        bind("go to status", "ctrl+g s"),
			FocusPriority:      bind("go to priority", "ctrl+g P"),
			FocusSize:          bind("go to size", "ctrl+g S"),
			CyclePrev:          bind("change", "left", "up", "ctrl+p"),
			CycleNext:          bind("change", "right", "down", "ctrl+n", " "),
			NavLeft:            bind("left", "h"),
			NavRight:           bind("right", "l"),
			NavUp:              bind("up", "k"),
			NavDown:            bind("down", "j"),
			TypeAhead:          bind("type to select", "i"),
			QuickSubmitAndEdit: bind("create & edit", "e"),
		},
		ArchiveConfirm: ArchiveConfirmKeyMap{
			Confirm: bind("archive", "y", "Y"),
			Cancel:  bind("cancel", "n", "N", "esc", "backspace"),
		},
		ConfigureLaunchers: ConfigureLaunchersKeyMap{
			Up:      bind("up", "up", "k"),
			Down:    bind("down", "down", "j"),
			Toggle:  bind("toggle", " ", "enter"),
			Confirm: bind("confirm", "y"),
			Cancel:  bind("cancel", "esc", "q"),
		},
	}
}

// NewKeyMap returns the default key map with the given overrides applied.
// An override replaces all keys of its action. It fails on unknown contexts or
// actions, and warns about keys bound to several actions of one context.
func NewKeyMap(overrides config.Keybindings) (KeyMap, []string, error) {
	km := DefaultKeyMap()
	root := reflect.ValueOf(&km).Elem()

	for _, ctxName := range sortedKeys(overrides) {
		ctx, ok := fieldByConfigName(root, ctxName)
		if !ok {
			return km, nil, fmt.Errorf("unknown keybinding context %q (known: %s)", ctxName, strings.Join(configNames(root), ", "))
		}
		for _, action := range sortedKeys(overrides[ctxName]) {
			field, ok := fieldByConfigName(ctx, action)
			if !ok {
				return km, nil, fmt.Errorf("unknown keybinding action %q in %q (known: %s)", action, ctxName, strings.Join(configNames(ctx), ", "))
			}
			b := field.Addr().Interface().(*key.Binding)
			keys := normalizeKeys(overrides[ctxName][action])
			b.SetKeys(keys...)
			b.SetHelp(keyLabel(keys), b.Help().Desc)
		}
	}

	return km, km.conflicts(), nil
}

// normalizeKeys maps config key names to bubbletea's ("space" -> " ").
func normalizeKeys(keys []string) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		steps := strings.Fields(k)
		for j, s := range steps {
			if s == "space" {
				steps[j] = " "
			}
		}
		out[i] = strings.Join(steps, chordSep)
	}
	return out
}

// conflicts lists keys bound to more than one action within a context (the
// panes include the global and layout bindings), and keys that are also the
// start of a chord there.
func (km KeyMap) conflicts() []string {
	root := reflect.ValueOf(km)
	shared := append(contextBindings(root.FieldByName("Global"), "global."),
		contextBindings(root.FieldByName("Layout"), "layout.")...)

	var warnings []string
	for i := 0; i < root.NumField(); i++ {
		ctxName := configName(root.Type().Field(i).Name)
		bindings := contextBindings(root.Field(i), "")
		if paneContexts[ctxName] {
			bindings = append(append([]namedBinding(nil), shared...), bindings...)
		}

		owners := map[string][]string{}
		var keys []string
		for _, nb := range bindings {
			for _, k := range nb.binding.Keys() {
				if owners[k] == nil {
					keys = append(keys, k)
				}
				owners[k] = append(owners[k], nb.name)
			}
		}
		for _, k := range keys {
			if len(owners[k]) > 1 {
				warnings = append(warnings, fmt.Sprintf("%s: %q is bound to %s", ctxName, keyName(k), strings.Join(owners[k], ", ")))
			}
			for _, other := range keys {
				if strings.HasPrefix(other, k+chordSep) {
					warnings = append(warnings, fmt.Sprintf("%s: %q (%s) shadows chord %q", ctxName, keyName(k), strings.Join(owners[k], ", "), keyName(other)))
				}
			}
		}
	}
	return warnings
}

// paneContexts are the contexts of panes, where the global and layout
// bindings apply too.
var paneContexts = map[string]bool{"list": true, "context": true, "depGraph": true, "log": true}

type namedBinding struct {
	name    string
	binding key.Binding
}

func contextBindings(ctx reflect.Value, prefix string) []namedBinding {
	var out []namedBinding
	for i := 0; i < ctx.NumField(); i++ {
		out = append(out, namedBinding{
			name:    prefix + configName(ctx.Type().Field(i).Name),
			binding: ctx.Field(i).Interface().(key.Binding),
		})
	}
	return out
}

// configName turns a Go field name into its config name (ChangeStatus -> changeStatus).
func configName(goName string) string {
	r := []rune(goName)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func configNames(v reflect.Value) []string {
	names := make([]string, v.NumField())
	for i := range names {
		names[i] = configName(v.Type().Field(i).Name)
	}
	return names
}

func fieldByConfigName(v reflect.Value, name string) (reflect.Value, bool) {
	for i := 0; i < v.NumField(); i++ {
		if configName(v.Type().Field(i).Name) == name {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// applyKeymap makes the key map from cfg's keybinding section active. On error
// the current key map stays active.
func applyKeymap(cfg *config.Config) ([]string, error) {
	km, warnings, err := NewKeyMap(cfg.Keybinding)
	if err != nil {
		return nil, err
	}
	keymap = km
	return warnings, nil
}

// --- Chords ---

// chordSep separates the steps of a chord ("g t").
const chordSep = " "

// chordMsg is the key message for a completed chord. Its String() is the
// chord ("g t"), so key.Matches works on it like on any other key.
func chordMsg(chord string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(chord)}
}

// isChordPrefix reports whether seq starts a chord of any of the bindings.
func isChordPrefix(seq string, bindings []key.Binding) bool {
	for _, b := range bindings {
		for _, k := range b.Keys() {
			if strings.HasPrefix(k, seq+chordSep) {
				return true
			}
		}
	}
	return false
}

// unprintableChords returns the chords of the bindings that start with a
// non-printable key (ctrl+g t), which can be typed while typing text.
func unprintableChords(bindings []key.Binding) []key.Binding {
	var out []key.Binding
	for _, b := range bindings {
		var keys []string
		for _, k := range b.Keys() {
			first, _, isChord := strings.Cut(k, chordSep)
			if isChord && len([]rune(first)) > 1 {
				keys = append(keys, k)
			}
		}
		if len(keys) > 0 {
			out = append(out, key.NewBinding(key.WithKeys(keys...)))
		}
	}
	return out
}

// allBindings returns all bindings of a context struct (e.g. keymap.List).
func allBindings(ctx any) []key.Binding {
	v := reflect.ValueOf(ctx)
	out := make([]key.Binding, v.NumField())
	for i := range out {
		out[i] = v.Field(i).Interface().(key.Binding)
	}
	return out
}

// --- Bubbles components ---

// listKeyMap returns a bubbles list key map using our navigation bindings.
// Quitting is left to the app.
func listKeyMap(up, down, prevPage, nextPage, goToStart, goToEnd, filter key.Binding) list.KeyMap {
	km := list.DefaultKeyMap()
	km.CursorUp = up
	km.CursorDown = down
	km.PrevPage = prevPage
	km.NextPage = nextPage
	km.GoToStart = goToStart
	km.GoToEnd = goToEnd
	km.Filter = filter
	km.Quit = key.NewBinding()
	km.ForceQuit = key.NewBinding()
	return km
}

// pickerListKeyMap returns the bubbles list key map for pickers.
func pickerListKeyMap() list.KeyMap {
	k := keymap.Picker
	return listKeyMap(k.Up, k.Down, k.PrevPage, k.NextPage, k.GoToStart, k.GoToEnd, k.Filter)
}

// --- Help rendering ---

// keyNames shortens key names for display.
var keyNames = map[string]string{
	"backspace": "bksp",
	"up":        "↑",
	"down":      "↓",
	"left":      "←",
	"right":     "→",
}

// keyName renders one key (or chord) for display.
func keyName(k string) string {
	steps := strings.Split(k, chordSep)
	if slices.Contains(steps, "") {
		// The space bar itself (" "), possibly inside a chord
		steps = strings.Fields(k)
		if len(steps) == 0 {
			return "space"
		}
	}
	for i, s := range steps {
		if name, ok := keyNames[s]; ok {
			steps[i] = name
		}
	}
	return strings.Join(steps, " ")
}

// keyLabel renders all keys of a binding for display ("esc/bksp").
func keyLabel(keys []string) string {
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = keyName(k)
	}
	return strings.Join(names, "/")
}

// primaryKey renders a binding's first key, or "" if it has none.
func primaryKey(b key.Binding) string {
	keys := b.Keys()
	if len(keys) == 0 {
		return ""
	}
	return keyName(keys[0])
}

// hint renders a footer hint for b ("s status  "), using desc or, if empty,
// the binding's own description. Unbound actions render as "".
func hint(b key.Binding, desc string) string {
	label := primaryKey(b)
	if label == "" {
		return ""
	}
	if desc == "" {
		desc = b.Help().Desc
	}
	return helpKeyStyle.Render(label) + " " + helpStyle.Render(desc) + "  "
}

// groupHint renders one footer hint for related bindings ("</> prio +/-").
func groupHint(desc string, bindings ...key.Binding) string {
	var labels []string
	for _, b := range bindings {
		if l := primaryKey(b); l != "" {
			labels = append(labels, l)
		}
	}
	if len(labels) == 0 {
		return ""
	}
	return helpKeyStyle.Render(strings.Join(labels, "/")) + " " + helpStyle.Render(desc) + "  "
}

// hints joins footer hints, dropping the trailing separator.
func hints(parts ...string) string {
	return strings.TrimSuffix(strings.Join(parts, ""), "  ")
}
