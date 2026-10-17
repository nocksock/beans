//go:build windows

package launcher

import "os/exec"

// detach is a no-op on Windows, where console signals aren't delivered by
// process group the same way.
func detach(cmd *exec.Cmd) {}
