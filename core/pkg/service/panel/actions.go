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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/spatial"
)

// Handle replaces the document with its created state.
func (p CreatePayload) Handle(Panel) (Panel, error) {
	return p.Panel, nil
}

// Handle replaces the panel's name.
func (p RenamePayload) Handle(state Panel) (Panel, error) {
	state.Name = p.Name
	return state, nil
}

// Handle upserts each tab in Tabs, in order, into one destination leaf. A tab
// whose key is already present always has its content refreshed; it keeps its
// current position unless the payload carries an explicit placement (TargetTab,
// TargetLeaf, Location, or Index), in which case it is relocated with the rest. A
// tab whose key is absent is inserted at the resolved destination.
//
// The destination is resolved in priority order: the leaf holding TargetTab, the
// TargetLeaf path key, then the first leaf in traversal order. Both are hints; one
// that no longer resolves to a leaf falls through to the next, so a placement
// invalidated between the gesture and the dispatch still lands the tabs. The
// fallback drops Location too, leaving a leaf the caller never pointed at unsplit.
// Index positions the first tab and the rest follow it; when Index is nil each tab
// appends. When Location is an edge, the resolved leaf is split once at that
// location and every tab is inserted into the new empty sibling leaf; a center
// Location places the tabs directly in the resolved leaf, equivalent to absent.
// The split is deferred to the first tab that lands, so a batch whose every tab is
// skipped leaves no empty pane behind.
//
// A tab that would put a second tab behind the same resource is skipped, as is a
// view whose type already backs a tab when Singleton is set: each may back at most
// one tab per panel, and callers select the existing tab instead. Skipping one tab
// does not stop the others.
//
// Errors when the tree holds no leaf to default to, when Index is outside
// [0, len(leaf.Tabs)], or when Location cannot produce a split. On any error the
// returned state is the zero Panel; the dispatch substrate aborts the transaction
// on error, so partial state would not be meaningful.
func (p InsertTabsPayload) Handle(state Panel) (Panel, error) {
	placementGiven := p.TargetTab != nil || p.TargetLeaf != nil ||
		p.Location != nil || p.Index != nil
	targetLeaf, stale, err := p.resolveTargetLeaf(state.Root)
	if err != nil {
		return Panel{}, err
	}
	var pendingSplit *spatial.Location
	if !stale && p.Location != nil && *p.Location != spatial.LocationCenter {
		pendingSplit = p.Location
	}
	index := p.Index
	changed := false
	for _, tab := range p.Tabs {
		if r, ok := tab.Variant.(ResourceTab); ok {
			if existing, found := findTabByResource(state.Root, r.Resource); found &&
				existing.Key() != tab.Key() {
				continue
			}
		}
		if v, ok := tab.Variant.(ViewTab); ok && p.Singleton != nil && *p.Singleton {
			if existing, found := findTabByType(state.Root, v.Type); found &&
				existing.Key() != tab.Key() {
				continue
			}
		}
		existingLeaf, existingIdx, exists := findTab(state.Root, tab.Key())
		if exists && !placementGiven {
			if err := updateLeafAt(
				&state.Root,
				existingLeaf,
				func(leaf LeafNode) (LeafNode, error) {
					tabs := append([]Tab{}, leaf.Tabs...)
					tabs[existingIdx] = tab
					leaf.Tabs = tabs
					return leaf, nil
				},
			); err != nil {
				return Panel{}, err
			}
			changed = true
			continue
		}
		if pendingSplit != nil {
			targetLeaf, err = splitLeafForPlacement(
				&state.Root,
				targetLeaf,
				*pendingSplit,
			)
			if err != nil {
				return Panel{}, err
			}
			pendingSplit = nil
		}
		if exists {
			if _, ok := removeTab(&state.Root, tab.Key()); !ok {
				return Panel{}, errTabNotFound
			}
		}
		leaf, err := walkLeaf(state.Root, targetLeaf)
		if err != nil {
			return Panel{}, err
		}
		// An index past the leaf's end is a drop position invalidated between the
		// gesture and the dispatch. It clamps to the end.
		at := int32(len(leaf.Tabs))
		if index != nil && *index >= 0 && *index <= at {
			at = *index
		}
		if err := insertTabAt(&state.Root, targetLeaf, tab, at); err != nil {
			return Panel{}, err
		}
		if index != nil {
			index = new(at + 1)
		}
		changed = true
	}
	if !changed {
		return state, nil
	}
	collapseEmptyLeaves(&state.Root)
	return state, nil
}

// resolveTargetLeaf resolves the destination leaf's path-derived key from the
// payload's addressing hints, in priority order: the leaf holding TargetTab, the
// TargetLeaf path key, then the first leaf in traversal order. stale reports that a
// hint was given and no longer resolves, which drops the Location along with it:
// the tabs still land, but a leaf the caller never pointed at is not split. Errors
// only when the tree contains no leaf to default to.
func (p InsertTabsPayload) resolveTargetLeaf(
	root Node,
) (leaf int32, stale bool, err error) {
	if p.TargetTab != nil {
		if leafPath, _, ok := findTab(root, *p.TargetTab); ok {
			return leafPath, false, nil
		}
	}
	if p.TargetLeaf != nil {
		if _, err := walkLeaf(root, *p.TargetLeaf); err == nil {
			return *p.TargetLeaf, false, nil
		}
	}
	leafPath, ok := firstLeafPath(root)
	if !ok {
		return 0, false, errInvalidPath
	}
	return leafPath, p.TargetTab != nil || p.TargetLeaf != nil, nil
}

// Handle removes the tab with the given key. When the containing leaf is left
// empty and has a non-empty sibling under a split, the split is collapsed:
// the sibling subtree takes the parent split's position in the tree. Errors
// when no tab matches the key.
func (p RemoveTabPayload) Handle(state Panel) (Panel, error) {
	if _, ok := removeTab(&state.Root, p.Key); !ok {
		return Panel{}, errTabNotFound
	}
	collapseEmptyLeaves(&state.Root)
	return state, nil
}

// Handle moves a tab to a position within the panel. The target leaf is resolved
// before the remove, and empty-leaf collapse is deferred until after the insert,
// so the destination may be a leaf the source's removal would otherwise collapse
// away (e.g. the empty sibling created by the preceding edge split). When Location
// is an edge, the target leaf is first split at that location and the tab moves
// into the new empty sibling leaf; moving a leaf's only tab to an edge of its
// own leaf is a no-op (the result would be the tab beside an empty pane). A
// center Location places the tab directly in the target leaf, equivalent to
// absent. Index counts the moved tab as still present, so a same-leaf move
// decrements it past the tab's own slot to stay valid after the remove. Errors
// when no tab matches the key, when the target path is bad, when the adjusted
// index is out of range, or when Location cannot produce a split.
func (p MoveTabPayload) Handle(state Panel) (Panel, error) {
	srcLeaf, srcIdx, ok := findTab(state.Root, p.Key)
	if !ok {
		return Panel{}, errTabNotFound
	}
	targetLeaf := p.TargetLeaf
	if p.Location != nil && *p.Location != spatial.LocationCenter {
		current, err := walkLeaf(state.Root, p.TargetLeaf)
		if err != nil {
			return Panel{}, err
		}
		if len(current.Tabs) == 1 && current.Tabs[0].Key() == p.Key {
			return state, nil
		}
		targetLeaf, err = splitLeafForPlacement(&state.Root, p.TargetLeaf, *p.Location)
		if err != nil {
			return Panel{}, err
		}
	}
	if _, err := walkLeaf(state.Root, targetLeaf); err != nil {
		return Panel{}, err
	}
	removed, ok := removeTab(&state.Root, p.Key)
	if !ok {
		return Panel{}, errTabNotFound
	}
	if err := updateLeafAt(
		&state.Root,
		targetLeaf,
		func(leaf LeafNode) (LeafNode, error) {
			idx := len(leaf.Tabs)
			if p.Index != nil {
				idx = int(*p.Index)
				if srcLeaf == targetLeaf && srcIdx < idx {
					idx--
				}
			}
			if idx < 0 || idx > len(leaf.Tabs) {
				return LeafNode{}, errIndexOutOfRange
			}
			tabs := make([]Tab, 0, len(leaf.Tabs)+1)
			tabs = append(tabs, leaf.Tabs[:idx]...)
			tabs = append(tabs, removed)
			tabs = append(tabs, leaf.Tabs[idx:]...)
			leaf.Tabs = tabs
			return leaf, nil
		},
	); err != nil {
		return Panel{}, err
	}
	collapseEmptyLeaves(&state.Root)
	return state, nil
}

// Handle splits the tab with the given key off its leaf into a new sibling
// pane, moving the tab into it. Direction x places the new pane to the right,
// y to the bottom. A leaf holding a single tab is a no-op (the result would be
// the tab beside an empty pane). Errors when no tab matches the key.
func (p SplitTabPayload) Handle(state Panel) (Panel, error) {
	leafPath, _, ok := findTab(state.Root, p.Key)
	if !ok {
		return Panel{}, errTabNotFound
	}
	source, err := walkLeaf(state.Root, leafPath)
	if err != nil {
		return Panel{}, err
	}
	if len(source.Tabs) < 2 {
		return state, nil
	}
	location := spatial.LocationRight
	if p.Direction == spatial.DirectionY {
		location = spatial.LocationBottom
	}
	targetLeaf, err := splitLeafForPlacement(&state.Root, leafPath, location)
	if err != nil {
		return Panel{}, err
	}
	removed, ok := removeTab(&state.Root, p.Key)
	if !ok {
		return Panel{}, errTabNotFound
	}
	if err := insertTabAt(&state.Root, targetLeaf, removed, 0); err != nil {
		return Panel{}, err
	}
	collapseEmptyLeaves(&state.Root)
	return state, nil
}

// Handle adjusts the size ratio of the split at the given path. Errors when
// the split path does not resolve, when it resolves to a leaf, or when size
// is outside [0, 1].
func (p ResizeSplitPayload) Handle(state Panel) (Panel, error) {
	if p.Size < 0 || p.Size > 1 {
		return Panel{}, errInvalidSize
	}
	updated, err := updateAt(
		state.Root,
		pathDirections(p.Split),
		func(n Node) (Node, error) {
			split, ok := n.Variant.(SplitNode)
			if !ok {
				return Node{}, errors.New("node at path is not a split")
			}
			split.Size = p.Size
			return Node{Variant: split}, nil
		},
	)
	if err != nil {
		return Panel{}, err
	}
	state.Root = updated
	return state, nil
}

// Handle swaps the tab's content to the given resource in place, clearing any
// view. A no-op when the resource already backs a different tab in the panel:
// a resource may back at most one tab, and callers select the existing tab
// instead. Errors when no tab matches the key.
func (p SetTabResourcePayload) Handle(state Panel) (Panel, error) {
	if existing, found := findTabByResource(state.Root, p.Resource); found &&
		existing.Key() != p.Key {
		return state, nil
	}
	path, idx, ok := findTab(state.Root, p.Key)
	if !ok {
		return Panel{}, errTabNotFound
	}
	if err := updateLeafAt(&state.Root, path, func(leaf LeafNode) (LeafNode, error) {
		tabs := append([]Tab{}, leaf.Tabs...)
		tabs[idx] = Tab{Variant: ResourceTab{Key: p.Key, Resource: p.Resource}}
		leaf.Tabs = tabs
		return leaf, nil
	}); err != nil {
		return Panel{}, err
	}
	return state, nil
}

// Handle swaps the tab's content to the given inline view in place, clearing
// any resource. Errors when no tab matches the key.
func (p SetTabViewPayload) Handle(state Panel) (Panel, error) {
	path, idx, ok := findTab(state.Root, p.Key)
	if !ok {
		return Panel{}, errTabNotFound
	}
	if err := updateLeafAt(&state.Root, path, func(leaf LeafNode) (LeafNode, error) {
		tabs := append([]Tab{}, leaf.Tabs...)
		tabs[idx] = Tab{Variant: ViewTab{Key: p.Key, View: p.View}}
		leaf.Tabs = tabs
		return leaf, nil
	}); err != nil {
		return Panel{}, err
	}
	return state, nil
}
