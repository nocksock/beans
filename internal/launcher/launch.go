package launcher

import (
	"fmt"
	"os"
)

// Launch describes running a script for a bean.
type Launch struct {
	Name        string // name the launch was invoked as; the script's $0
	Script      string
	Args        []string // the script's positional parameters
	Builtin     bool     // an embedded builtin: runs with bash in the caller's directory
	ProjectRoot string   // directory of the project's config file (BEANS_ROOT)
	ConfigDir   string   // directory of the config file that declared Script; empty means ProjectRoot
	ConfigPath  string   // config file nested `beans launch` calls load; empty when there is none
	BeansDir    string   // beans directory (BEANS_DIR)
	BeanID      string
	BeanTitle   string
	Depth       int // how many launches this one is nested in
}

// WorkDir is where the script runs: the directory of the config file that
// declared it, so relative paths in inherited configs keep working. Builtins
// run where they were called from, which "" means for exec.Cmd.
func (l Launch) WorkDir() string {
	if l.Builtin {
		return ""
	}
	return l.configDir()
}

func (l Launch) configDir() string {
	if l.ConfigDir != "" {
		return l.ConfigDir
	}
	return l.ProjectRoot
}

// layer is the config directory `parent.` lookups of nested calls start
// from; builtins belong to no config layer.
func (l Launch) layer() string {
	if l.Builtin {
		return ""
	}
	return l.configDir()
}

// Env returns the process environment plus the BEANS_* variables, including
// the ones nested `beans launch` calls use to find the same config.
func (l Launch) Env() []string {
	bin, _ := os.Executable()
	return append(os.Environ(),
		fmt.Sprintf("BEANS_ROOT=%s", l.ProjectRoot),
		fmt.Sprintf("BEANS_CONFIG_DIR=%s", l.configDir()),
		fmt.Sprintf("BEANS_DIR=%s", l.BeansDir),
		fmt.Sprintf("BEANS_ID=%s", l.BeanID),
		fmt.Sprintf("BEANS_TASK=%s", l.BeanTitle),
		fmt.Sprintf("BEANS_BIN=%s", bin),
		fmt.Sprintf("BEANS_LAUNCH_CONFIG=%s", l.ConfigPath),
		fmt.Sprintf("BEANS_LAUNCH_LAYER=%s", l.layer()),
		fmt.Sprintf("BEANS_LAUNCH_DEPTH=%d", l.Depth+1),
	)
}
