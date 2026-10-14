package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestLauncherResultView(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "launch.log")
	long := strings.Repeat("x", 200)
	if err := os.WriteFile(logPath, []byte("first\nboom\n"+long+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		success  bool
		exitCode int
		want     []string
		notWant  []string
	}{
		{
			name:    "success hides log",
			success: true,
			want:    []string{"claude completed successfully"},
			notWant: []string{"boom"},
		},
		{
			name:     "failure shows exit code and log tail",
			exitCode: 3,
			want:     []string{"claude failed (exit 3)", "boom", "e: open in editor"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newLauncherResultModel("claude", tt.success, tt.exitCode, logPath, 60, 30)
			view := m.View()
			for _, s := range tt.want {
				if !strings.Contains(view, s) {
					t.Errorf("view missing %q:\n%s", s, view)
				}
			}
			for _, s := range tt.notWant {
				if strings.Contains(view, s) {
					t.Errorf("view should not contain %q:\n%s", s, view)
				}
			}
			for _, line := range strings.Split(view, "\n") {
				if w := lipgloss.Width(line); w > 60 {
					t.Errorf("line exceeds terminal width (%d > 60): %q", w, line)
				}
			}
		})
	}
}

func TestLauncherResultUpdate(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "launch.log")
	if err := os.WriteFile(logPath, []byte("x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	m := newLauncherResultModel("claude", false, 1, logPath, 0, 0)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if m.width != 80 || m.height != 24 {
		t.Errorf("size = %dx%d, want 80x24", m.width, m.height)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected close command")
	}
	if _, ok := cmd().(closeLauncherResultMsg); !ok {
		t.Error("expected closeLauncherResultMsg")
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Error("log file should be removed on close")
	}
}
