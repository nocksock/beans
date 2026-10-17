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
	"github.com/charmbracelet/bubbles/viewport"
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
	List               ListKeyMap
	Detail             DetailKeyMap
	Help               HelpKeyMap
	Picker             PickerKeyMap
	TagEditor          TagEditorKeyMap
	CreateModal        CreateModalKeyMap
	ArchiveConfirm     ArchiveConfirmKeyMap
	ConfigureLaunchers ConfigureLaunchersKeyMap
}

// GlobalKeyMap holds bindings available in the list and detail views.
type GlobalKeyMap struct {
	Quit         key.Binding
	ForceQuit    key.Binding // works everywhere
	Help         key.Binding
	ReloadConfig key.Binding
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
	View            key.Binding
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
	RaisePriority   key.Binding
	LowerPriority   key.Binding
	EditTags        key.Binding
	ChangeParent    key.Binding
	ManageBlocking  key.Binding
	ManageBlockedBy key.Binding
	Launch          key.Binding
	Archive         key.Binding
	ToggleArchived  key.Binding
	ToggleLog       key.Binding
	OpenLog         key.Binding
}

// DetailKeyMap holds bindings for the bean detail view.
type DetailKeyMap struct {
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	ScrollLeft   key.Binding
	ScrollRight  key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	FilterLinks  key.Binding

	Back            key.Binding
	PrevBean        key.Binding
	NextBean        key.Binding
	ToggleFocus     key.Binding
	FollowLink      key.Binding
	Edit            key.Binding
	OpenEditor      key.Binding
	ChangeStatus    key.Binding
	ChangeType      key.Binding
	ChangePriority  key.Binding
	EditTags        key.Binding
	ChangeParent    key.Binding
	ManageBlocking  key.Binding
	ManageBlockedBy key.Binding
	Launch          key.Binding
	Archive         key.Binding
}

// HelpKeyMap holds bindings for the help overlay.
type HelpKeyMap struct {
	Close key.Binding
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
// all other keys, so printable keys bound here (except the Cycle* and
// QuickSubmitAndEdit bindings, which only apply on selector fields) can't be
// typed.
type CreateModalKeyMap struct {
	Cancel             key.Binding
	NextField          key.Binding
	PrevField          key.Binding
	Submit             key.Binding
	SubmitAndEdit      key.Binding
	QuickSubmit        key.Binding // everywhere but the description field
	CyclePrev          key.Binding // selector fields only
	CycleNext          key.Binding // selector fields only
	QuickSubmitAndEdit key.Binding // selector fields only
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
			Help:         bind("help", "?"),
			ReloadConfig: bind("reload config", "R"),
		},
		List: ListKeyMap{
			Up:        bind("up", "up", "k"),
			Down:      bind("down", "down", "j"),
			PrevPage:  bind("prev page", "left", "pgup", "u"),
			NextPage:  bind("next page", "right", "pgdown", "f", "d"),
			GoToStart: bind("go to start", "home"),
			GoToEnd:   bind("go to end", "end", "G"),
			Filter:    bind("filter", "/"),

			Open:            bind("open", "enter"),
			View:            bind("view", "v"),
			Back:            bind("back", "esc", "backspace"),
			ToggleSelect:    bind("select", " "),
			Collapse:        bind("collapse", "h"),
			Expand:          bind("expand", "l"),
			ToggleCollapse:  bind("fold", "z"),
			CollapseAll:     bind("collapse all", "Z"),
			FilterByTag:     bind("filter by tag", "g t"),
			Create:          bind("create", "c"),
			CreateChild:     bind("create child", "C"),
			Edit:            bind("edit", "e"),
			OpenEditor:      bind("$EDITOR", "E"),
			ChangeStatus:    bind("status", "s"),
			ChangeType:      bind("type", "t"),
			ChangePriority:  bind("priority", "P"),
			RaisePriority:   bind("raise priority", "<"),
			LowerPriority:   bind("lower priority", ">"),
			EditTags:        bind("tags", "T"),
			ChangeParent:    bind("parent", "p"),
			ManageBlocking:  bind("blocking", "b"),
			ManageBlockedBy: bind("blocked by", "B"),
			Launch:          bind("launch", "!"),
			Archive:         bind("archive", "Y"),
			ToggleArchived:  bind("show archived", "A"),
			ToggleLog:       bind("log", "L"),
			OpenLog:         bind("open log", "O"),
		},
		Detail: DetailKeyMap{
			ScrollUp:     bind("scroll up", "up", "k"),
			ScrollDown:   bind("scroll down", "down", "j"),
			ScrollLeft:   bind("scroll left", "left"),
			ScrollRight:  bind("scroll right", "right"),
			PageUp:       bind("page up", "pgup"),
			PageDown:     bind("page down", "pgdown", " ", "f"),
			HalfPageUp:   bind("½ page up", "u", "ctrl+u"),
			HalfPageDown: bind("½ page down", "d", "ctrl+d"),
			FilterLinks:  bind("filter", "/"),

			Back:            bind("back", "esc", "backspace"),
			PrevBean:        bind("prev bean", "h"),
			NextBean:        bind("next bean", "l"),
			ToggleFocus:     bind("switch", "tab"),
			FollowLink:      bind("go to", "enter"),
			Edit:            bind("edit", "e"),
			OpenEditor:      bind("$EDITOR", "E"),
			ChangeStatus:    bind("status", "s"),
			ChangeType:      bind("type", "t"),
			ChangePriority:  bind("priority", "P"),
			EditTags:        bind("tags", "T"),
			ChangeParent:    bind("parent", "p"),
			ManageBlocking:  bind("blocking", "b"),
			ManageBlockedBy: bind("blocked by", "B"),
			Launch:          bind("launch", "!"),
			Archive:         bind("archive", "Y"),
		},
		Help: HelpKeyMap{
			Close: bind("close", "?", "esc"),
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
			CyclePrev:          bind("change", "left", "h", "up", "k"),
			CycleNext:          bind("change", "right", "l", "down", "j", " "),
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
// list and detail views include the global bindings), and keys that are also
// the start of a chord there.
func (km KeyMap) conflicts() []string {
	root := reflect.ValueOf(km)
	global := root.FieldByName("Global")

	var warnings []string
	for i := 0; i < root.NumField(); i++ {
		ctxName := configName(root.Type().Field(i).Name)
		bindings := contextBindings(root.Field(i), "")
		if ctxName == "list" || ctxName == "detail" {
			bindings = append(contextBindings(global, "global."), bindings...)
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

// detailViewportKeyMap returns the bubbles viewport key map for the detail body.
func detailViewportKeyMap() viewport.KeyMap {
	k := keymap.Detail
	return viewport.KeyMap{
		PageDown:     k.PageDown,
		PageUp:       k.PageUp,
		HalfPageUp:   k.HalfPageUp,
		HalfPageDown: k.HalfPageDown,
		Up:           k.ScrollUp,
		Down:         k.ScrollDown,
		Left:         k.ScrollLeft,
		Right:        k.ScrollRight,
	}
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
