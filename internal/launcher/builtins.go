package launcher

import (
	"embed"
	"path"
	"strings"
)

//go:embed builtin/*.sh
var builtinFS embed.FS

// Builtin is a launch script shipped inside the beans binary.
type Builtin struct {
	Name        string
	Description string // the script's second line, without "# "
	Script      string
}

// Builtins returns all builtin launches, sorted by name.
func Builtins() []Builtin {
	entries, err := builtinFS.ReadDir("builtin")
	if err != nil {
		panic(err) // embedded at build time
	}
	builtins := make([]Builtin, 0, len(entries))
	for _, e := range entries {
		raw, err := builtinFS.ReadFile(path.Join("builtin", e.Name()))
		if err != nil {
			panic(err)
		}
		builtins = append(builtins, newBuiltin(strings.TrimSuffix(e.Name(), ".sh"), string(raw)))
	}
	return builtins
}

// LookupBuiltin returns the builtin with the given name.
func LookupBuiltin(name string) (Builtin, bool) {
	raw, err := builtinFS.ReadFile(path.Join("builtin", name+".sh"))
	if err != nil {
		return Builtin{}, false
	}
	return newBuiltin(name, string(raw)), true
}

func newBuiltin(name, script string) Builtin {
	var description string
	if lines := strings.SplitN(script, "\n", 3); len(lines) > 1 {
		description = strings.TrimSpace(strings.TrimPrefix(lines[1], "#"))
	}
	return Builtin{Name: name, Description: description, Script: script}
}
