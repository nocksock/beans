package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/tui/layout"
)

// pane is what the app needs from a model shown in a layout pane.
type pane interface {
	name() string              // the pane's plain name ("Beans"), for the keybinds dialog
	title() string             // the frame title, may include the shown bean
	setSize(width, height int) // the content area, inside the frame
	View() string              // the content at the set size
	footer() string            // hints while focused
	update(msg tea.Msg) tea.Cmd
	bindings() []key.Binding // keys while focused, for chords
	typing() bool            // keys go to a text input (no quitting, no layout keys)
}

// selectionSource is a pane whose selected bean (nil if none) the context
// pane follows while it has focus.
type selectionSource interface {
	selection() *bean.Bean
}

// runSource is a pane whose selected run (nil if none) the context pane
// shows the output of while it has focus.
type runSource interface {
	selectedRun() *logEntry
}

// newPanes returns the panes of a, by layout ID.
func (a *App) newPanes() map[layout.ID]pane {
	return map[layout.ID]pane{
		paneList:     listPane{&a.list},
		paneContext:  &a.ctxPane,
		paneDepGraph: depGraphPane{&a.deps},
		paneLog:      &logPane{logPanelModel: &a.log},
	}
}

type listPane struct{ *listModel }

func (p listPane) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	*p.listModel, cmd = p.Update(msg)
	return cmd
}
func (p listPane) name() string            { return "Beans" }
func (p listPane) bindings() []key.Binding { return allBindings(keymap.List) }
func (p listPane) typing() bool            { return p.editingQuery() }
func (p listPane) selection() *bean.Bean {
	b, _ := p.previewData()
	return b
}

// contextPane shows the bean, or the run output, that the last focused
// selection source picked.
type contextPane struct {
	bean    *contextModel
	run     *logDetailModel
	showRun bool
}

// active is the view the pane shows.
func (p *contextPane) active() pane {
	if p.showRun {
		return p.run
	}
	return beanContextPane{p.bean}
}

func (p *contextPane) name() string               { return p.active().name() }
func (p *contextPane) title() string              { return p.active().title() }
func (p *contextPane) View() string               { return p.active().View() }
func (p *contextPane) footer() string             { return p.active().footer() }
func (p *contextPane) update(msg tea.Msg) tea.Cmd { return p.active().update(msg) }
func (p *contextPane) bindings() []key.Binding    { return p.active().bindings() }
func (p *contextPane) typing() bool               { return false }

// setSize sizes both views, so either is ready to show.
func (p *contextPane) setSize(width, height int) {
	p.bean.setSize(width, height)
	p.run.setSize(width, height)
}

type beanContextPane struct{ *contextModel }

func (p beanContextPane) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	*p.contextModel, cmd = p.Update(msg)
	return cmd
}
func (p beanContextPane) name() string            { return "Details" }
func (p beanContextPane) bindings() []key.Binding { return allBindings(keymap.Context) }
func (p beanContextPane) typing() bool            { return false }

type depGraphPane struct{ *depNavModel }

func (p depGraphPane) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	*p.depNavModel, cmd = p.Update(msg)
	return cmd
}
func (p depGraphPane) name() string            { return "Dependencies" }
func (p depGraphPane) bindings() []key.Binding { return allBindings(keymap.DepGraph) }
func (p depGraphPane) typing() bool            { return false }
func (p depGraphPane) selection() *bean.Bean   { return p.selectedBean() }

// logPane shows the log, with a cursor on the run whose output the context
// pane shows.
type logPane struct {
	*logPanelModel
	width, height int
}

func (p *logPane) name() string              { return "Log" }
func (p *logPane) title() string             { return "Log" }
func (p *logPane) setSize(width, height int) { p.width, p.height = width, height }
func (p *logPane) View() string              { return p.logPanelModel.View(p.width, p.height) }
func (p *logPane) bindings() []key.Binding   { return allBindings(keymap.Log) }
func (p *logPane) typing() bool              { return false }
func (p *logPane) selectedRun() *logEntry    { return p.selectedEntry() }

func (p *logPane) update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return nil
	}
	k := keymap.Log
	switch {
	case key.Matches(keyMsg, k.Up):
		p.moveCursor(-1)
	case key.Matches(keyMsg, k.Down):
		p.moveCursor(1)
	case key.Matches(keyMsg, k.GoToStart):
		p.moveCursor(-len(p.entries))
	case key.Matches(keyMsg, k.GoToEnd):
		p.moveCursor(len(p.entries))
	case key.Matches(keyMsg, k.Open):
		if p.selectedEntry() != nil {
			return func() tea.Msg { return focusContextMsg{} }
		}
	case key.Matches(keyMsg, k.OpenEditor):
		if e := p.selectedEntry(); e != nil && e.logPath != "" {
			path := e.logPath
			return func() tea.Msg { return openLogMsg{path: path} }
		}
	case key.Matches(keyMsg, k.Back):
		return func() tea.Msg { return focusBackMsg{} }
	}
	return nil
}

func (p *logPane) footer() string {
	k := keymap.Log
	return hints(
		groupHint("move", k.Down, k.Up),
		hint(k.Open, ""),
		hint(k.OpenEditor, ""),
		hint(k.Back, ""),
	)
}
