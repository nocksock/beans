package graph

import (
	"fmt"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/graph/model"
)

// applyBodyModification applies all replacements in order, then the append.
// It returns an error, and leaves body untouched, if any replacement fails.
func applyBodyModification(body string, mod *model.BodyModification) (string, error) {
	for i, op := range mod.Replace {
		replaced, err := bean.ReplaceOnce(body, op.Old, op.New)
		if err != nil {
			return "", fmt.Errorf("replacement %d failed: %w", i, err)
		}
		body = replaced
	}
	if mod.Append != nil {
		body = bean.AppendWithSeparator(body, *mod.Append)
	}
	return body, nil
}
