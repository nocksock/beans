package layout

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Rect is the area a pane takes on screen.
type Rect struct {
	ID         ID
	X, Y, W, H int
}

// Layout is an arranged tree: every visible pane with its area. The areas
// tile the whole screen.
type Layout struct {
	root  *box
	width int
}

// box is an arranged node.
type box struct {
	rect     Rect
	leaf     bool
	axis     axis
	children []*box
}

// Arrange lays root out on a width x height screen, showing only the panes
// for which visible returns true. The split containing focus never drops it.
func Arrange(root Node, visible func(ID) bool, focus ID, width, height int) Layout {
	n, ok := root.prune(visible)
	if !ok || width <= 0 || height <= 0 {
		return Layout{width: max(width, 0)}
	}
	return Layout{root: place(n, focus, Rect{W: width, H: height}), width: width}
}

func place(n Node, focus ID, r Rect) *box {
	if !n.split {
		r.ID = n.id
		return &box{rect: r, leaf: true}
	}

	total := r.W
	if n.axis == vertical {
		total = r.H
	}
	children := fit(n.children, n.axis, focus, total)
	sizes := distribute(children, total)

	b := &box{rect: r, axis: n.axis}
	offset := 0
	for i, c := range children {
		cr := Rect{X: r.X + offset, Y: r.Y, W: sizes[i], H: r.H}
		if n.axis == vertical {
			cr = Rect{X: r.X, Y: r.Y + offset, W: r.W, H: sizes[i]}
		}
		b.children = append(b.children, place(c, focus, cr))
		offset += sizes[i]
	}
	return b
}

// fit drops children until their minimum sizes fit into total, highest drop
// first (the later one on ties). The child containing focus is never dropped.
func fit(children []Node, a axis, focus ID, total int) []Node {
	children = append([]Node(nil), children...)
	for len(children) > 1 {
		need := 0
		for _, c := range children {
			need += c.minSize(a)
		}
		if need <= total {
			break
		}
		victim := -1
		for i, c := range children {
			if c.contains(focus) {
				continue
			}
			if victim < 0 || c.drop >= children[victim].drop {
				victim = i
			}
		}
		if victim < 0 {
			break
		}
		children = append(children[:victim], children[victim+1:]...)
	}
	return children
}

// distribute splits total among children: fixed ones get their size, the rest
// share what's left by weight within their min/max bounds. An only child
// takes everything; leftover space goes to the last child.
func distribute(children []Node, total int) []int {
	sizes := make([]int, len(children))
	if len(children) == 1 {
		sizes[0] = total
		return sizes
	}

	rest := total
	var pool []int // flexible children not yet clamped
	for i, c := range children {
		if c.fixed > 0 {
			sizes[i] = c.fixed
			rest -= c.fixed
		} else {
			pool = append(pool, i)
		}
	}

	for len(pool) > 0 {
		weights := 0
		for _, i := range pool {
			weights += max(children[i].weight, 1)
		}
		left := rest
		for j, i := range pool {
			if j == len(pool)-1 {
				sizes[i] = left
				break
			}
			sizes[i] = max(rest, 0) * max(children[i].weight, 1) / weights
			left -= sizes[i]
		}

		clamped := clamp(children, sizes, pool, func(c Node, s int) (int, bool) {
			return c.max, c.max > 0 && s > c.max
		})
		if clamped == nil {
			clamped = clamp(children, sizes, pool, func(c Node, s int) (int, bool) {
				return c.min, s < c.min
			})
		}
		if clamped == nil {
			break
		}
		for _, i := range clamped {
			rest -= sizes[i]
		}
		pool = without(pool, clamped)
	}

	used := 0
	for _, s := range sizes {
		used += s
	}
	sizes[len(sizes)-1] += total - used
	for i := range sizes {
		sizes[i] = max(sizes[i], 0)
	}
	return sizes
}

// clamp sets the size of each pooled child violating a bound to that bound,
// returning the indices it changed.
func clamp(children []Node, sizes, pool []int, violates func(Node, int) (int, bool)) []int {
	var changed []int
	for _, i := range pool {
		if bound, ok := violates(children[i], sizes[i]); ok {
			sizes[i] = bound
			changed = append(changed, i)
		}
	}
	return changed
}

func without(pool, remove []int) []int {
	var out []int
	for _, i := range pool {
		keep := true
		for _, r := range remove {
			if i == r {
				keep = false
				break
			}
		}
		if keep {
			out = append(out, i)
		}
	}
	return out
}

// Rects returns the areas of all arranged panes, in tree order.
func (l Layout) Rects() []Rect {
	var out []Rect
	var walk func(*box)
	walk = func(b *box) {
		if b.leaf {
			out = append(out, b.rect)
			return
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	if l.root != nil {
		walk(l.root)
	}
	return out
}

// Rect returns the area of the pane id, if it was arranged.
func (l Layout) Rect(id ID) (Rect, bool) {
	for _, r := range l.Rects() {
		if r.ID == id {
			return r, true
		}
	}
	return Rect{}, false
}

// Render draws the layout. draw must render each pane at exactly its size.
func (l Layout) Render(draw func(Rect) string) string {
	if l.root == nil {
		return ""
	}
	var render func(*box) string
	render = func(b *box) string {
		if b.leaf {
			return draw(b.rect)
		}
		parts := make([]string, len(b.children))
		for i, c := range b.children {
			parts[i] = render(c)
		}
		if b.axis == horizontal {
			return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
		}
		return strings.Join(parts, "\n")
	}
	return render(l.root)
}
