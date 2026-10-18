package graph

import (
	"fmt"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/githistory"
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

// newBeanID generates an ID with the given prefix and the configured length.
// Returns "" for a nil prefix, so beancore generates one with the configured
// prefix.
func (r *Resolver) newBeanID(prefix *string) string {
	if prefix == nil {
		return ""
	}
	length := 4
	if cfg := r.Core.Config(); cfg != nil && cfg.Beans.IDLength > 0 {
		length = cfg.Beans.IDLength
	}
	return bean.NewID(*prefix, length)
}

// historyFile locates a bean's file for reading its git history.
func (r *Resolver) historyFile(b *bean.Bean) githistory.File {
	return githistory.File{Dir: r.Core.Root(), ID: b.ID}
}
