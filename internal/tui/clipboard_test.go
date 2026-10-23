package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/config"
)

// fakeClipboard replaces the system clipboard for the test and records what
// was written to it.
func fakeClipboard(t *testing.T, err error) *[]string {
	t.Helper()
	var written []string
	orig := writeClipboard
	writeClipboard = func(s string) error {
		written = append(written, s)
		return err
	}
	t.Cleanup(func() { writeClipboard = orig })
	return &written
}

func TestCopyIDDefaultBinding(t *testing.T) {
	km := DefaultKeyMap()
	tests := []struct {
		name    string
		binding []string
	}{
		{"list", km.List.CopyID.Keys()},
		{"context", km.Context.CopyID.Keys()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.binding) != 1 || tt.binding[0] != "y" {
				t.Errorf("copyID keys = %v, want [y]", tt.binding)
			}
		})
	}
}

func TestCopyID(t *testing.T) {
	tests := []struct {
		name        string
		fromContext bool // focus the context pane first
		clipErr     error
		wantNotice  string
		wantErr     bool
	}{
		{name: "list", wantNotice: "Copied "},
		{name: "context pane", fromContext: true, wantNotice: "Copied "},
		{name: "clipboard unavailable", clipErr: errors.New("no clipboard utility"), wantNotice: "Could not copy ", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetKeymap(t)
			written := fakeClipboard(t, tt.clipErr)
			app, _ := newTestApp(t)

			var want string
			if tt.fromContext {
				app.Update(keyMsg("tab"))
				if app.ws.focus != paneContext || app.ctx.bean == nil {
					t.Fatal("tab should focus the context pane showing a bean")
				}
				want = app.ctx.bean.ID
			} else {
				want = app.list.list.SelectedItem().(beanItem).bean.ID
			}

			_, cmd := app.Update(runes("y"))
			if cmd == nil {
				t.Fatal("y should request copying the ID")
			}
			msg := cmd()
			if m, ok := msg.(copyIDMsg); !ok || m.beanID != want {
				t.Fatalf("y sent %#v, want copyIDMsg{%q}", msg, want)
			}
			_, cmd = app.Update(msg)
			app.Update(cmd())

			if len(*written) != 1 || (*written)[0] != want {
				t.Errorf("clipboard got %v, want [%s]", *written, want)
			}
			if !strings.HasPrefix(app.list.notice, tt.wantNotice+want) {
				t.Errorf("notice = %q, want %q...", app.list.notice, tt.wantNotice+want)
			}
			if app.list.noticeIsErr != tt.wantErr {
				t.Errorf("noticeIsErr = %v, want %v", app.list.noticeIsErr, tt.wantErr)
			}
		})
	}
}

func TestCopyIDRebound(t *testing.T) {
	written := fakeClipboard(t, nil)
	app := newKeymapTestApp(t, config.Keybindings{"list": {"copyID": {"ctrl+y"}}})

	if _, cmd := app.Update(runes("y")); cmd != nil {
		if _, ok := cmd().(copyIDMsg); ok {
			t.Error("the default key should no longer copy the ID")
		}
	}
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	if cmd == nil {
		t.Fatal("the rebound key should copy the ID")
	}
	_, cmd = app.Update(cmd())
	app.Update(cmd())
	if len(*written) != 1 {
		t.Errorf("clipboard got %v, want one ID", *written)
	}
}
