package launcher

import (
	"fmt"
	"strings"

	"github.com/hmans/beans/internal/config"
)

const (
	builtinPrefix = "builtin."
	parentPrefix  = "parent."
)

// Resolved is a launch script found by name.
type Resolved struct {
	Name    string // configured name, or "builtin.<name>" for builtins
	Script  string
	Dir     string // directory of the config file that declared it; empty for builtins
	Builtin bool
}

// Resolve finds the script for a launch name:
//   - "builtin.X" is always the embedded builtin X
//   - "parent.X" is X as declared by the nearest config layer above
//     callerLayer (above the leaf config when empty), else builtin X
//   - "X" is the configured launcher X, else builtin X
func Resolve(cfg *config.Config, name, callerLayer string) (Resolved, error) {
	if base, ok := strings.CutPrefix(name, builtinPrefix); ok {
		return resolveBuiltin(base)
	}
	if base, ok := strings.CutPrefix(name, parentPrefix); ok {
		l, found, err := cfg.ParentLauncher(base, callerLayer)
		if err != nil {
			return Resolved{}, err
		}
		if found {
			return fromConfig(l), nil
		}
		return resolveBuiltin(base)
	}
	if l, ok := cfg.Launcher(name); ok {
		return fromConfig(l), nil
	}
	if b, ok := LookupBuiltin(name); ok {
		return fromBuiltin(b), nil
	}
	return Resolved{}, fmt.Errorf("unknown launch %q, see beans launch --list", name)
}

func resolveBuiltin(name string) (Resolved, error) {
	b, ok := LookupBuiltin(name)
	if !ok {
		return Resolved{}, fmt.Errorf("unknown builtin %q, see beans launch --list", name)
	}
	return fromBuiltin(b), nil
}

func fromConfig(l config.Launcher) Resolved {
	return Resolved{Name: l.Name, Script: l.Exec, Dir: l.Dir}
}

func fromBuiltin(b Builtin) Resolved {
	return Resolved{Name: builtinPrefix + b.Name, Script: b.Script, Builtin: true}
}
