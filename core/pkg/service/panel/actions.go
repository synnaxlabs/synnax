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

// Handle upserts Tab into the tree keyed on its tab key. A tab whose key is
// already present always has its content refreshed; it keeps its current
// position unless the payload carries an explicit placement (TargetTab,
// TargetLeaf, Location, or Index), in which case it is relocated. A tab whose
// key is absent is inserted at the resolved destination. Inserting a resource
// tab whose resource already backs a different tab is a no-op: a resource may
// back at most one tab per panel, and callers select the existing tab instead.
// With Singleton set, inserting a view is likewise a no-op when a view of the
// same type already backs a tab in the panel.
//
// The destination is resolved in priority order: the leaf holding TargetTab when
// set, otherwise the TargetLeaf path key, otherwise the first leaf in traversal
// order. Appends when Index is nil; otherwise the caller is responsible for
// choosing a valid index in [0, len(leaf.Tabs)]. When Location is an edge, the
// resolved leaf is first split at that location and the tab is inserted into the
// new empty sibling leaf; a center Location places the tab directly in the
// resolved leaf, equivalent to absent. Empty leaves left under a split are
// collapsed afterwards, so an edge insert into an empty leaf degrades to a
// direct insert. Errors when TargetTab is set but unmatched, when the resolved
// leaf path does not resolve or resolves to a split, when index is outside
// [0, len(leaf.Tabs)], or when Location cannot produce a split. On any error the
// returned state is the zero Panel; the dispatch substrate aborts the
// transaction on error, so partial state would not be meaningful.
func (p InsertTabPayload) Handle(state Panel) (Panel, error) {
	if r, ok := p.Tab.Variant.(TabResource); ok {
		if existing, found := findTabByResource(state.Root, r.Resource); found &&
			existing.Key() != p.Tab.Key() {
			return state, nil
		}
	}
	if v, ok := p.Tab.Variant.(TabView); ok && p.Singleton != nil && *p.Singleton {
		if existing, found := findTabByType(state.Root, v.Type); found &&
			existing.Key() != p.Tab.Key() {
			return state, nil
		}
	}
	existingLeaf, existingIdx, exists := findTab(state.Root, p.Tab.Key())
	placementGiven := p.TargetTab != nil || p.TargetLeaf != nil ||
		p.Location != nil || p.Index != nil
	if exists && !placementGiven {
		if err := updateLeafAt(&state.Root, existingLeaf, func(leaf Leaf) (Leaf, error) {
			tabs := append([]Tab{}, leaf.Tabs...)
			tabs[existingIdx] = p.Tab
			leaf.Tabs = tabs
			return leaf, nil
		}); err != nil {
			return Panel{}, err
		}
		return state, nil
	}
	targetLeaf, err := p.resolveTargetLeaf(state.Root)
	if err != nil {
		return Panel{}, err
	}
	if p.Location != nil && *p.Location != spatial.LocationCenter {
		targetLeaf, err = splitLeafForPlacement(&state.Root, targetLeaf, *p.Location)
		if err != nil {
			return Panel{}, err
		}
	}
	if exists {
		if _, ok := removeTab(&state.Root, p.Tab.Key()); !ok {
			return Panel{}, errTabNotFound
		}
	}
	leaf, err := walkLeaf(state.Root, targetLeaf)
	if err != nil {
		return Panel{}, err
	}
	index := int32(len(leaf.Tabs))
	if p.Index != nil {
		index = *p.Index
	}
	if err := insertTabAt(&state.Root, targetLeaf, p.Tab, index); err != nil {
		return Panel{}, err
	}
	collapseEmptyLeaves(&state.Root)
	return state, nil
}

// resolveTargetLeaf resolves the destination leaf's path-derived key from the
// payload's addressing fields, in priority order: the leaf holding TargetTab
// when set, then the TargetLeaf path key, then the first leaf in traversal
// order. Errors when TargetTab is set but no tab matches it, or when the tree
// contains no leaf to default to.
func (p InsertTabPayload) resolveTargetLeaf(root Node) (int32, error) {
	if p.TargetTab != nil {
		leafPath, _, ok := findTab(root, *p.TargetTab)
		if !ok {
			return 0, errTabNotFound
		}
		return leafPath, nil
	}
	if p.TargetLeaf != nil {
		return *p.TargetLeaf, nil
	}
	leafPath, ok := firstLeafPath(root)
	if !ok {
		return 0, errInvalidPath
	}
	return leafPath, nil
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
	if err := updateLeafAt(&state.Root, targetLeaf, func(leaf Leaf) (Leaf, error) {
		idx := len(leaf.Tabs)
		if p.Index != nil {
			idx = int(*p.Index)
			if srcLeaf == targetLeaf && srcIdx < idx {
				idx--
			}
		}
		if idx < 0 || idx > len(leaf.Tabs) {
			return Leaf{}, errIndexOutOfRange
		}
		tabs := make([]Tab, 0, len(leaf.Tabs)+1)
		tabs = append(tabs, leaf.Tabs[:idx]...)
		tabs = append(tabs, removed)
		tabs = append(tabs, leaf.Tabs[idx:]...)
		leaf.Tabs = tabs
		return leaf, nil
	}); err != nil {
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
	updated, err := updateAt(state.Root, pathDirections(p.Split), func(n Node) (Node, error) {
		split, ok := n.Variant.(NodeSplit)
		if !ok {
			return Node{}, errors.New("node at path is not a split")
		}
		split.Size = p.Size
		return Node{Variant: split}, nil
	})
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
	if err := updateLeafAt(&state.Root, path, func(leaf Leaf) (Leaf, error) {
		tabs := append([]Tab{}, leaf.Tabs...)
		tabs[idx] = Tab{Variant: TabResource{
			TabBase:  TabBase{Key: p.Key},
			Resource: p.Resource,
		}}
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
	if err := updateLeafAt(&state.Root, path, func(leaf Leaf) (Leaf, error) {
		tabs := append([]Tab{}, leaf.Tabs...)
		tabs[idx] = Tab{Variant: TabView{
			TabBase: TabBase{Key: p.Key},
			View:    p.View,
		}}
		leaf.Tabs = tabs
		return leaf, nil
	}); err != nil {
		return Panel{}, err
	}
	return state, nil
}
