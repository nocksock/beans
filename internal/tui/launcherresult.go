package tui

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/ui"
)

type launcherResultModel struct {
	name     string
	success  bool
	exitCode int
	logPath  string
	width    int
	height   int
}

type launcherResultMsg struct {
	name     string
	success  bool
	exitCode int
	logPath  string
}

type closeLauncherResultMsg struct{}

func newLauncherResultModel(name string, success bool, exitCode int, logPath string, width, height int) launcherResultModel {
	return launcherResultModel{
		name:     name,
		success:  success,
		exitCode: exitCode,
		logPath:  logPath,
		width:    width,
		height:   height,
	}
}

func (m *launcherResultModel) Init() tea.Cmd {
	return nil
}

func (m *launcherResultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc", "q":
			// Clean up temp log file
			if m.logPath != "" {
				os.Remove(m.logPath)
			}
			return m, func() tea.Msg { return closeLauncherResultMsg{} }
		case "e":
			// Open log in editor
			if m.logPath != "" {
				editor := getEditor()
				cmd := exec.Command(editor, m.logPath)
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return nil })
			}
		}
	}
	return m, nil
}

func (m *launcherResultModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true)
	if m.success {
		headerStyle = headerStyle.Foreground(lipgloss.Color("10"))
		content.WriteString(headerStyle.Render("✓ ") + m.name + " completed successfully\n\n")
		content.WriteString(ui.Muted.Render("Press Enter or Esc to continue"))
	} else {
		headerStyle = headerStyle.Foreground(lipgloss.Color("9"))
		content.WriteString(headerStyle.Render("✗ ") + m.name + " failed (exit " + strconv.Itoa(m.exitCode) + ")\n\n")

		if m.logPath != "" {
			output := readLastLines(m.logPath, 10)
			content.WriteString(ui.Muted.Render("--- Last 10 lines of stderr ---") + "\n")
			content.WriteString(output)
			content.WriteString("\n\n")
		}

		content.WriteString(ui.Muted.Render("e: open in editor    Enter/Esc: close"))
	}

	// Center the content
	lines := strings.Split(content.String(), "\n")
	return centerContent(lines, m.width, m.height)
}

func readLastLines(path string, n int) string {
	file, err := os.Open(path)
	if err != nil {
		return "  (could not read log file)"
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) == 0 {
		return "  (empty output)"
	}

	// Get last n lines
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}

	var result strings.Builder
	for i := start; i < len(lines); i++ {
		result.WriteString("  " + lines[i] + "\n")
	}
	return result.String()
}

func centerContent(lines []string, width, height int) string {
	if len(lines) == 0 {
		return ""
	}

	maxWidth := 0
	for _, line := range lines {
		if lipgloss.Width(line) > maxWidth {
			maxWidth = lipgloss.Width(line)
		}
	}

	// Box is content plus "│ " and " │"; keep it within the terminal
	contentWidth := maxWidth
	if contentWidth > width-6 {
		contentWidth = max(width-6, 1)
	}

	// Center vertically by adding newlines
	vertPadding := (height - len(lines) - 2) / 2
	if vertPadding < 0 {
		vertPadding = 0
	}

	var result strings.Builder
	result.WriteString(strings.Repeat("\n", vertPadding))

	// Top border
	result.WriteString(ui.Muted.Render("┌"+strings.Repeat("─", contentWidth+2)+"┐") + "\n")

	// Content
	for _, line := range lines {
		line = ansi.Truncate(line, contentWidth, "…")
		padding := contentWidth - lipgloss.Width(line)
		result.WriteString(ui.Muted.Render("│ "))
		result.WriteString(line)
		result.WriteString(strings.Repeat(" ", padding))
		result.WriteString(ui.Muted.Render(" │") + "\n")
	}

	// Bottom border
	result.WriteString(ui.Muted.Render("└" + strings.Repeat("─", contentWidth+2) + "┘"))

	return result.String()
}
