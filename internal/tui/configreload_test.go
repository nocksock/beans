package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hmans/beans/internal/config"
)

func TestReloadConfigKey(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantPrefix string
		wantNotice string
		wantErr    bool
	}{
		{name: "applies new config", content: "beans:\n  prefix: new-\n", wantPrefix: "new-", wantNotice: "Config reloaded"},
		{name: "keeps config on error", content: "beans: [", wantPrefix: "", wantNotice: "Config not reloaded", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app, _ := newTestApp(t)
			cfg := app.config
			path := filepath.Join(cfg.ConfigDir(), config.ConfigFileName)
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
			if cmd == nil {
				t.Fatal("R should request a config reload")
			}
			app.Update(cmd())

			if app.config != cfg || app.core.Config() != cfg {
				t.Error("config should be updated in place")
			}
			if cfg.Beans.Prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", cfg.Beans.Prefix, tt.wantPrefix)
			}
			if !strings.HasPrefix(app.list.notice, tt.wantNotice) || app.list.noticeIsErr != tt.wantErr {
				t.Errorf("notice = %q (err %v), want %q (err %v)", app.list.notice, app.list.noticeIsErr, tt.wantNotice, tt.wantErr)
			}
			if !strings.Contains(app.View(), tt.wantNotice) {
				t.Error("notice should be shown in the footer")
			}
		})
	}
}

func TestClearNoticeIgnoresStale(t *testing.T) {
	app, _ := newTestApp(t)
	app.showNotice("first", false)
	stale := app.noticeSeq
	app.showNotice("second", false)

	app.Update(clearNoticeMsg{seq: stale})
	if app.list.notice != "second" {
		t.Errorf("stale clear removed newer notice, got %q", app.list.notice)
	}

	app.Update(clearNoticeMsg{seq: app.noticeSeq})
	if app.list.notice != "" {
		t.Errorf("notice = %q, want cleared", app.list.notice)
	}
}
