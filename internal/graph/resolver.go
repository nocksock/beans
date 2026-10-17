package graph

import (
	"fmt"

	"github.com/hmans/beans/internal/beancore"
)

//go:generate go tool gqlgen generate

// Resolver is the root resolver for the GraphQL schema.
// It holds a reference to beancore.Core for data access.
type Resolver struct {
	Core *beancore.Core
}

// validateSize rejects sizes that aren't configured. Empty is allowed and
// clears the size (which then counts as the default size).
func (r *Resolver) validateSize(size string) error {
	cfg := r.Core.Config()
	if !cfg.IsValidSize(size) {
		return fmt.Errorf("invalid size: %s (must be %s)", size, cfg.SizeList())
	}
	return nil
}
