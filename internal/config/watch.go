package config

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const watchDebounce = 100 * time.Millisecond

// Watch calls onChange (debounced) whenever a config file is created, changed
// or removed in startDir or any directory above it, i.e. anywhere a file
// affecting the config loaded from startDir could live. Directories that
// can't be watched are skipped. Call the returned function to stop watching.
func Watch(startDir string, onChange func()) (func(), error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	for {
		_ = watcher.Add(dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	done := make(chan struct{})
	go func() {
		defer watcher.Close()
		var timer *time.Timer
		for {
			select {
			case <-done:
				if timer != nil {
					timer.Stop()
				}
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Base(event.Name) != ConfigFileName || event.Op == fsnotify.Chmod {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(watchDebounce, onChange)
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	return func() { close(done) }, nil
}
