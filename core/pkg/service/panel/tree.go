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
	"github.com/synnaxlabs/x/spatial"
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
func walk(root *Node, pathKey int32) (*Node, error) {
	if root == nil {
		return nil, ErrInvalidPath
	}
	n := root
	for _, isLast := range pathDirections(pathKey) {
		if n.Split == nil {
			return nil, ErrInvalidPath
		}
		if isLast {
			n = n.Split.Last
		} else {
			n = n.Split.First
		}
		if n == nil {
			return nil, ErrInvalidPath
		}
	}
	return n, nil
}

// walkLeaf returns the leaf at the given path key, or an error if the path resolves
// to a split or does not resolve at all.
func walkLeaf(root *Node, pathKey int32) (*Leaf, error) {
	n, err := walk(root, pathKey)
	if err != nil {
		return nil, err
	}
	if n.Leaf == nil {
		return nil, ErrNotALeaf
	}
	return n.Leaf, nil
}

// walkSplit returns the split at the given path key, or an error if the path
// resolves to a leaf or does not resolve at all.
func walkSplit(root *Node, pathKey int32) (*Split, error) {
	n, err := walk(root, pathKey)
	if err != nil {
		return nil, err
	}
	if n.Split == nil {
		return nil, ErrNotASplit
	}
	return n.Split, nil
}

// findTab walks the tree to find the leaf containing the tab with the given key.
// Returns the containing leaf, the leaf's path key, the tab's index within the
// leaf, and ok=true. When the tab is not present, returns ok=false.
func findTab(root *Node, tabKey uuid.UUID) (leaf *Leaf, leafPath int32, idx int, ok bool) {
	return findTabAt(root, rootPathKey, tabKey)
}

func findTabAt(n *Node, path int32, tabKey uuid.UUID) (*Leaf, int32, int, bool) {
	if n == nil {
		return nil, 0, 0, false
	}
	if n.Leaf != nil {
		for i, t := range n.Leaf.Tabs {
			if t.Key == tabKey {
				return n.Leaf, path, i, true
			}
		}
		return nil, 0, 0, false
	}
	if l, p, i, ok := findTabAt(n.Split.First, path*2, tabKey); ok {
		return l, p, i, true
	}
	return findTabAt(n.Split.Last, path*2+1, tabKey)
}

// removeTabReturning removes the tab with the given key from the tree, collapsing
// any leaf that becomes empty into its sibling. Returns the (possibly new) root,
// the removed tab, and ok=true. When the tab is not present, returns ok=false.
func removeTabReturning(root *Node, tabKey uuid.UUID) (*Node, Tab, bool) {
	return removeTabFromNode(root, tabKey)
}

func removeTabFromNode(n *Node, tabKey uuid.UUID) (*Node, Tab, bool) {
	if n == nil {
		return n, Tab{}, false
	}
	if n.Leaf != nil {
		for i, t := range n.Leaf.Tabs {
			if t.Key == tabKey {
				n.Leaf.Tabs = append(n.Leaf.Tabs[:i], n.Leaf.Tabs[i+1:]...)
				return n, t, true
			}
		}
		return n, Tab{}, false
	}
	if next, removed, ok := removeTabFromNode(n.Split.First, tabKey); ok {
		n.Split.First = next
		return collapseIfEmptySide(n), removed, true
	}
	if next, removed, ok := removeTabFromNode(n.Split.Last, tabKey); ok {
		n.Split.Last = next
		return collapseIfEmptySide(n), removed, true
	}
	return n, Tab{}, false
}

// collapseIfEmptySide returns the surviving child when one of a split's children
// is an empty leaf. Otherwise returns the node unchanged. The split must have both
// children populated as nodes; only the leaf-with-zero-tabs condition triggers
// collapse.
func collapseIfEmptySide(n *Node) *Node {
	if n == nil || n.Split == nil {
		return n
	}
	firstEmpty := n.Split.First != nil &&
		n.Split.First.Leaf != nil &&
		len(n.Split.First.Tabs()) == 0
	lastEmpty := n.Split.Last != nil &&
		n.Split.Last.Leaf != nil &&
		len(n.Split.Last.Tabs()) == 0
	if firstEmpty && !lastEmpty {
		return n.Split.Last
	}
	if lastEmpty && !firstEmpty {
		return n.Split.First
	}
	return n
}

// Tabs is a leaf-only convenience returning the leaf's tabs or nil for split
// nodes. Used by tree mutation helpers to avoid repeated nil checks.
func (n *Node) Tabs() []Tab {
	if n == nil || n.Leaf == nil {
		return nil
	}
	return n.Leaf.Tabs
}

// insertTabAt inserts the tab into the leaf at the given path key at the given
// index in [0, len(leaf.Tabs)]. To append, pass len(leaf.Tabs).
func insertTabAt(root *Node, leafPath int32, tab Tab, index int32) error {
	leaf, err := walkLeaf(root, leafPath)
	if err != nil {
		return err
	}
	idx := int(index)
	if idx < 0 || idx > len(leaf.Tabs) {
		return ErrIndexOutOfRange
	}
	leaf.Tabs = append(leaf.Tabs[:idx], append([]Tab{tab}, leaf.Tabs[idx:]...)...)
	return nil
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

// splitLeafAt replaces the leaf at leafPath with a Split node containing the
// original leaf and a new empty sibling leaf. loc determines which side the
// new empty leaf occupies. Mutates root in place; returns ErrInvalidPath when
// the path does not resolve, ErrNotALeaf when it resolves to a split, or
// ErrInvalidSplitLocation when loc is not one of left/right/top/bottom.
func splitLeafAt(root *Node, leafPath int32, loc spatial.Location, size float64) error {
	direction, side, err := directionAndSideForLocation(loc)
	if err != nil {
		return err
	}
	node, err := walk(root, leafPath)
	if err != nil {
		return err
	}
	if node.Leaf == nil {
		return ErrNotALeaf
	}
	original := &Node{Leaf: node.Leaf}
	empty := &Node{Leaf: &Leaf{Tabs: []Tab{}}}
	split := &Split{Direction: direction, Size: size}
	if side == spatial.OrderFirst {
		split.First = empty
		split.Last = original
	} else {
		split.First = original
		split.Last = empty
	}
	node.Leaf = nil
	node.Split = split
	return nil
}
