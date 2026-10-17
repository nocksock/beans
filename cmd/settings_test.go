package cmd

import (
	"testing"

	"github.com/hmans/beans/internal/config"
)

func TestFormatSetting(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "string", value: "app-", want: "app-\n"},
		{name: "empty string", value: "", want: "\n"},
		{name: "int", value: 4, want: "4\n"},
		{name: "bool", value: true, want: "true\n"},
		{name: "list", value: config.KeyList{"s", "S"}, want: "- s\n- S\n"},
		{name: "struct", value: config.TUIConfig{DisableLauncherWarning: true}, want: "disable_launcher_warning: true\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatSetting(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("formatSetting(%#v) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
