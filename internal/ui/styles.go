package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/hmans/beans/internal/config"
)

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#6B7280") // Gray
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorDanger    = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#9CA3AF") // Light gray
	ColorSubtle    = lipgloss.Color("#555555") // Dark gray (for tree lines)
	ColorBlue      = lipgloss.Color("#3B82F6") // Blue
	ColorCyan      = lipgloss.Color("14")      // Bright Cyan (ANSI)
)

// NamedColors maps color names to lipgloss colors.
var NamedColors = map[string]lipgloss.Color{
	"green":  ColorSuccess,
	"yellow": ColorWarning,
	"red":    ColorDanger,
	"gray":   ColorSecondary,
	"grey":   ColorSecondary,
	"blue":   ColorBlue,
	"purple": ColorPrimary,
	"cyan":   ColorCyan,
}

// ResolveColor converts a color name or hex code to a lipgloss.Color.
func ResolveColor(color string) lipgloss.Color {
	if strings.HasPrefix(color, "#") {
		return lipgloss.Color(color)
	}
	if c, ok := NamedColors[strings.ToLower(color)]; ok {
		return c
	}
	return ColorMuted
}

// IsValidColor returns true if the color is a valid named color or hex code.
func IsValidColor(color string) bool {
	if strings.HasPrefix(color, "#") {
		// Valid hex: #RGB or #RRGGBB
		return len(color) == 4 || len(color) == 7
	}
	_, ok := NamedColors[strings.ToLower(color)]
	return ok
}

// Tag badge style - black text on gray background
var TagBadge = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#000")).
	Background(ColorMuted).
	Padding(0, 1)

// RenderTag renders a single tag as a badge
func RenderTag(tag string) string {
	return TagBadge.Render(tag)
}

// RenderTags renders multiple tags as badges separated by spaces
func RenderTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	rendered := make([]string, len(tags))
	for i, tag := range tags {
		rendered[i] = RenderTag(tag)
	}
	return strings.Join(rendered, " ")
}

// RenderTagsCompact renders tags for list views with a max count.
// Shows up to maxTags badges, with "+N" indicator if there are more.
// Tags longer than 12 chars are truncated.
func RenderTagsCompact(tags []string, maxTags int) string {
	if len(tags) == 0 {
		return ""
	}
	if maxTags <= 0 {
		maxTags = 1
	}

	showTags := tags
	var extra int
	if len(tags) > maxTags {
		showTags = tags[:maxTags]
		extra = len(tags) - maxTags
	}

	rendered := make([]string, len(showTags))
	for i, tag := range showTags {
		// Truncate long tags
		displayTag := tag
		if len(displayTag) > 12 {
			displayTag = displayTag[:10] + ".."
		}
		rendered[i] = RenderTag(displayTag)
	}

	result := strings.Join(rendered, " ")
	if extra > 0 {
		result += Muted.Render(fmt.Sprintf(" +%d", extra))
	}
	return result
}

// Text styles
var (
	Bold    = lipgloss.NewStyle().Bold(true)
	Muted   = lipgloss.NewStyle().Foreground(ColorMuted)
	Primary = lipgloss.NewStyle().Foreground(ColorPrimary)
	Success = lipgloss.NewStyle().Foreground(ColorSuccess)
	Warning = lipgloss.NewStyle().Foreground(ColorWarning)
	Danger  = lipgloss.NewStyle().Foreground(ColorDanger)
)

// ID style - distinctive for bean IDs
var ID = lipgloss.NewStyle().
	Foreground(ColorPrimary).
	Bold(true)

// TreeLine style - subtle for tree connectors
var TreeLine = lipgloss.NewStyle().Foreground(ColorSubtle)

// Title style
var Title = lipgloss.NewStyle().Bold(true)

// RenderStatusWithColor returns a styled status badge using the specified color.
func RenderStatusWithColor(status, color string, isArchiveStatus bool) string {
	c := ResolveColor(color)
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#fff")).
		Background(c).
		Padding(0, 1)

	if !isArchiveStatus {
		style = style.Bold(true)
	}

	return style.Render(status)
}

// RenderStatusTextWithColor returns styled status text (for tables) using the specified color.
func RenderStatusTextWithColor(status, color string, isArchiveStatus bool) string {
	c := ResolveColor(color)
	style := lipgloss.NewStyle().Foreground(c)

	if !isArchiveStatus {
		style = style.Bold(true)
	}

	return style.Render(status)
}

// RenderTypeText returns styled type text using the specified color.
// If color is empty, uses muted styling.
func RenderTypeText(typeName, color string) string {
	if typeName == "" {
		return ""
	}
	if color == "" {
		return Muted.Render(typeName)
	}
	c := ResolveColor(color)
	return lipgloss.NewStyle().Foreground(c).Render(typeName)
}

// RenderPriorityWithColor returns a styled priority badge using the specified color.
func RenderPriorityWithColor(priority, color string) string {
	if priority == "" {
		return ""
	}
	c := ResolveColor(color)
	style := lipgloss.NewStyle().
		Foreground(c).
		Bold(priority == "critical" || priority == "high")
	return style.Render("[" + priority + "]")
}

// RenderPriorityText returns styled priority text for tables.
func RenderPriorityText(priority, color string) string {
	if priority == "" {
		return ""
	}
	c := ResolveColor(color)
	style := lipgloss.NewStyle().Foreground(c)
	if priority == "critical" || priority == "high" {
		style = style.Bold(true)
	}
	return style.Render(priority)
}

// SizeColor returns the configured color for a size (empty = default size).
func SizeColor(cfg *config.Config, size string) string {
	if s := cfg.GetSize(size); s != nil {
		return s.Color
	}
	return ""
}

// RenderSizeText returns styled size text for tables. An empty size renders
// as the default size.
func RenderSizeText(size, color string) string {
	size = config.EffectiveSize(size)
	if color == "" {
		return Muted.Render(size)
	}
	return lipgloss.NewStyle().Foreground(ResolveColor(color)).Render(size)
}

// RenderSizeWithColor returns a styled size badge using the specified color.
func RenderSizeWithColor(size, color string) string {
	return RenderSizeText("["+config.EffectiveSize(size)+"]", color)
}

// ShortCode returns a single-character code for a type or status name,
// e.g. "in-progress" -> "I", "milestone" -> "M".
func ShortCode(name string) string {
	if name == "" {
		return ""
	}
	return strings.ToUpper(string([]rune(name)[0]))
}

// GetPrioritySymbol returns the raw symbol for a priority without styling.
// Returns empty string for normal/empty priority.
func GetPrioritySymbol(priority string) string {
	switch priority {
	case "critical":
		return "‼"
	case "high":
		return "!"
	case "low":
		return "↓"
	case "deferred":
		return "→"
	default:
		return ""
	}
}

// RenderPrioritySymbol returns a compact symbol for priority (used in TUI).
// Returns empty string for normal/empty priority.
func RenderPrioritySymbol(priority, color string) string {
	symbol := GetPrioritySymbol(priority)
	if symbol == "" {
		return ""
	}

	c := ResolveColor(color)
	style := lipgloss.NewStyle().Foreground(c)
	if priority == "critical" || priority == "high" {
		style = style.Bold(true)
	}
	return style.Render(symbol)
}

// BeanRowConfig holds configuration for rendering a bean row
type BeanRowConfig struct {
	StatusColor   string
	TypeColor     string
	PriorityColor string
	Priority      string // Priority value (critical, high, normal, low, deferred)
	Size          string // Size value (xs, sm, md, lg, xl); empty renders as the default size
	SizeColor     string
	ShowSize      bool // Whether to show the size column
	IsArchive     bool
	MaxTitleWidth int  // Width of the title column, including tree prefix and priority symbol; 0 means no truncation
	ShowCursor    bool // Show selection cursor
	IsSelected    bool
	IsMarked      bool     // Marked for multi-select batch operations
	Tags          []string // Tags to display (optional)
	ShowTags      bool     // Whether to show tags column
	TagsColWidth  int      // Width of tags column (0 = default)
	MaxTags       int      // Max tags to show (0 = default of 1)
	TreePrefix    string   // Tree prefix (e.g., "├─" or "  └─") to prepend to the title
	Dimmed        bool     // Render row dimmed (for unmatched ancestor beans in tree)
	TitleMuted    bool     // Render the title muted (e.g. a body snippet shown for an untitled bean)
	TitleShift    int      // How far a cut-off title is scrolled (see ScrollTitle)
	HasChildren   bool     // Whether this bean has children (for caret indicator)
	IsCollapsed   bool     // Whether this bean's children are collapsed
	Compact       bool     // Show single-character type/status codes instead of full names
}

// Base column widths for bean lists (minimum sizes)
const (
	ColWidthID     = 12
	ColWidthStatus = 9  // the longest status label plus a space
	ColWidthType   = 10 // the longest type label plus a space
	ColWidthTags   = 24
	ColWidthSize   = 3 // two-letter size plus a space

	// Compact type/status columns: one character plus a space
	ColWidthCompact = 2
)

// CompactColumnsMaxWidth is the width below which type/status are abbreviated.
const CompactColumnsMaxWidth = 100

// ResponsiveColumns holds calculated column widths based on available space
type ResponsiveColumns struct {
	ID       int
	Status   int
	Type     int
	Tags     int
	MaxTags  int // How many tags to show
	ShowTags bool
	Compact  bool // Single-character type/status codes (narrow viewports)
}

// CalculateResponsiveColumns determines column widths based on available width.
// Prioritizes title width - tags are only shown when there's plenty of room.
func CalculateResponsiveColumns(totalWidth int, hasTags bool) ResponsiveColumns {
	cols := ResponsiveColumns{
		ID:       ColWidthID,
		Status:   ColWidthStatus,
		Type:     ColWidthType,
		Tags:     0,
		MaxTags:  0,
		ShowTags: false,
	}

	// Abbreviate type/status in narrow viewports to leave room for the title
	if totalWidth < CompactColumnsMaxWidth {
		cols.Compact = true
		cols.Status = ColWidthCompact
		cols.Type = ColWidthCompact
	}

	// Don't show tags in narrow viewports - prioritize title space
	// Only consider showing tags if terminal is wide enough (140+ columns)
	const minWidthForTags = 140

	if !hasTags || totalWidth < minWidthForTags {
		return cols
	}

	// At this point we have at least 140 columns
	// Base usage: cursor (2) + ID (12) + status (14) + type (12) = 40
	cursorWidth := 2
	baseWidth := cursorWidth + cols.ID + cols.Status + cols.Type
	available := totalWidth - baseWidth

	// Reserve generous space for title, then allocate remaining to tags
	minTitleWidth := 50
	spaceForTags := available - minTitleWidth

	if spaceForTags >= ColWidthTags {
		cols.ShowTags = true

		if spaceForTags >= 80 {
			// Lots of space: show all tags (up to 5)
			cols.Tags = 70
			cols.MaxTags = 5
		} else if spaceForTags >= 60 {
			// Good space: show 4 tags
			cols.Tags = 55
			cols.MaxTags = 4
		} else if spaceForTags >= 45 {
			// Moderate space: show 3 tags
			cols.Tags = 42
			cols.MaxTags = 3
		} else if spaceForTags >= 35 {
			// Limited space: show 2 tags
			cols.Tags = 32
			cols.MaxTags = 2
		} else {
			// Minimal: show 1 tag
			cols.Tags = ColWidthTags
			cols.MaxTags = 1
		}
	}

	return cols
}

// columnLabels are shorter names for statuses and types in bean rows
var columnLabels = map[string]string{
	"in-progress": "active",
	"completed":   "done",
	"feature":     "feat",
}

// ColumnLabel is the label of a status or type in bean rows.
func ColumnLabel(name string) string {
	if label, ok := columnLabels[name]; ok {
		return label
	}
	return name
}

// RenderBeanRow renders a bean as a single row: status, type, title, tags
// (optional), size (optional) and ID. Status and type are right-aligned.
func RenderBeanRow(id, status, typeName, title string, cfg BeanRowConfig) string {
	// Epics group work rather than being worked on, so their status is left out
	if typeName == "epic" {
		status = ""
	}

	// Status and type columns, each followed by a space
	typeWidth, statusWidth := ColWidthType, ColWidthStatus
	if cfg.Compact {
		typeWidth, statusWidth = ColWidthCompact, ColWidthCompact
		typeName = ShortCode(typeName)
		status = ShortCode(status)
	} else {
		typeName = ColumnLabel(typeName)
		status = ColumnLabel(status)
	}
	typeStyle := lipgloss.NewStyle().Width(typeWidth - 1).Align(lipgloss.Right).MarginRight(1)
	statusStyle := lipgloss.NewStyle().Width(statusWidth - 1).Align(lipgloss.Right).MarginRight(1)

	tagsColWidth := ColWidthTags
	if cfg.TagsColWidth > 0 {
		tagsColWidth = cfg.TagsColWidth
	}
	tagsStyle := lipgloss.NewStyle().Width(tagsColWidth)

	maxTags := 1
	if cfg.MaxTags > 0 {
		maxTags = cfg.MaxTags
	}

	// Highlight style for marked rows
	highlightStyle := lipgloss.NewStyle().Foreground(ColorWarning)

	// The ID comes last
	var idCol string
	switch {
	case cfg.Dimmed:
		idCol = Muted.Render(id)
	case cfg.IsMarked:
		idCol = highlightStyle.Render(id)
	default:
		idCol = ID.Render(id)
	}

	caret := rowCaret(cfg)
	lead := caret + cfg.TreePrefix
	var leadStyled string
	switch {
	case cfg.Dimmed:
		leadStyled = Muted.Render(lead)
	case cfg.IsMarked:
		leadStyled = highlightStyle.Render(lead)
	default:
		leadStyled = Muted.Render(caret) + TreeLine.Render(cfg.TreePrefix)
	}

	var typeCol string
	if typeName != "" {
		if cfg.Dimmed {
			typeCol = typeStyle.Render(Muted.Render(typeName))
		} else {
			typeCol = typeStyle.Render(RenderTypeText(typeName, cfg.TypeColor))
		}
	} else {
		typeCol = typeStyle.Render("")
	}

	var statusCol string
	if cfg.Dimmed {
		statusCol = statusStyle.Render(Muted.Render(status))
	} else {
		statusCol = statusStyle.Render(RenderStatusTextWithColor(status, cfg.StatusColor, cfg.IsArchive))
	}

	// Size column (optional)
	var sizeCol string
	if cfg.ShowSize {
		sizeStyle := lipgloss.NewStyle().Width(ColWidthSize)
		if cfg.Dimmed {
			sizeCol = sizeStyle.Render(Muted.Render(config.EffectiveSize(cfg.Size)))
		} else {
			sizeCol = sizeStyle.Render(RenderSizeText(cfg.Size, cfg.SizeColor))
		}
	}

	// Tags column (optional)
	var tagsCol string
	if cfg.ShowTags {
		if cfg.Dimmed {
			if len(cfg.Tags) > 0 {
				tagsCol = tagsStyle.Render(Muted.Render(cfg.Tags[0]))
			} else {
				tagsCol = tagsStyle.Render("")
			}
		} else {
			tagsCol = tagsStyle.Render(RenderTagsCompact(cfg.Tags, maxTags))
		}
	}

	// The title column holds the lead, priority symbol and title. The title
	// is cut to fit, scrolled by TitleShift.
	prioritySymbol := rowPrioritySymbol(cfg)
	titleColWidth := cfg.MaxTitleWidth
	displayTitle := title
	if titleColWidth > 0 {
		displayTitle = ScrollTitle(title, titleSpace(cfg), cfg.TitleShift)
	}

	// Cursor and title styling
	var cursor string
	var titleStyled string
	if cfg.ShowCursor {
		if cfg.IsSelected {
			cursor = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true).Render("▌")
			titleStyled = lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render(displayTitle)
		} else {
			cursor = " "
			if cfg.Dimmed || cfg.TitleMuted {
				titleStyled = Muted.Render(displayTitle)
			} else {
				titleStyled = displayTitle
			}
		}
	} else {
		cursor = ""
		if cfg.Dimmed || cfg.TitleMuted {
			titleStyled = Muted.Render(displayTitle)
		} else {
			titleStyled = displayTitle
		}
	}

	// Pad the title column so the columns after it line up
	titleLen := ansi.StringWidth(lead) + ansi.StringWidth(prioritySymbol) + ansi.StringWidth(displayTitle)
	padding := strings.Repeat(" ", max(0, titleColWidth-titleLen))

	row := cursor + statusCol + typeCol + leadStyled + prioritySymbol + titleStyled + padding + " "
	if cfg.ShowTags {
		row += tagsCol
	}
	return row + sizeCol + idCol
}

// rowCaret is the indicator for rows with children, leading the tree prefix.
func rowCaret(cfg BeanRowConfig) string {
	switch {
	case !cfg.HasChildren:
		return ""
	case cfg.IsCollapsed:
		return "▸ " // Right-pointing triangle (collapsed)
	default:
		return "▾ " // Down-pointing triangle (expanded)
	}
}

// rowPrioritySymbol is the priority symbol and a space before the title, if any.
func rowPrioritySymbol(cfg BeanRowConfig) string {
	if cfg.Dimmed {
		return ""
	}
	if symbol := RenderPrioritySymbol(cfg.Priority, cfg.PriorityColor); symbol != "" {
		return symbol + " "
	}
	return ""
}

// titleSpace is the width left for the title text in the title column.
func titleSpace(cfg BeanRowConfig) int {
	lead := ansi.StringWidth(rowCaret(cfg) + cfg.TreePrefix)
	return max(cfg.MaxTitleWidth-lead-ansi.StringWidth(rowPrioritySymbol(cfg)), 1)
}

// TitleOverflow is how far the title of a row rendered with cfg can be
// scrolled by TitleShift, or 0 if it fits.
func TitleOverflow(title string, cfg BeanRowConfig) int {
	if cfg.MaxTitleWidth <= 0 {
		return 0
	}
	return max(0, len([]rune(title))-titleSpace(cfg))
}

// ScrollTitle cuts title to width, marking cut-off text with "...". shift
// scrolls it: 0 shows the start, TitleOverflow the end.
func ScrollTitle(title string, width, shift int) string {
	r := []rune(title)
	if len(r) <= width {
		return title
	}
	if width <= 3 {
		return string(r[:width])
	}
	shift = max(0, min(shift, len(r)-width))
	start, end := shift, shift+width
	head, tail := "", ""
	if start > 0 {
		head, start = "...", start+3
	}
	if end < len(r) {
		tail, end = "...", end-3
	}
	if start >= end {
		return string(r[shift : shift+width])
	}
	return head + string(r[start:end]) + tail
}
