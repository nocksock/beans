package beancore

import (
	"testing"

	"github.com/hmans/beans/internal/bean"
)

func TestFindIncomingLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create beans with relationships
	// A -> B (blocks)
	// A -> C (parent)
	// D -> B (blocks)
	beanA := &bean.Bean{
		ID:       "aaa1",
		Title:    "Bean A",
		Status:   "todo",
		Blocking: []string{"bbb2"},
		Parent:   "ccc3",
	}
	beanB := &bean.Bean{ID: "bbb2", Title: "Bean B", Status: "todo"}
	beanC := &bean.Bean{ID: "ccc3", Title: "Bean C", Status: "todo"}
	beanD := &bean.Bean{
		ID:       "ddd4",
		Title:    "Bean D",
		Status:   "todo",
		Blocking: []string{"bbb2"},
	}

	for _, b := range []*bean.Bean{beanA, beanB, beanC, beanD} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("multiple incoming blocks", func(t *testing.T) {
		links := core.FindIncomingLinks("bbb2")
		if len(links) != 2 {
			t.Errorf("FindIncomingLinks(bbb2) = %d links, want 2", len(links))
		}

		// Check both A and D block B
		fromIDs := make(map[string]string)
		for _, link := range links {
			fromIDs[link.FromBean.ID] = link.LinkType
		}
		if fromIDs["aaa1"] != "blocking" {
			t.Error("expected aaa1 -> bbb2 via blocks")
		}
		if fromIDs["ddd4"] != "blocking" {
			t.Error("expected ddd4 -> bbb2 via blocks")
		}
	})

	t.Run("single incoming parent link", func(t *testing.T) {
		links := core.FindIncomingLinks("ccc3")
		if len(links) != 1 {
			t.Errorf("FindIncomingLinks(ccc3) = %d links, want 1", len(links))
		}
		if links[0].FromBean.ID != "aaa1" || links[0].LinkType != "parent" {
			t.Errorf("expected aaa1 -> ccc3 via parent, got %s -> ccc3 via %s", links[0].FromBean.ID, links[0].LinkType)
		}
	})

	t.Run("no incoming links", func(t *testing.T) {
		links := core.FindIncomingLinks("aaa1")
		if len(links) != 0 {
			t.Errorf("FindIncomingLinks(aaa1) = %d links, want 0", len(links))
		}
	})

	t.Run("nonexistent bean", func(t *testing.T) {
		links := core.FindIncomingLinks("nonexistent")
		if len(links) != 0 {
			t.Errorf("FindIncomingLinks(nonexistent) = %d links, want 0", len(links))
		}
	})
}

func TestDetectCycle(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create a chain: A blocks B, B blocks C
	beanA := &bean.Bean{
		ID:       "aaa1",
		Title:    "Bean A",
		Status:   "todo",
		Blocking: []string{"bbb2"},
	}
	beanB := &bean.Bean{
		ID:       "bbb2",
		Title:    "Bean B",
		Status:   "todo",
		Blocking: []string{"ccc3"},
	}
	beanC := &bean.Bean{
		ID:     "ccc3",
		Title:  "Bean C",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB, beanC} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("would create cycle", func(t *testing.T) {
		// Adding C blocks A would create: A -> B -> C -> A
		cycle := core.DetectCycle("ccc3", "blocking", "aaa1")
		if cycle == nil {
			t.Error("DetectCycle should return cycle path when cycle would be created")
		}
		if len(cycle) < 3 {
			t.Errorf("cycle path too short: %v", cycle)
		}
	})

	t.Run("no cycle", func(t *testing.T) {
		// Adding D blocks A would not create a cycle (D doesn't exist in chain)
		beanD := &bean.Bean{ID: "ddd4", Title: "Bean D", Status: "todo"}
		if err := core.Create(beanD); err != nil {
			t.Fatalf("Create error: %v", err)
		}

		cycle := core.DetectCycle("ddd4", "blocking", "aaa1")
		if cycle != nil {
			t.Errorf("DetectCycle should return nil when no cycle, got: %v", cycle)
		}
	})

	t.Run("parent cycle detection", func(t *testing.T) {
		// Create parent chain: X -> Y -> Z (X has parent Y, Y has parent Z)
		beanX := &bean.Bean{
			ID:     "xxx1",
			Title:  "Bean X",
			Status: "todo",
			Parent: "yyy2",
		}
		beanY := &bean.Bean{
			ID:     "yyy2",
			Title:  "Bean Y",
			Status: "todo",
			Parent: "zzz3",
		}
		beanZ := &bean.Bean{
			ID:     "zzz3",
			Title:  "Bean Z",
			Status: "todo",
		}

		for _, b := range []*bean.Bean{beanX, beanY, beanZ} {
			if err := core.Create(b); err != nil {
				t.Fatalf("Create error: %v", err)
			}
		}

		// Adding Z parent of X would create: X -> Y -> Z -> X
		cycle := core.DetectCycle("zzz3", "parent", "xxx1")
		if cycle == nil {
			t.Error("DetectCycle should detect parent cycles")
		}
	})
}

func TestCheckAllLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create beans with various link issues:
	// - Broken parent link to nonexistent bean
	// - Self-reference in blocks
	// - Cycle (A -> B -> A via blocks)
	beanA := &bean.Bean{
		ID:       "aaa1",
		Title:    "Bean A",
		Status:   "todo",
		Blocking: []string{"bbb2", "aaa1"}, // aaa1 is self-reference
		Parent:   "nonexistent",
	}
	beanB := &bean.Bean{
		ID:       "bbb2",
		Title:    "Bean B",
		Status:   "todo",
		Blocking: []string{"aaa1"}, // creates cycle
	}

	for _, b := range []*bean.Bean{beanA, beanB} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	result := core.CheckAllLinks()

	t.Run("detects broken links", func(t *testing.T) {
		if len(result.BrokenLinks) != 1 {
			t.Errorf("BrokenLinks = %d, want 1", len(result.BrokenLinks))
		}
		if len(result.BrokenLinks) > 0 {
			bl := result.BrokenLinks[0]
			if bl.BeanID != "aaa1" || bl.LinkType != "parent" || bl.Target != "nonexistent" {
				t.Errorf("unexpected broken link: %+v", bl)
			}
		}
	})

	t.Run("detects self-references", func(t *testing.T) {
		if len(result.SelfLinks) != 1 {
			t.Errorf("SelfLinks = %d, want 1", len(result.SelfLinks))
		}
		if len(result.SelfLinks) > 0 {
			sl := result.SelfLinks[0]
			if sl.BeanID != "aaa1" || sl.LinkType != "blocking" {
				t.Errorf("unexpected self-link: %+v", sl)
			}
		}
	})

	t.Run("detects cycles", func(t *testing.T) {
		if len(result.Cycles) != 1 {
			t.Errorf("Cycles = %d, want 1", len(result.Cycles))
		}
		if len(result.Cycles) > 0 {
			c := result.Cycles[0]
			if c.LinkType != "blocking" {
				t.Errorf("cycle link type = %q, want 'blocks'", c.LinkType)
			}
			if len(c.Path) < 3 {
				t.Errorf("cycle path too short: %v", c.Path)
			}
		}
	})

	t.Run("HasIssues returns true", func(t *testing.T) {
		if !result.HasIssues() {
			t.Error("HasIssues() should return true")
		}
	})

	t.Run("TotalIssues counts all", func(t *testing.T) {
		if result.TotalIssues() != 3 {
			t.Errorf("TotalIssues() = %d, want 3", result.TotalIssues())
		}
	})
}

func TestCheckAllLinksClean(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create clean beans with no issues
	beanA := &bean.Bean{
		ID:       "aaa1",
		Title:    "Bean A",
		Status:   "todo",
		Blocking: []string{"bbb2"},
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	result := core.CheckAllLinks()

	if result.HasIssues() {
		t.Errorf("HasIssues() should return false for clean beans, got broken=%d self=%d cycles=%d",
			len(result.BrokenLinks), len(result.SelfLinks), len(result.Cycles))
	}
}

func TestRemoveLinksTo(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create beans where multiple beans link to one target
	beanA := &bean.Bean{
		ID:       "aaa1",
		Title:    "Bean A",
		Status:   "todo",
		Blocking: []string{"target"},
		Parent:   "target",
	}
	beanB := &bean.Bean{
		ID:       "bbb2",
		Title:    "Bean B",
		Status:   "todo",
		Blocking: []string{"target"},
	}
	target := &bean.Bean{
		ID:     "target",
		Title:  "Target Bean",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB, target} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	// Remove all links to target
	removed, err := core.RemoveLinksTo("target")
	if err != nil {
		t.Fatalf("RemoveLinksTo error: %v", err)
	}

	if removed != 3 {
		t.Errorf("removed = %d, want 3", removed)
	}

	// Verify links are gone
	loadedA, _ := core.Get("aaa1")
	if loadedA.Parent != "" || len(loadedA.Blocking) != 0 {
		t.Errorf("Bean A still has relationships: parent=%q blocks=%v", loadedA.Parent, loadedA.Blocking)
	}

	loadedB, _ := core.Get("bbb2")
	if len(loadedB.Blocking) != 0 {
		t.Errorf("Bean B still has %d blocks, want 0", len(loadedB.Blocking))
	}
}

func TestFixBrokenLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// Create bean with broken link and self-reference
	beanA := &bean.Bean{
		ID:       "aaa1",
		Title:    "Bean A",
		Status:   "todo",
		Blocking: []string{"bbb2", "aaa1"}, // bbb2 is valid, aaa1 is self-reference
		Parent:   "nonexistent",            // broken
	}
	beanB := &bean.Bean{
		ID:     "bbb2",
		Title:  "Bean B",
		Status: "todo",
	}

	for _, b := range []*bean.Bean{beanA, beanB} {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	// Fix broken links
	fixed, err := core.FixBrokenLinks()
	if err != nil {
		t.Fatalf("FixBrokenLinks error: %v", err)
	}

	if fixed != 2 {
		t.Errorf("fixed = %d, want 2", fixed)
	}

	// Verify only valid link remains
	loadedA, _ := core.Get("aaa1")
	if len(loadedA.Blocking) != 1 {
		t.Errorf("Bean A has %d blocks, want 1", len(loadedA.Blocking))
	}
	if !loadedA.IsBlocking("bbb2") {
		t.Error("valid 'blocks' link should be preserved")
	}
	if loadedA.Parent != "" {
		t.Errorf("broken parent should be removed, got %q", loadedA.Parent)
	}
}

func TestLinkCheckResultMethods(t *testing.T) {
	t.Run("empty result", func(t *testing.T) {
		r := &LinkCheckResult{
			BrokenLinks: []BrokenLink{},
			SelfLinks:   []SelfLink{},
			Cycles:      []Cycle{},
		}
		if r.HasIssues() {
			t.Error("empty result should not have issues")
		}
		if r.TotalIssues() != 0 {
			t.Errorf("TotalIssues() = %d, want 0", r.TotalIssues())
		}
	})

	t.Run("with issues", func(t *testing.T) {
		r := &LinkCheckResult{
			BrokenLinks: []BrokenLink{{BeanID: "a", LinkType: "blocking", Target: "x"}},
			SelfLinks:   []SelfLink{{BeanID: "b", LinkType: "parent"}},
			Cycles:      []Cycle{{LinkType: "blocking", Path: []string{"a", "b", "a"}}},
		}
		if !r.HasIssues() {
			t.Error("result with issues should have issues")
		}
		if r.TotalIssues() != 3 {
			t.Errorf("TotalIssues() = %d, want 3", r.TotalIssues())
		}
	})
}

func TestCanonicalCycleKey(t *testing.T) {
	tests := []struct {
		path []string
		want string
	}{
		{[]string{"a", "b", "c", "a"}, "a->b->c"},
		{[]string{"c", "a", "b", "c"}, "a->b->c"}, // same cycle, different start
		{[]string{"b", "c", "a", "b"}, "a->b->c"}, // same cycle, different start
		{[]string{"x", "y", "x"}, "x->y"},
		{[]string{"a"}, ""},
		{[]string{}, ""},
	}

	for _, tt := range tests {
		got := canonicalCycleKey(tt.path)
		if got != tt.want {
			t.Errorf("canonicalCycleKey(%v) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func idsOf(beans []*bean.Bean) map[string]bool {
	ids := make(map[string]bool)
	for _, b := range beans {
		ids[b.ID] = true
	}
	return ids
}

func TestBlockedByLinks(t *testing.T) {
	core, _ := setupTestCore(t)

	// aaa1 blocking bbb2 (declared on the blocker)
	// bbb2 blocked_by ccc3 (declared on the blocked bean)
	// bbb2 blocked_by aaa1 (duplicate of the blocking link above)
	// ddd4 blocked_by missing (broken link)
	beans := []*bean.Bean{
		{ID: "aaa1", Title: "A", Status: "todo", Blocking: []string{"bbb2"}},
		{ID: "bbb2", Title: "B", Status: "todo", BlockedBy: []string{"ccc3", "aaa1"}},
		{ID: "ccc3", Title: "C", Status: "todo"},
		{ID: "ddd4", Title: "D", Status: "todo", BlockedBy: []string{"missing"}},
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	t.Run("FindBlockers merges both directions without duplicates", func(t *testing.T) {
		got := core.FindBlockers("bbb2")
		ids := idsOf(got)
		if len(got) != 2 || !ids["aaa1"] || !ids["ccc3"] {
			t.Errorf("FindBlockers(bbb2) = %v, want aaa1 and ccc3", ids)
		}
	})

	t.Run("FindBlockers skips broken links", func(t *testing.T) {
		if got := core.FindBlockers("ddd4"); len(got) != 0 {
			t.Errorf("FindBlockers(ddd4) = %v, want none", idsOf(got))
		}
	})

	t.Run("FindBlocked includes incoming blocked_by", func(t *testing.T) {
		if ids := idsOf(core.FindBlocked("ccc3")); len(ids) != 1 || !ids["bbb2"] {
			t.Errorf("FindBlocked(ccc3) = %v, want bbb2", ids)
		}
		if ids := idsOf(core.FindBlocked("aaa1")); len(ids) != 1 || !ids["bbb2"] {
			t.Errorf("FindBlocked(aaa1) = %v, want bbb2", ids)
		}
	})

	t.Run("FindIncomingLinks reports blocked_by", func(t *testing.T) {
		links := core.FindIncomingLinks("ccc3")
		if len(links) != 1 || links[0].LinkType != "blocked_by" || links[0].FromBean.ID != "bbb2" {
			t.Errorf("FindIncomingLinks(ccc3) = %+v, want one blocked_by from bbb2", links)
		}
	})

	t.Run("DetectCycle across mixed link types", func(t *testing.T) {
		tests := []struct {
			name     string
			from     string
			linkType string
			to       string
			want     bool
		}{
			{"bbb2 blocking ccc3 closes ccc3->bbb2", "bbb2", "blocking", "ccc3", true},
			{"ccc3 blocked_by bbb2 closes ccc3->bbb2", "ccc3", "blocked_by", "bbb2", true},
			{"aaa1 blocked_by bbb2 closes aaa1->bbb2", "aaa1", "blocked_by", "bbb2", true},
			{"ccc3 blocked_by aaa1 is fine", "ccc3", "blocked_by", "aaa1", false},
			{"ddd4 blocked_by bbb2 is fine", "ddd4", "blocked_by", "bbb2", false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := core.DetectCycle(tt.from, tt.linkType, tt.to) != nil
				if got != tt.want {
					t.Errorf("DetectCycle(%s, %s, %s) = %v, want %v", tt.from, tt.linkType, tt.to, got, tt.want)
				}
			})
		}
	})

	t.Run("CheckAllLinks reports broken blocked_by", func(t *testing.T) {
		result := core.CheckAllLinks()
		if len(result.BrokenLinks) != 1 {
			t.Fatalf("BrokenLinks = %+v, want 1", result.BrokenLinks)
		}
		bl := result.BrokenLinks[0]
		if bl.BeanID != "ddd4" || bl.LinkType != "blocked_by" || bl.Target != "missing" {
			t.Errorf("BrokenLinks[0] = %+v", bl)
		}
		if len(result.Cycles) != 0 {
			t.Errorf("Cycles = %+v, want none", result.Cycles)
		}
	})

	t.Run("FixBrokenLinks removes broken blocked_by", func(t *testing.T) {
		fixed, err := core.FixBrokenLinks()
		if err != nil {
			t.Fatalf("FixBrokenLinks error: %v", err)
		}
		if fixed != 1 {
			t.Errorf("FixBrokenLinks fixed %d, want 1", fixed)
		}
		d, _ := core.Get("ddd4")
		if len(d.BlockedBy) != 0 {
			t.Errorf("ddd4.BlockedBy = %v, want empty", d.BlockedBy)
		}
	})

	t.Run("RemoveLinksTo removes blocked_by", func(t *testing.T) {
		removed, err := core.RemoveLinksTo("ccc3")
		if err != nil {
			t.Fatalf("RemoveLinksTo error: %v", err)
		}
		if removed != 1 {
			t.Errorf("RemoveLinksTo removed %d, want 1", removed)
		}
		b, _ := core.Get("bbb2")
		if b.IsBlockedBy("ccc3") {
			t.Errorf("bbb2.BlockedBy still contains ccc3: %v", b.BlockedBy)
		}
	})
}

func TestCheckAllLinksMixedBlockingCycle(t *testing.T) {
	core, _ := setupTestCore(t)

	// aaa1 blocking bbb2, aaa1 blocked_by bbb2 → bbb2 blocks aaa1 → cycle
	beans := []*bean.Bean{
		{ID: "aaa1", Title: "A", Status: "todo", Blocking: []string{"bbb2"}, BlockedBy: []string{"bbb2"}},
		{ID: "bbb2", Title: "B", Status: "todo"},
		{ID: "ccc3", Title: "C", Status: "todo", BlockedBy: []string{"ccc3"}},
	}
	for _, b := range beans {
		if err := core.Create(b); err != nil {
			t.Fatalf("Create error: %v", err)
		}
	}

	result := core.CheckAllLinks()
	if len(result.Cycles) != 1 || result.Cycles[0].LinkType != "blocking" {
		t.Errorf("Cycles = %+v, want one blocking cycle", result.Cycles)
	}
	if len(result.SelfLinks) != 1 || result.SelfLinks[0].LinkType != "blocked_by" || result.SelfLinks[0].BeanID != "ccc3" {
		t.Errorf("SelfLinks = %+v, want one blocked_by self link on ccc3", result.SelfLinks)
	}
}
