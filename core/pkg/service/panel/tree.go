// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
	"github.com/synnaxlabs/x/validate"
)

// Path-derived numeric keys for Node positions in the panel tree:
//   - The root is at key 1.
//   - For any node at key k, its first child is at 2k and its last child is at 2k+1.
//
// Path keys are NOT stable across structural changes — splitting a leaf or collapsing
// a split shifts subsequent path keys. Action vectors that reference path keys in
// later actions must compute them against the post-state of the previous action.

const rootPathKey int32 = 1

// ErrInvalidPath is returned when an action references a path-derived key that does
// not resolve to a node in the tree.
var ErrInvalidPath = errors.New("invalid node path")

// ErrNotALeaf is returned when an action requires a leaf at a given path but the
// node at that path is a split.
var ErrNotALeaf = errors.New("node at path is not a leaf")

// ErrNotASplit is returned when an action requires a split at a given path but the
// node at that path is a leaf.
var ErrNotASplit = errors.New("node at path is not a split")

// ErrTabNotFound is returned when an action references a tab key that does not exist
// in the tree.
var ErrTabNotFound = errors.New("tab not found in tree")

// ErrIndexOutOfRange is returned when an InsertTab index exceeds the leaf's current
// tab count.
var ErrIndexOutOfRange = errors.New("index out of range")

// ErrInvalidSize is returned when a split size ratio falls outside [0, 1].
var ErrInvalidSize = errors.Wrap(validate.ErrValidation, "split size must be in [0, 1]")

// ErrNilNode is returned when a node in the tree has no variant set.
var ErrNilNode = errors.Wrap(validate.ErrValidation, "node has no variant")

// ErrDuplicateTab is returned when two tabs in the same tree share a key.
var ErrDuplicateTab = errors.Wrap(validate.ErrValidation, "duplicate tab key in panel tree")

// validateTree checks the structural invariants every persisted panel tree must
// uphold: every node has a variant, split sizes are within [0, 1], and tab keys
// are unique across the tree. Called before any tree is persisted, both for
// caller-provided trees on create and for reduced trees on dispatch.
func validateTree(root Node) error {
	return validateNode(root, set.New[uuid.UUID]())
}

func validateNode(n Node, seen set.Set[uuid.UUID]) error {
	switch v := n.Variant.(type) {
	case NodeLeaf:
		for _, t := range v.Tabs {
			key := t.Key()
			if seen.Contains(key) {
				return ErrDuplicateTab
			}
			seen.Add(key)
		}
		return nil
	case NodeSplit:
		if v.Size < 0 || v.Size > 1 {
			return ErrInvalidSize
		}
		if err := validateNode(v.First, seen); err != nil {
			return err
		}
		return validateNode(v.Last, seen)
	default:
		return ErrNilNode
	}
}

// Key returns the stable identifier of the tab regardless of its content variant.
// Returns uuid.Nil for a Tab with no variant set.
func (t Tab) Key() uuid.UUID {
	switch v := t.Variant.(type) {
	case TabResource:
		return v.Key
	case TabView:
		return v.Key
	case TabEmpty:
		return v.Key
	default:
		return uuid.Nil
	}
}

// pathDirections returns the sequence of child selections (false=first, true=last)
// needed to walk from the root to the node at pathKey. Returns nil for the root.
func pathDirections(pathKey int32) []bool {
	if pathKey <= 1 {
		return nil
	}
	var bits []bool
	for pathKey > 1 {
		bits = append([]bool{pathKey&1 == 1}, bits...)
		pathKey >>= 1
	}
	return bits
}

// walk returns the node at the given path key, or an error if the path does not
// resolve to a node in the tree.
func walk(root Node, pathKey int32) (Node, error) {
	n := root
	for _, isLast := range pathDirections(pathKey) {
		split, ok := n.Variant.(NodeSplit)
		if !ok {
			return Node{}, ErrInvalidPath
		}
		if isLast {
			n = split.Last
		} else {
			n = split.First
		}
	}
	if n.Variant == nil {
		return Node{}, ErrInvalidPath
	}
	return n, nil
}

// updateAt replaces the node at the given path key with f(node), rebuilding the
// spine of the tree from the root down to that node. Returns ErrInvalidPath when
// the path does not resolve.
func updateAt(n Node, dirs []bool, f func(Node) (Node, error)) (Node, error) {
	if n.Variant == nil {
		return Node{}, ErrInvalidPath
	}
	if len(dirs) == 0 {
		return f(n)
	}
	split, ok := n.Variant.(NodeSplit)
	if !ok {
		return Node{}, ErrInvalidPath
	}
	child := split.First
	if dirs[0] {
		child = split.Last
	}
	updated, err := updateAt(child, dirs[1:], f)
	if err != nil {
		return Node{}, err
	}
	if dirs[0] {
		split.Last = updated
	} else {
		split.First = updated
	}
	return Node{Variant: split}, nil
}

// updateLeafAt replaces the leaf at the given path key with f(leaf), returning
// ErrInvalidPath when the path does not resolve or ErrNotALeaf when it resolves
// to a split.
func updateLeafAt(root *Node, pathKey int32, f func(Leaf) (Leaf, error)) error {
	updated, err := updateAt(*root, pathDirections(pathKey), func(n Node) (Node, error) {
		leaf, ok := n.Variant.(NodeLeaf)
		if !ok {
			return Node{}, ErrNotALeaf
		}
		next, err := f(leaf.Leaf)
		if err != nil {
			return Node{}, err
		}
		return Node{Variant: NodeLeaf{Leaf: next}}, nil
	})
	if err != nil {
		return err
	}
	*root = updated
	return nil
}

// walkLeaf returns the leaf at the given path key, or an error if the path resolves
// to a split or does not resolve at all.
func walkLeaf(root Node, pathKey int32) (Leaf, error) {
	n, err := walk(root, pathKey)
	if err != nil {
		return Leaf{}, err
	}
	leaf, ok := n.Variant.(NodeLeaf)
	if !ok {
		return Leaf{}, ErrNotALeaf
	}
	return leaf.Leaf, nil
}

// findTab walks the tree to find the leaf containing the tab with the given key.
// Returns the leaf's path key, the tab's index within the leaf, and ok=true. When
// the tab is not present, returns ok=false.
func findTab(root Node, tabKey uuid.UUID) (leafPath int32, idx int, ok bool) {
	return findTabAt(root, rootPathKey, tabKey)
}

func findTabAt(n Node, path int32, tabKey uuid.UUID) (int32, int, bool) {
	switch v := n.Variant.(type) {
	case NodeLeaf:
		for i, t := range v.Tabs {
			if t.Key() == tabKey {
				return path, i, true
			}
		}
		return 0, 0, false
	case NodeSplit:
		if p, i, ok := findTabAt(v.First, path*2, tabKey); ok {
			return p, i, true
		}
		return findTabAt(v.Last, path*2+1, tabKey)
	default:
		return 0, 0, false
	}
}

// removeTab removes the tab with the given key from the tree, leaving any
// emptied leaf in place. Collapsing empty leaves is the caller's responsibility
// (collapseEmptyLeaves), deferred so that a composed action sequence (e.g.
// SplitLeaf followed by MoveTab) can target a freshly created empty sibling
// before the tree is tidied. Returns the removed tab and ok=true; ok=false when
// the tab is not present.
func removeTab(root *Node, tabKey uuid.UUID) (Tab, bool) {
	path, idx, ok := findTab(*root, tabKey)
	if !ok {
		return Tab{}, false
	}
	var removed Tab
	if err := updateLeafAt(root, path, func(leaf Leaf) (Leaf, error) {
		removed = leaf.Tabs[idx]
		leaf.Tabs = append(append([]Tab{}, leaf.Tabs[:idx]...), leaf.Tabs[idx+1:]...)
		return leaf, nil
	}); err != nil {
		return Tab{}, false
	}
	return removed, true
}

// collapseEmptyLeaves rewrites the tree bottom-up, replacing every split that has
// exactly one empty-leaf side with its surviving sibling subtree.
func collapseEmptyLeaves(root *Node) {
	*root = collapseNode(*root)
}

func collapseNode(n Node) Node {
	split, ok := n.Variant.(NodeSplit)
	if !ok {
		return n
	}
	split.First = collapseNode(split.First)
	split.Last = collapseNode(split.Last)
	if isEmptyLeaf(split.First) && !isEmptyLeaf(split.Last) {
		return split.Last
	}
	if isEmptyLeaf(split.Last) && !isEmptyLeaf(split.First) {
		return split.First
	}
	return Node{Variant: split}
}

// isEmptyLeaf reports whether the node is a leaf with no tabs.
func isEmptyLeaf(n Node) bool {
	leaf, ok := n.Variant.(NodeLeaf)
	return ok && len(leaf.Tabs) == 0
}

// insertTabAt inserts the tab into the leaf at the given path key at the given
// index in [0, len(leaf.Tabs)]. To append, pass len(leaf.Tabs).
func insertTabAt(root *Node, leafPath int32, tab Tab, index int32) error {
	return updateLeafAt(root, leafPath, func(leaf Leaf) (Leaf, error) {
		idx := int(index)
		if idx < 0 || idx > len(leaf.Tabs) {
			return Leaf{}, ErrIndexOutOfRange
		}
		tabs := make([]Tab, 0, len(leaf.Tabs)+1)
		tabs = append(tabs, leaf.Tabs[:idx]...)
		tabs = append(tabs, tab)
		tabs = append(tabs, leaf.Tabs[idx:]...)
		leaf.Tabs = tabs
		return leaf, nil
	})
}

// ErrInvalidSplitLocation is returned when SplitLeaf is given a location that
// cannot produce a binary split (e.g., "center").
var ErrInvalidSplitLocation = errors.New("invalid split location")

// directionAndSideForLocation maps a spatial.Location onto the (direction, side)
// pair that places a new empty leaf on that side of the original. "left"/"right"
// split along the x axis; "top"/"bottom" split along y. The original leaf
// always takes the opposite side; size is the fraction allocated to the
// original. Returns ErrInvalidSplitLocation for locations that do not divide
// the area in two.
func directionAndSideForLocation(loc spatial.Location) (spatial.Direction, spatial.Order, error) {
	switch loc {
	case spatial.LocationLeft:
		return spatial.DirectionX, spatial.OrderFirst, nil
	case spatial.LocationRight:
		return spatial.DirectionX, spatial.OrderLast, nil
	case spatial.LocationTop:
		return spatial.DirectionY, spatial.OrderFirst, nil
	case spatial.LocationBottom:
		return spatial.DirectionY, spatial.OrderLast, nil
	default:
		return "", "", ErrInvalidSplitLocation
	}
}

// splitLeafAt replaces the leaf at leafPath with a split node containing the
// original leaf and a new empty sibling leaf. loc determines which side the
// new empty leaf occupies. Returns ErrInvalidPath when the path does not
// resolve, ErrNotALeaf when it resolves to a split, or ErrInvalidSplitLocation
// when loc is not one of left/right/top/bottom.
func splitLeafAt(root *Node, leafPath int32, loc spatial.Location, size float64) error {
	direction, side, err := directionAndSideForLocation(loc)
	if err != nil {
		return err
	}
	updated, err := updateAt(*root, pathDirections(leafPath), func(n Node) (Node, error) {
		if _, ok := n.Variant.(NodeLeaf); !ok {
			return Node{}, ErrNotALeaf
		}
		empty := Node{Variant: NodeLeaf{Leaf: Leaf{Tabs: []Tab{}}}}
		split := Split{Direction: direction, Size: size}
		if side == spatial.OrderFirst {
			split.First = empty
			split.Last = n
		} else {
			split.First = n
			split.Last = empty
		}
		return Node{Variant: NodeSplit{Split: split}}, nil
	})
	if err != nil {
		return err
	}
	*root = updated
	return nil
}

// splitLeafForPlacement splits the leaf at leafPath at loc and returns the path
// key of the new empty sibling leaf created by the split. Returns the same
// errors as splitLeafAt.
func splitLeafForPlacement(root *Node, leafPath int32, loc spatial.Location) (int32, error) {
	if err := splitLeafAt(root, leafPath, loc, 0.5); err != nil {
		return 0, err
	}
	_, side, err := directionAndSideForLocation(loc)
	if err != nil {
		return 0, err
	}
	if side == spatial.OrderFirst {
		return leafPath * 2, nil
	}
	return leafPath*2 + 1, nil
}
