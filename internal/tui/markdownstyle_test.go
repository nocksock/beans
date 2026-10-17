package tui

import (
	"testing"

	"github.com/charmbracelet/glamour/styles"
)

func TestResolveMarkdownStyle(t *testing.T) {
	tests := []struct {
		name string
		env  string
		dark bool
		want string
	}{
		{"dark background", "", true, styles.DarkStyle},
		{"light background", "", false, styles.LightStyle},
		{"env overrides light background", "dracula", false, "dracula"},
		{"env overrides dark background", styles.LightStyle, true, styles.LightStyle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveMarkdownStyle(tt.env, func() bool { return tt.dark })
			if got != tt.want {
				t.Errorf("resolveMarkdownStyle() = %q, want %q", got, tt.want)
			}
		})
	}
}
