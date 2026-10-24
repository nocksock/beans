package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph/model"
)

// sortSpec is the sort order requested by a sort: token. An empty Field means the default order.
type sortSpec struct {
	Field string
	Desc  bool
}

// query is a parsed list filter query.
type query struct {
	Filter   *model.BeanFilter // field:value and -field:value tokens
	Text     string            // remaining words, fuzzy-matched against title and ID
	Sort     sortSpec
	Warnings []string // tokens that were ignored, for display
}

// parseQuery parses whitespace-separated tokens: field:value includes (same field ORed,
// different fields ANDed), -field:value excludes, prefix:value matches the start of bean
// IDs, sort:field[:asc|desc] sorts, and every other word is free text. Unknown fields or
// values are reported as warnings and ignored.
// Tokens with an empty value are ignored silently, since the user is likely still typing.
func parseQuery(input string, cfg *config.Config) query {
	q := query{Filter: &model.BeanFilter{}}
	var text []string

	for _, token := range strings.Fields(input) {
		rawField, value, ok := strings.Cut(token, ":")
		if !ok {
			text = append(text, token)
			continue
		}
		field := strings.ToLower(rawField)
		if value == "" {
			continue
		}
		// Prefixes match bean IDs case-sensitively and are not a list field
		if field == "prefix" {
			prefix := value
			q.Filter.IDPrefix = &prefix
			continue
		}
		value = strings.ToLower(value)
		if field == "sort" {
			q.parseSort(value)
			continue
		}

		exclude := strings.HasPrefix(field, "-")
		field = strings.TrimPrefix(field, "-")
		include, excl, valid := filterTarget(q.Filter, field, cfg)
		switch {
		case include == nil:
			q.warn("unknown field %q", field)
		case valid != nil && !slices.Contains(valid, value):
			q.warn("unknown %s %q", field, value)
		case exclude:
			*excl = append(*excl, value)
		default:
			*include = append(*include, value)
		}
	}

	q.Text = strings.Join(text, " ")
	return q
}

// parseSort applies the value of a sort: token.
func (q *query) parseSort(value string) {
	field, dir, _ := strings.Cut(value, ":")
	if !slices.Contains(bean.SortFields, field) {
		q.warn("unknown sort field %q", field)
		return
	}
	desc := bean.DefaultSortDesc(field)
	switch dir {
	case "":
	case "asc":
		desc = false
	case "desc":
		desc = true
	default:
		q.warn("unknown sort direction %q", dir)
		return
	}
	q.Sort = sortSpec{Field: field, Desc: desc}
}

func (q *query) warn(format string, args ...any) {
	q.Warnings = append(q.Warnings, fmt.Sprintf(format, args...))
}

// filterTarget returns the include and exclude lists of f for field, and the valid values
// (nil if any value is allowed). include is nil for an unknown field.
func filterTarget(f *model.BeanFilter, field string, cfg *config.Config) (include, exclude *[]string, valid []string) {
	switch field {
	case "status":
		return &f.Status, &f.ExcludeStatus, cfg.StatusNames()
	case "type":
		return &f.Type, &f.ExcludeType, cfg.TypeNames()
	case "priority":
		return &f.Priority, &f.ExcludePriority, cfg.PriorityNames()
	case "size":
		return &f.Size, &f.ExcludeSize, cfg.SizeNames()
	case "tag":
		return &f.Tags, &f.ExcludeTags, nil
	}
	return nil, nil, nil
}

// matchesText reports whether every word of text fuzzy-matches the bean's title or ID.
func matchesText(b *bean.Bean, text string) bool {
	target := []string{b.DisplayTitle() + " " + b.ID}
	for _, word := range strings.Fields(text) {
		if len(list.DefaultFilter(word, target)) == 0 {
			return false
		}
	}
	return true
}

// withTag returns input with its tag: include tokens replaced by tag:<tag>.
func withTag(input, tag string) string {
	tokens := slices.DeleteFunc(strings.Fields(input), func(t string) bool {
		return strings.HasPrefix(strings.ToLower(t), "tag:")
	})
	return strings.Join(append(tokens, "tag:"+tag), " ")
}

// beanPrefill holds field values for a new bean, taken from the active filter query.
type beanPrefill struct {
	Type, Status, Priority, Size string
	Tags                         []string
}

// prefill returns the query's included values as defaults for a new bean, so it
// matches the filter. If a field is given several times, the last value wins.
func (q query) prefill() beanPrefill {
	last := func(values []string) string {
		if len(values) == 0 {
			return ""
		}
		return values[len(values)-1]
	}
	f := q.Filter
	return beanPrefill{
		Type:     last(f.Type),
		Status:   last(f.Status),
		Priority: last(f.Priority),
		Size:     last(f.Size),
		Tags:     slices.Clone(f.Tags),
	}
}
