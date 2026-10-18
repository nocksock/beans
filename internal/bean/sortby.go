package bean

import (
	"cmp"
	"slices"
	"strings"
	"time"

	"github.com/hmans/beans/internal/config"
)

// SortFields lists the fields SortBy understands.
var SortFields = []string{"status", "priority", "type", "size", "created", "updated", "title", "id"}

// SortOrders holds the configured orderings of the ranked fields.
type SortOrders struct {
	Statuses   []string
	Priorities []string
	Types      []string
	Sizes      []string
}

// SortOrdersFrom returns the orderings configured in cfg.
func SortOrdersFrom(cfg *config.Config) SortOrders {
	return SortOrders{
		Statuses:   cfg.StatusNames(),
		Priorities: cfg.PriorityNames(),
		Types:      cfg.TypeNames(),
		Sizes:      cfg.SizeNames(),
	}
}

// DefaultSortDesc reports whether field sorts descending when no direction is given.
// Timestamps default to newest first, everything else to ascending.
func DefaultSortDesc(field string) bool {
	return field == "created" || field == "updated"
}

// SortBy sorts beans by field, descending if desc. Ties are broken by ID ascending,
// and beans without a timestamp come last regardless of direction.
// An empty or unknown field falls back to SortByStatusPriorityAndType.
func SortBy(beans []*Bean, field string, desc bool, o SortOrders) {
	compare := fieldComparator(field, o)
	if compare == nil {
		SortByStatusPriorityAndType(beans, o.Statuses, o.Priorities, o.Types)
		return
	}
	slices.SortStableFunc(beans, func(a, b *Bean) int {
		if c := compare(a, b, desc); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
}

// fieldComparator returns a direction-aware comparison for field, or nil if unknown.
func fieldComparator(field string, o SortOrders) func(a, b *Bean, desc bool) int {
	ranked := func(names []string, value func(*Bean) string) func(a, b *Bean, desc bool) int {
		return func(a, b *Bean, desc bool) int {
			return directed(cmp.Compare(rank(names, value(a)), rank(names, value(b))), desc)
		}
	}
	timed := func(value func(*Bean) *time.Time) func(a, b *Bean, desc bool) int {
		return func(a, b *Bean, desc bool) int { return compareTimes(value(a), value(b), desc) }
	}

	switch field {
	case "status":
		return ranked(o.Statuses, func(b *Bean) string { return b.Status })
	case "priority":
		return ranked(o.Priorities, func(b *Bean) string { return cmp.Or(b.Priority, "normal") })
	case "type":
		return ranked(o.Types, func(b *Bean) string { return b.Type })
	case "size":
		return ranked(o.Sizes, func(b *Bean) string { return config.EffectiveSize(b.Size) })
	case "created":
		return timed(func(b *Bean) *time.Time { return b.CreatedAt })
	case "updated":
		return timed(func(b *Bean) *time.Time { return b.UpdatedAt })
	case "title":
		return func(a, b *Bean, desc bool) int {
			return directed(strings.Compare(strings.ToLower(a.DisplayTitle()), strings.ToLower(b.DisplayTitle())), desc)
		}
	case "id":
		return func(a, b *Bean, desc bool) int { return directed(cmp.Compare(a.ID, b.ID), desc) }
	}
	return nil
}

// rank returns the index of value in names, or len(names) if it is unrecognized.
func rank(names []string, value string) int {
	if i := slices.Index(names, value); i >= 0 {
		return i
	}
	return len(names)
}

func directed(c int, desc bool) int {
	if desc {
		return -c
	}
	return c
}

// compareTimes orders timestamps by direction, with nil always last.
func compareTimes(a, b *time.Time, desc bool) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	}
	return directed(a.Compare(*b), desc)
}
