package launcher

import (
	"fmt"
	"os"
)

// Launch describes running a script for a bean.
type Launch struct {
	Script      string
	ProjectRoot string // directory of the project's config file (BEANS_ROOT)
	ConfigDir   string // directory of the config file that declared Script; empty means ProjectRoot
	BeansDir    string // beans directory (BEANS_DIR)
	BeanID      string
	BeanTitle   string
}

// WorkDir is where the script runs: the directory of the config file that
// declared it, so relative paths in inherited configs keep working.
func (l Launch) WorkDir() string {
	if l.ConfigDir != "" {
		return l.ConfigDir
	}
	return l.ProjectRoot
}

// Env returns the process environment plus the BEANS_* variables.
func (l Launch) Env() []string {
	return append(os.Environ(),
		fmt.Sprintf("BEANS_ROOT=%s", l.ProjectRoot),
		fmt.Sprintf("BEANS_CONFIG_DIR=%s", l.WorkDir()),
		fmt.Sprintf("BEANS_DIR=%s", l.BeansDir),
		fmt.Sprintf("BEANS_ID=%s", l.BeanID),
		fmt.Sprintf("BEANS_TASK=%s", l.BeanTitle),
	)
}
