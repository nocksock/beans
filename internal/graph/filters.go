package graph

import (
	"strings"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/beancore"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph/model"
)

// ApplyFilter applies BeanFilter to a slice of beans and returns filtered results.
// This is used by both the top-level beans query and relationship field resolvers.
func ApplyFilter(beans []*bean.Bean, filter *model.BeanFilter, core *beancore.Core) []*bean.Bean {
	if filter == nil {
		return beans
	}

	result := beans

	// ID prefix filter
	if filter.IDPrefix != nil && *filter.IDPrefix != "" {
		result = filterByIDPrefix(result, *filter.IDPrefix)
	}

	// Status filters (a builtin status also matches its custom substatuses)
	cfg := core.Config()
	if len(filter.Status) > 0 {
		result = filterByField(result, cfg.ExpandStatuses(filter.Status), func(b *bean.Bean) string { return b.Status })
	}
	if len(filter.ExcludeStatus) > 0 {
		result = excludeByField(result, cfg.ExpandStatuses(filter.ExcludeStatus), func(b *bean.Bean) string { return b.Status })
	}

	// Type filters
	if len(filter.Type) > 0 {
		result = filterByField(result, filter.Type, func(b *bean.Bean) string { return b.Type })
	}
	if len(filter.ExcludeType) > 0 {
		result = excludeByField(result, filter.ExcludeType, func(b *bean.Bean) string { return b.Type })
	}

	// Priority filters (empty priority treated as "normal")
	if len(filter.Priority) > 0 {
		result = filterByPriority(result, filter.Priority)
	}
	if len(filter.ExcludePriority) > 0 {
		result = excludeByPriority(result, filter.ExcludePriority)
	}

	// Size filters (empty size treated as the default size)
	beanSize := func(b *bean.Bean) string { return config.EffectiveSize(b.Size) }
	if len(filter.Size) > 0 {
		result = filterByField(result, filter.Size, beanSize)
	}
	if len(filter.ExcludeSize) > 0 {
		result = excludeByField(result, filter.ExcludeSize, beanSize)
	}

	// Tag filters
	if len(filter.Tags) > 0 {
		result = filterByTags(result, filter.Tags)
	}
	if len(filter.ExcludeTags) > 0 {
		result = excludeByTags(result, filter.ExcludeTags)
	}

	// Parent filters
	if filter.HasParent != nil && *filter.HasParent {
		result = filterByHasParent(result)
	}
	if filter.NoParent != nil && *filter.NoParent {
		result = filterByNoParent(result)
	}
	if filter.ParentID != nil && *filter.ParentID != "" {
		result = filterByParentID(result, *filter.ParentID)
	}

	// Blocking filters
	if filter.HasBlocking != nil && *filter.HasBlocking {
		result = filterByHasBlocking(result, core)
	}
	if filter.BlockingID != nil && *filter.BlockingID != "" {
		result = filterByBlockingID(result, *filter.BlockingID, core)
	}
	if filter.NoBlocking != nil && *filter.NoBlocking {
		result = filterByNoBlocking(result, core)
	}
	if filter.IsBlocked != nil {
		if *filter.IsBlocked {
			result = filterByIsBlocked(result, core)
		} else {
			result = filterByNotBlocked(result, core)
		}
	}

	return result
}

// filterByField filters beans to include only those where getter returns a value in values (OR logic).
func filterByField(beans []*bean.Bean, values []string, getter func(*bean.Bean) string) []*bean.Bean {
	valueSet := make(map[string]bool, len(values))
	for _, v := range values {
		valueSet[v] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		if valueSet[getter(b)] {
			result = append(result, b)
		}
	}
	return result
}

// excludeByField filters beans to exclude those where getter returns a value in values.
func excludeByField(beans []*bean.Bean, values []string, getter func(*bean.Bean) string) []*bean.Bean {
	valueSet := make(map[string]bool, len(values))
	for _, v := range values {
		valueSet[v] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		if !valueSet[getter(b)] {
			result = append(result, b)
		}
	}
	return result
}

// filterByPriority filters beans to include only those with matching priorities (OR logic).
// Empty priority in the bean is treated as "normal" for matching purposes.
func filterByPriority(beans []*bean.Bean, priorities []string) []*bean.Bean {
	prioritySet := make(map[string]bool, len(priorities))
	for _, p := range priorities {
		prioritySet[p] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		priority := b.Priority
		if priority == "" {
			priority = "normal"
		}
		if prioritySet[priority] {
			result = append(result, b)
		}
	}
	return result
}

// excludeByPriority filters beans to exclude those with matching priorities.
// Empty priority in the bean is treated as "normal" for matching purposes.
func excludeByPriority(beans []*bean.Bean, priorities []string) []*bean.Bean {
	prioritySet := make(map[string]bool, len(priorities))
	for _, p := range priorities {
		prioritySet[p] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		priority := b.Priority
		if priority == "" {
			priority = "normal"
		}
		if !prioritySet[priority] {
			result = append(result, b)
		}
	}
	return result
}

// filterByTags filters beans to include only those with any of the given tags (OR logic).
func filterByTags(beans []*bean.Bean, tags []string) []*bean.Bean {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}

	var result []*bean.Bean
	for _, b := range beans {
		for _, t := range b.Tags {
			if tagSet[t] {
				result = append(result, b)
				break
			}
		}
	}
	return result
}

// excludeByTags filters beans to exclude those with any of the given tags.
func excludeByTags(beans []*bean.Bean, tags []string) []*bean.Bean {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}

	var result []*bean.Bean
outer:
	for _, b := range beans {
		for _, t := range b.Tags {
			if tagSet[t] {
				continue outer
			}
		}
		result = append(result, b)
	}
	return result
}

// filterByHasParent filters beans to include only those with a parent.
func filterByHasParent(beans []*bean.Bean) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if b.Parent != "" {
			result = append(result, b)
		}
	}
	return result
}

// filterByNoParent filters beans to include only those without a parent.
func filterByNoParent(beans []*bean.Bean) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if b.Parent == "" {
			result = append(result, b)
		}
	}
	return result
}

// filterByIDPrefix filters beans whose ID starts with prefix.
func filterByIDPrefix(beans []*bean.Bean, prefix string) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if strings.HasPrefix(b.ID, prefix) {
			result = append(result, b)
		}
	}
	return result
}

// filterByParentID filters beans with specific parent ID.
func filterByParentID(beans []*bean.Bean, parentID string) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if b.Parent == parentID {
			result = append(result, b)
		}
	}
	return result
}

// filterByHasBlocking filters beans that are blocking other beans.
func filterByHasBlocking(beans []*bean.Bean, core *beancore.Core) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if len(core.FindBlocked(b.ID)) > 0 {
			result = append(result, b)
		}
	}
	return result
}

// filterByBlockingID filters beans that are blocking a specific bean ID.
func filterByBlockingID(beans []*bean.Bean, targetID string, core *beancore.Core) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		for _, blocked := range core.FindBlocked(b.ID) {
			if blocked.ID == targetID {
				result = append(result, b)
				break
			}
		}
	}
	return result
}

// filterByNoBlocking filters beans that aren't blocking other beans.
func filterByNoBlocking(beans []*bean.Bean, core *beancore.Core) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if len(core.FindBlocked(b.ID)) == 0 {
			result = append(result, b)
		}
	}
	return result
}

// filterByIsBlocked filters beans that are blocked by at least one active blocker.
func filterByIsBlocked(beans []*bean.Bean, core *beancore.Core) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if hasActiveBlocker(b.ID, core) {
			result = append(result, b)
		}
	}
	return result
}

// filterByNotBlocked filters beans that are NOT blocked by any active blocker.
func filterByNotBlocked(beans []*bean.Bean, core *beancore.Core) []*bean.Bean {
	var result []*bean.Bean
	for _, b := range beans {
		if !hasActiveBlocker(b.ID, core) {
			result = append(result, b)
		}
	}
	return result
}

// hasActiveBlocker reports whether any bean blocking targetID (via blocking or
// blocked_by) is still active.
// Blockers with an archive status (e.g. completed, scrapped, archived) no longer block.
func hasActiveBlocker(targetID string, core *beancore.Core) bool {
	cfg := core.Config()
	for _, blocker := range core.FindBlockers(targetID) {
		if cfg != nil && cfg.IsArchiveStatus(blocker.Status) {
			continue
		}
		return true
	}
	return false
}
