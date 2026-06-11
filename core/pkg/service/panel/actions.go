// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

// Handle replaces the panel's name.
func (p RenamePayload) Handle(state Panel) (Panel, error) {
	state.Name = p.Name
	return state, nil
}

// Handle inserts the tab into the leaf at the given path-derived key, at the
// given index. The caller is responsible for choosing a valid index in
// [0, len(leaf.Tabs)] — to append, pass len(leaf.Tabs). When Location is
// present, the target leaf is first split at that location and the tab is
// inserted into the new empty sibling leaf. Returns ErrInvalidPath when the
// target leaf path does not resolve, ErrNotALeaf when it resolves to a split,
// ErrIndexOutOfRange when index is outside [0, len(leaf.Tabs)], or
// ErrInvalidSplitLocation when Location cannot produce a split. On any error
// the returned state is the zero Panel; the dispatch substrate aborts the
// transaction on error, so partial state would not be meaningful.
func (p InsertTabPayload) Handle(state Panel) (Panel, error) {
	targetLeaf := p.TargetLeaf
	if p.Location != nil {
		var err error
		targetLeaf, err = splitLeafForPlacement(&state.Root, p.TargetLeaf, *p.Location)
		if err != nil {
			return Panel{}, err
		}
	}
	leaf, err := walkLeaf(&state.Root, targetLeaf)
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
	return state, nil
}

// Handle removes the tab with the given key. When the containing leaf is left
// empty and has a non-empty sibling under a split, the split is collapsed:
// the sibling subtree takes the parent split's position in the tree. Returns
// ErrTabNotFound when no tab matches the key.
func (p RemoveTabPayload) Handle(state Panel) (Panel, error) {
	if _, ok := removeTab(&state.Root, p.Key); !ok {
		return Panel{}, ErrTabNotFound
	}
	collapseEmptyLeaves(&state.Root)
	return state, nil
}

// Handle moves a tab to a position within the panel. The target leaf is resolved
// before the remove, and empty-leaf collapse is deferred until after the insert,
// so the destination may be a leaf the source's removal would otherwise collapse
// away (e.g. the empty sibling created by a preceding SplitLeaf). When Location
// is present, the target leaf is first split at that location and the tab moves
// into the new empty sibling leaf; moving a leaf's only tab to an edge of its
// own leaf is a no-op (the result would be the tab beside an empty pane).
// Returns ErrTabNotFound when no tab matches the key, ErrInvalidPath /
// ErrNotALeaf when the target path is bad, ErrIndexOutOfRange when index is
// outside [0, len(targetLeaf.Tabs)] after the remove (the count may shrink by
// one when moving within the same leaf), or ErrInvalidSplitLocation when
// Location cannot produce a split.
func (p MoveTabPayload) Handle(state Panel) (Panel, error) {
	targetLeaf := p.TargetLeaf
	if p.Location != nil {
		current, err := walkLeaf(&state.Root, p.TargetLeaf)
		if err != nil {
			return Panel{}, err
		}
		if len(current.Tabs) == 1 && current.Tabs[0].Key == p.Key {
			return state, nil
		}
		targetLeaf, err = splitLeafForPlacement(&state.Root, p.TargetLeaf, *p.Location)
		if err != nil {
			return Panel{}, err
		}
	}
	target, err := walkLeaf(&state.Root, targetLeaf)
	if err != nil {
		return Panel{}, err
	}
	removed, ok := removeTab(&state.Root, p.Key)
	if !ok {
		return Panel{}, ErrTabNotFound
	}
	idx := len(target.Tabs)
	if p.Index != nil {
		idx = int(*p.Index)
	}
	if idx < 0 || idx > len(target.Tabs) {
		return Panel{}, ErrIndexOutOfRange
	}
	target.Tabs = append(target.Tabs[:idx], append([]Tab{removed}, target.Tabs[idx:]...)...)
	collapseEmptyLeaves(&state.Root)
	return state, nil
}

// Handle splits the given leaf into a parent split with two children: the
// original leaf and a new empty leaf. Location determines which side the new
// empty leaf occupies. Size is the fraction allocated to the original leaf;
// defaults to 0.5 when absent. Returns ErrInvalidPath when the leaf path does
// not resolve, ErrNotALeaf when it resolves to a split, or
// ErrInvalidSplitLocation when Location is not one of left/right/top/bottom.
func (p SplitLeafPayload) Handle(state Panel) (Panel, error) {
	size := 0.5
	if p.Size != nil {
		size = *p.Size
	}
	if err := splitLeafAt(&state.Root, p.Leaf, p.Location, size); err != nil {
		return Panel{}, err
	}
	return state, nil
}

// Handle adjusts the size ratio of the split at the given path. Returns
// ErrInvalidPath when the split path does not resolve, or ErrNotASplit when
// it resolves to a leaf.
func (p ResizeSplitPayload) Handle(state Panel) (Panel, error) {
	split, err := walkSplit(&state.Root, p.Split)
	if err != nil {
		return Panel{}, err
	}
	split.Size = p.Size
	return state, nil
}

// Handle sets the visualization resource displayed by the tab with the given key,
// swapping it in place without changing the tab's identity or position. Clears any
// view set on the tab so that exactly one content arm is populated. Returns
// ErrTabNotFound when no tab matches the key.
func (p SetTabResourcePayload) Handle(state Panel) (Panel, error) {
	leaf, _, idx, ok := findTab(&state.Root, p.Key)
	if !ok {
		return Panel{}, ErrTabNotFound
	}
	resource := p.Resource
	leaf.Tabs[idx].Resource = &resource
	leaf.Tabs[idx].View = nil
	return state, nil
}

// Handle sets the inline view displayed by the tab with the given key, swapping it
// in place without changing the tab's identity or position. Clears any resource set
// on the tab so that exactly one content arm is populated. Returns ErrTabNotFound
// when no tab matches the key.
func (p SetTabViewPayload) Handle(state Panel) (Panel, error) {
	leaf, _, idx, ok := findTab(&state.Root, p.Key)
	if !ok {
		return Panel{}, ErrTabNotFound
	}
	view := p.View
	leaf.Tabs[idx].View = &view
	leaf.Tabs[idx].Resource = nil
	return state, nil
}
