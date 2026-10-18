package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestRenderBeanRow_NarrowWidth(t *testing.T) {
	// Test that RenderBeanRow doesn't panic with very small MaxTitleWidth values
	// This was a bug where MaxTitleWidth < 4 caused a slice bounds panic

	tests := []struct {
		name          string
		maxTitleWidth int
		title         string
	}{
		{"zero width", 0, "Test Title"},
		{"width 1", 1, "Test Title"},
		{"width 2", 2, "Test Title"},
		{"width 3", 3, "Test Title"},
		{"width 4", 4, "Test Title"},
		{"width 5", 5, "Test Title"},
		{"short title fits", 10, "Hi"},
		{"exact fit", 10, "0123456789"},
		{"needs truncation", 10, "This is a longer title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RenderBeanRow panicked with MaxTitleWidth=%d: %v", tt.maxTitleWidth, r)
				}
			}()

			cfg := BeanRowConfig{
				MaxTitleWidth: tt.maxTitleWidth,
				StatusColor:   "green",
				TypeColor:     "blue",
			}

			result := RenderBeanRow("abc123", "todo", "task", tt.title, cfg)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestRenderBeanRow_NarrowWidthWithPriority(t *testing.T) {
	// Priority symbol takes 2 extra chars, which reduces available title width
	// This tests that the adjustment doesn't cause negative slice bounds

	tests := []struct {
		name          string
		maxTitleWidth int
		priority      string
	}{
		{"width 1 with priority", 1, "high"},
		{"width 2 with priority", 2, "high"},
		{"width 3 with priority", 3, "critical"},
		{"width 4 with priority", 4, "high"},
		{"width 5 with priority", 5, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RenderBeanRow panicked with MaxTitleWidth=%d and priority=%s: %v",
						tt.maxTitleWidth, tt.priority, r)
				}
			}()

			cfg := BeanRowConfig{
				MaxTitleWidth: tt.maxTitleWidth,
				Priority:      tt.priority,
				PriorityColor: "red",
				StatusColor:   "green",
				TypeColor:     "blue",
			}

			result := RenderBeanRow("abc123", "todo", "task", "Long title that needs truncation", cfg)
			if result == "" {
				t.Error("expected non-empty result")
			}
		})
	}
}

func TestShortCode(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"milestone", "M"},
		{"epic", "E"},
		{"bug", "B"},
		{"feature", "F"},
		{"task", "T"},
		{"draft", "D"},
		{"todo", "T"},
		{"in-progress", "I"},
		{"completed", "C"},
		{"scrapped", "S"},
		{"ünicode", "Ü"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ShortCode(tt.input); got != tt.want {
				t.Errorf("ShortCode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCalculateResponsiveColumns_Compact(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		wantCompact bool
		wantStatus  int
		wantType    int
	}{
		{"narrow", 60, true, ColWidthCompact, ColWidthCompact},
		{"just below threshold", CompactColumnsMaxWidth - 1, true, ColWidthCompact, ColWidthCompact},
		{"at threshold", CompactColumnsMaxWidth, false, ColWidthStatus, ColWidthType},
		{"wide", 200, false, ColWidthStatus, ColWidthType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cols := CalculateResponsiveColumns(tt.width, false)
			if cols.Compact != tt.wantCompact || cols.Status != tt.wantStatus || cols.Type != tt.wantType {
				t.Errorf("got compact=%v status=%d type=%d, want compact=%v status=%d type=%d",
					cols.Compact, cols.Status, cols.Type, tt.wantCompact, tt.wantStatus, tt.wantType)
			}
		})
	}
}

func TestRenderBeanRow_Compact(t *testing.T) {
	cfg := BeanRowConfig{StatusColor: "green", TypeColor: "blue"}

	full := RenderBeanRow("abc123", "in-progress", "feature", "Title", cfg)
	if !strings.Contains(full, "active") || !strings.Contains(full, "feat") {
		t.Errorf("full row should contain the labels, got %q", full)
	}

	cfg.Compact = true
	compact := RenderBeanRow("abc123", "in-progress", "feature", "Title", cfg)
	if strings.Contains(compact, "active") || strings.Contains(compact, "feat") {
		t.Errorf("compact row should not contain the labels, got %q", compact)
	}
	if lipgloss.Width(compact) >= lipgloss.Width(full) {
		t.Errorf("compact row (%d) should be narrower than full row (%d)", lipgloss.Width(compact), lipgloss.Width(full))
	}
}

func TestRenderBeanRow_Size(t *testing.T) {
	tests := []struct {
		name     string
		cfg      BeanRowConfig
		want     string
		wantNone bool
	}{
		{name: "hidden by default", cfg: BeanRowConfig{Size: "xl"}, wantNone: true},
		{name: "explicit size", cfg: BeanRowConfig{ShowSize: true, Size: "xl", SizeColor: "red"}, want: "xl"},
		{name: "empty size shows default", cfg: BeanRowConfig{ShowSize: true}, want: "md"},
		{name: "compact", cfg: BeanRowConfig{ShowSize: true, Size: "xs", Compact: true}, want: "xs"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := RenderBeanRow("abc123", "todo", "task", "Title", tt.cfg)
			if tt.wantNone {
				if strings.Contains(row, tt.cfg.Size) {
					t.Errorf("row should not show size, got %q", row)
				}
				return
			}
			if !strings.Contains(row, tt.want+" ") {
				t.Errorf("row should contain size %q, got %q", tt.want, row)
			}
		})
	}
}

func TestScrollTitle(t *testing.T) {
	const title = "abcdefghijklmnop" // 16 runes
	tests := []struct {
		name  string
		width int
		shift int
		want  string
	}{
		{"fits", 20, 0, title},
		{"cut at the end", 10, 0, "abcdefg..."},
		{"scrolled, cut at both ends", 10, 3, "...ghij..."},
		{"scrolled to the end", 10, 6, "...jklmnop"},
		{"shift clamped to the end", 10, 99, "...jklmnop"},
		{"too narrow for dots", 3, 2, "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScrollTitle(title, tt.width, tt.shift)
			if got != tt.want {
				t.Errorf("ScrollTitle(%d, %d) = %q, want %q", tt.width, tt.shift, got, tt.want)
			}
			if len([]rune(got)) > tt.width && tt.width < len(title) {
				t.Errorf("%q is wider than %d", got, tt.width)
			}
		})
	}
}

func TestRenderBeanRow_ColumnOrder(t *testing.T) {
	cfg := BeanRowConfig{ShowSize: true, Size: "xl", MaxTitleWidth: 20, TreePrefix: "└─ "}
	row := ansi.Strip(RenderBeanRow("abc123", "todo", "task", "Title", cfg))
	want := []string{"todo", "└─ Title", "task", "xl", "abc123"}
	last := -1
	for _, w := range want {
		i := strings.Index(row, w)
		if i <= last {
			t.Fatalf("%q out of order in %q", w, row)
		}
		last = i
	}
	if !strings.HasSuffix(row, "abc123") {
		t.Errorf("the ID should come last: %q", row)
	}

	// Titles fill their column, so the IDs line up
	other := ansi.Strip(RenderBeanRow("abc123", "todo", "task", "A much longer title", cfg))
	if strings.Index(other, "abc123") != strings.Index(row, "abc123") {
		t.Errorf("IDs don't line up:\n%s\n%s", row, other)
	}
}

func TestTitleOverflow(t *testing.T) {
	cfg := BeanRowConfig{MaxTitleWidth: 10, TreePrefix: "└─ "} // 7 left for the title
	if got := TitleOverflow("abcdefghij", cfg); got != 3 {
		t.Errorf("TitleOverflow = %d, want 3", got)
	}
	if got := TitleOverflow("abc", cfg); got != 0 {
		t.Errorf("TitleOverflow of a fitting title = %d, want 0", got)
	}
}

func TestRenderBeanRow_LabelsRightAligned(t *testing.T) {
	cfg := BeanRowConfig{MaxTitleWidth: 10}
	row := ansi.Strip(RenderBeanRow("abc123", "in-progress", "feature", "Title", cfg))
	want := strings.Repeat(" ", ColWidthStatus-1-len("active")) + "active Title"
	if !strings.HasPrefix(row, want) {
		t.Errorf("status should be right-aligned as \"active\": %q", row)
	}
	if !strings.Contains(row, strings.Repeat(" ", ColWidthType-1-len("feat"))+"feat abc123") {
		t.Errorf("type should be right-aligned as \"feat\" before the ID: %q", row)
	}
	if done := ansi.Strip(RenderBeanRow("x", "completed", "task", "T", cfg)); !strings.Contains(done, "done") {
		t.Errorf("completed should show as done: %q", done)
	}
}
