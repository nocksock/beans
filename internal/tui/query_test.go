package tui

import (
	"reflect"
	"testing"

	"github.com/hmans/beans/internal/bean"
	"github.com/hmans/beans/internal/config"
	"github.com/hmans/beans/internal/graph/model"
)

func TestParseQuery(t *testing.T) {
	cfg := config.Default()

	tests := []struct {
		name     string
		input    string
		filter   model.BeanFilter
		text     string
		sort     sortSpec
		warnings []string
	}{
		{
			name:  "empty",
			input: "  ",
		},
		{
			name:  "free text only",
			input: "login  page",
			text:  "login page",
		},
		{
			name:   "same field is ORed",
			input:  "size:sm size:lg",
			filter: model.BeanFilter{Size: []string{"sm", "lg"}},
		},
		{
			name:  "different fields and text",
			input: "type:bug fix status:todo crash",
			filter: model.BeanFilter{
				Type:   []string{"bug"},
				Status: []string{"todo"},
			},
			text: "fix crash",
		},
		{
			name:  "excludes",
			input: "-status:completed -size:xl -type:task -priority:low -tag:wip",
			filter: model.BeanFilter{
				ExcludeStatus:   []string{"completed"},
				ExcludeSize:     []string{"xl"},
				ExcludeType:     []string{"task"},
				ExcludePriority: []string{"low"},
				ExcludeTags:     []string{"wip"},
			},
		},
		{
			name:   "tags and case-insensitive field/value",
			input:  "TAG:idea Priority:HIGH",
			filter: model.BeanFilter{Tags: []string{"idea"}, Priority: []string{"high"}},
		},
		{
			name:  "sort with default direction",
			input: "sort:created",
			sort:  sortSpec{Field: "created", Desc: true},
		},
		{
			name:  "sort with explicit direction",
			input: "sort:priority:desc",
			sort:  sortSpec{Field: "priority", Desc: true},
		},
		{
			name:  "last sort wins",
			input: "sort:size sort:created:asc",
			sort:  sortSpec{Field: "created"},
		},
		{
			name:  "incomplete tokens are ignored silently",
			input: "size: sort: -status:",
		},
		{
			name:     "unknown field is a warning and ignored",
			input:    "color:red text",
			text:     "text",
			warnings: []string{`unknown field "color"`},
		},
		{
			name:     "unknown value is a warning and ignored",
			input:    "size:huge size:sm",
			filter:   model.BeanFilter{Size: []string{"sm"}},
			warnings: []string{`unknown size "huge"`},
		},
		{
			name:     "bad sort field and direction",
			input:    "sort:color sort:id:sideways",
			warnings: []string{`unknown sort field "color"`, `unknown sort direction "sideways"`},
		},
		{
			name:  "dash word without colon stays text",
			input: "-foo bar",
			text:  "-foo bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := parseQuery(tt.input, cfg)
			if !reflect.DeepEqual(*q.Filter, tt.filter) {
				t.Errorf("filter = %+v, want %+v", *q.Filter, tt.filter)
			}
			if q.Text != tt.text {
				t.Errorf("text = %q, want %q", q.Text, tt.text)
			}
			if q.Sort != tt.sort {
				t.Errorf("sort = %+v, want %+v", q.Sort, tt.sort)
			}
			if !reflect.DeepEqual(q.Warnings, tt.warnings) {
				t.Errorf("warnings = %q, want %q", q.Warnings, tt.warnings)
			}
		})
	}
}

func TestMatchesText(t *testing.T) {
	b := &bean.Bean{ID: "beans-kumc", Title: "Make the filter more usable"}
	tests := map[string]bool{
		"":              true,
		"filter":        true,
		"fltr usbl":     true,
		"kumc":          true,
		"filter zzz":    false,
		"usable filter": true,
		"nothinglikeit": false,
	}
	for text, want := range tests {
		if got := matchesText(b, text); got != want {
			t.Errorf("matchesText(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestWithTag(t *testing.T) {
	tests := []struct{ input, tag, want string }{
		{"", "idea", "tag:idea"},
		{"size:sm foo", "idea", "size:sm foo tag:idea"},
		{"tag:old size:sm -tag:wip", "idea", "size:sm -tag:wip tag:idea"},
	}
	for _, tt := range tests {
		if got := withTag(tt.input, tt.tag); got != tt.want {
			t.Errorf("withTag(%q, %q) = %q, want %q", tt.input, tt.tag, got, tt.want)
		}
	}
}
