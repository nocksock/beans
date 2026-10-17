package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// noticeDuration is how long a footer notice stays visible
const noticeDuration = 3 * time.Second

// configChangedMsg is sent when a config file changes on disk (via file watcher)
type configChangedMsg struct{}

// reloadConfigMsg requests re-reading the config from disk
type reloadConfigMsg struct{}

// clearNoticeMsg clears the footer notice, unless a newer one replaced it
type clearNoticeMsg struct {
	seq int
}

// reloadConfig re-reads the config from disk and applies it in place, so
// every view holding the config sees the new settings. On error the current
// config is kept and the error is shown in the footer.
func (a *App) reloadConfig() tea.Cmd {
	cfg, err := a.config.Reload()
	if err != nil {
		return a.showNotice(fmt.Sprintf("Config not reloaded: %v", err), true)
	}
	keyWarnings, err := applyKeymap(cfg)
	if err != nil {
		return a.showNotice(fmt.Sprintf("Config not reloaded: keybinding: %v", err), true)
	}
	a.list.list.KeyMap = mainListKeyMap()

	a.core.ReplaceConfig(cfg)
	if a.config != a.core.Config() {
		*a.config = *cfg
	}

	if a.state == viewDetail {
		if updated, err := a.resolver.Query().Bean(context.Background(), a.detail.bean.ID); err == nil && updated != nil {
			a.detail = newDetailModel(updated, a.resolver, a.config, a.width, a.height)
		}
	}

	if len(keyWarnings) > 0 {
		return tea.Batch(a.showNotice("Config reloaded; "+keyWarningNotice(keyWarnings), true), a.list.loadBeans)
	}
	return tea.Batch(a.showNotice("Config reloaded", false), a.list.loadBeans)
}

// keyWarningNotice summarizes key binding conflicts for the footer.
func keyWarningNotice(warnings []string) string {
	msg := "key conflict: " + warnings[0]
	if len(warnings) > 1 {
		msg += fmt.Sprintf(" (+%d more)", len(warnings)-1)
	}
	return msg
}

// showNotice shows msg in the list footer for noticeDuration.
func (a *App) showNotice(msg string, isErr bool) tea.Cmd {
	a.noticeSeq++
	seq := a.noticeSeq
	a.list.notice = msg
	a.list.noticeIsErr = isErr
	return tea.Tick(noticeDuration, func(time.Time) tea.Msg {
		return clearNoticeMsg{seq: seq}
	})
}
