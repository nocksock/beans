// Package layout arranges panes on screen from a declarative split tree.
//
// A tree is built from leaves (one per pane) nested in horizontal and
// vertical splits:
//
//	VSplit(
//		HSplit(Leaf("list").Min(40), Leaf("preview").Max(80)),
//		Leaf("log").Fixed(8),
//	)
//
// Arrange lays the tree out for a set of visible panes: hidden leaves (and
// splits left empty) collapse, and their siblings take the space. When the
// children of a split don't fit their minimum sizes, children are dropped,
// highest Drop first, never the one containing the focused pane.
package layout

import "slices"

// ID identifies a pane (a leaf of the tree).
type ID string

// defaultMin is the minimum size of a leaf: a border on each side plus one
// line or column of content.
const defaultMin = 3

type axis int

const (
	horizontal axis = iota // children side by side
	vertical               // children stacked
)

// Node is a leaf or a split of a layout tree. Its sizing options apply along
// the axis of the split containing it.
type Node struct {
	id       ID
	split    bool
	axis     axis
	children []Node

	fixed  int // exact size, if > 0
	weight int // share of the space left after fixed siblings
	min    int
	max    int // 0 = unbounded
	drop   int // children with a higher drop are dropped first
}

// Leaf returns the node of a single pane.
func Leaf(id ID) Node {
	return Node{id: id, weight: 1, min: defaultMin}
}

// HSplit lays its children out side by side.
func HSplit(children ...Node) Node {
	return Node{split: true, axis: horizontal, children: children, weight: 1}
}

// VSplit stacks its children.
func VSplit(children ...Node) Node {
	return Node{split: true, axis: vertical, children: children, weight: 1}
}

// Fixed gives the node exactly size cells.
func (n Node) Fixed(size int) Node { n.fixed = size; n.min = size; return n }

// Weight sets the node's share of the space left after fixed siblings (default 1).
func (n Node) Weight(w int) Node { n.weight = w; return n }

// Min sets the node's minimum size. Siblings are dropped to make room for it.
func (n Node) Min(size int) Node { n.min = size; return n }

// Max caps the node's size while it has siblings to take the rest.
func (n Node) Max(size int) Node { n.max = size; return n }

// Drop sets the order in which the node is dropped when its split runs out of
// space: higher first (default 0, dropped last).
func (n Node) Drop(order int) Node { n.drop = order; return n }

// contains reports whether id is a leaf of n.
func (n Node) contains(id ID) bool {
	if !n.split {
		return n.id == id
	}
	return slices.ContainsFunc(n.children, func(c Node) bool { return c.contains(id) })
}

// minSize is the size n needs along a, the axis of the split containing it:
// its own minimum, or more if its children need it.
func (n Node) minSize(a axis) int {
	return max(n.min, n.contentMin(a))
}

// contentMin is the size n's content needs along a. A node's own minimum only
// counts along its parent's axis, so across a split only the children's
// content counts.
func (n Node) contentMin(a axis) int {
	if !n.split {
		return defaultMin
	}
	need := 0
	for _, c := range n.children {
		if n.axis == a {
			need += c.minSize(a)
		} else {
			need = max(need, c.contentMin(a))
		}
	}
	return need
}

// prune removes hidden leaves and splits left without children. ok is false
// if nothing of n is visible.
func (n Node) prune(visible func(ID) bool) (_ Node, ok bool) {
	if !n.split {
		return n, visible(n.id)
	}
	var children []Node
	for _, c := range n.children {
		if c, ok := c.prune(visible); ok {
			children = append(children, c)
		}
	}
	n.children = children
	return n, len(children) > 0
}
