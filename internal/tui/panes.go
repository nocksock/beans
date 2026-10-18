package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/tui/layout"
)

// pane is what the app needs from a model shown in a layout pane.
type pane interface {
	title() string
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

// newPanes returns the panes of a, by layout ID.
func (a *App) newPanes() map[layout.ID]pane {
	return map[layout.ID]pane{
		paneList:     listPane{&a.list},
		paneContext:  contextPane{&a.ctx},
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
func (p listPane) bindings() []key.Binding { return allBindings(keymap.List) }
func (p listPane) typing() bool            { return p.list.FilterState() == list.Filtering }
func (p listPane) selection() *bean.Bean {
	b, _ := p.previewData()
	return b
}

type contextPane struct{ *contextModel }

func (p contextPane) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	*p.contextModel, cmd = p.Update(msg)
	return cmd
}
func (p contextPane) bindings() []key.Binding { return allBindings(keymap.Context) }
func (p contextPane) typing() bool            { return false }

type depGraphPane struct{ *depNavModel }

func (p depGraphPane) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	*p.depNavModel, cmd = p.Update(msg)
	return cmd
}
func (p depGraphPane) bindings() []key.Binding { return allBindings(keymap.DepGraph) }
func (p depGraphPane) typing() bool            { return false }
func (p depGraphPane) selection() *bean.Bean   { return p.selectedBean() }

// logPane shows the log; it can't take focus yet (see unfocusable).
type logPane struct {
	*logPanelModel
	width, height int
}

func (p *logPane) title() string             { return "Log" }
func (p *logPane) setSize(width, height int) { p.width, p.height = width, height }
func (p *logPane) View() string              { return p.logPanelModel.View(p.width, p.height) }
func (p *logPane) footer() string            { return "" }
func (p *logPane) update(tea.Msg) tea.Cmd    { return nil }
func (p *logPane) bindings() []key.Binding   { return nil }
func (p *logPane) typing() bool              { return false }
