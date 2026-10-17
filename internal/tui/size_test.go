package tui

import (
	"strings"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
)

func TestCreateModalSize(t *testing.T) {
	tests := []struct {
		name string
		keys []string // pressed after typing a title
		want string
	}{
		{"default size is omitted", []string{"enter"}, ""},
		{"cycle to lg", []string{"tab", "tab", "tab", "tab", "l", "enter"}, "lg"},
		{"cycle to xs", []string{"tab", "tab", "tab", "tab", "h", "h", "enter"}, "xs"},
		{"cycle back to md is omitted", []string{"tab", "tab", "tab", "tab", "l", "h", "enter"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := typeIntoCreate(newTestCreateModal(), "Sized")
			for _, k := range tt.keys[:len(tt.keys)-1] {
				m, _ = pressCreate(m, keyMsg(k))
			}
			_, cmd := pressCreate(m, keyMsg(tt.keys[len(tt.keys)-1]))
			got, ok := cmd().(beanCreatedMsg)
			if !ok {
				t.Fatalf("got %#v, want beanCreatedMsg", cmd())
			}
			if got.size != tt.want {
				t.Errorf("size = %q, want %q", got.size, tt.want)
			}
		})
	}
}

func TestEditModalSize(t *testing.T) {
	str := func(p *string) string {
		if p == nil {
			return "<nil>"
		}
		return *p
	}
	tests := []struct {
		name     string
		size     string
		keys     []string
		wantSize string
	}{
		{"unchanged", "lg", nil, "<nil>"},
		{"unset stays unset", "", nil, "<nil>"},
		{"grow", "lg", []string{"tab", "tab", "tab", "tab", "l"}, "xl"},
		{"back to md clears", "lg", []string{"tab", "tab", "tab", "tab", "h"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &bean.Bean{ID: "beans-abcd", Title: "T", Type: "task", Status: "todo", Size: tt.size}
			m := newEditModalModel(config.Default(), 120, 40, b)
			if got := m.sizes[m.sizeIdx].Name; got != config.EffectiveSize(tt.size) {
				t.Errorf("prefilled size = %q, want %q", got, config.EffectiveSize(tt.size))
			}
			for _, k := range tt.keys {
				m, _ = pressCreate(m, keyMsg(k))
			}
			_, cmd := pressCreate(m, keyMsg("enter"))
			if got := str(submitted(t, cmd).input.Size); got != tt.wantSize {
				t.Errorf("input.Size = %s, want %s", got, tt.wantSize)
			}
		})
	}
}

func TestCreateModalViewShowsSize(t *testing.T) {
	for _, width := range []int{60, 120} {
		view := newCreateModalModel(config.Default(), width, 40, "", "").View()
		if !strings.Contains(view, "Size") || !strings.Contains(view, "md") {
			t.Errorf("width %d: view missing size selector:\n%s", width, view)
		}
	}
}

func TestSizePicker(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{"", "md"},
		{"xs", "xs"},
		{"xl", "xl"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			m := newSizePickerModel([]string{"beans-abcd"}, "T", tt.current, 120, 40)
			_, cmd := m.Update(keyMsg("enter"))
			got, ok := cmd().(sizeSelectedMsg)
			if !ok {
				t.Fatalf("got %#v, want sizeSelectedMsg", cmd())
			}
			if got.size != tt.want {
				t.Errorf("selected = %q, want %q", got.size, tt.want)
			}
		})
	}
}

func TestSizePickerUpdatesBean(t *testing.T) {
	app, b := newTestApp(t)
	_, cmd := app.Update(keyMsg("S"))
	if cmd == nil {
		t.Fatal("expected command")
	}
	app.Update(cmd())
	if app.state != viewSizePicker {
		t.Fatalf("state = %v, want size picker", app.state)
	}
	app.Update(sizeSelectedMsg{beanIDs: []string{b.ID}, size: "xl"})
	if app.state != viewList {
		t.Errorf("state = %v, want list", app.state)
	}
	got, err := app.core.Get(b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Size != "xl" {
		t.Errorf("size = %q, want xl", got.Size)
	}
}
