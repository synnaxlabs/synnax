// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

// Handle replaces the panel's name. When the panel is owned by a user (draft),
// the writer is responsible for promoting it to project ownership in the same
// transaction; the reducer itself only updates the name.
func (p RenamePayload) Handle(state Panel) (Panel, error) {
	state.Name = p.Name
	return state, nil
}

// Handle inserts the tab into the leaf at the given path-derived key, at the
// given index. The caller is responsible for choosing a valid index in
// [0, len(leaf.Tabs)] — to append, pass len(leaf.Tabs). Returns ErrInvalidPath
// when the target leaf path does not resolve, ErrNotALeaf when it resolves to a
// split, or ErrIndexOutOfRange when index is outside [0, len(leaf.Tabs)]. On any
// error the returned state is the zero Panel; the dispatch substrate aborts the
// transaction on error, so partial state would not be meaningful.
func (p InsertTabPayload) Handle(state Panel) (Panel, error) {
	leaf, err := walkLeaf(&state.Root, p.TargetLeaf)
	if err != nil {
		return Panel{}, err
	}
	index := int32(len(leaf.Tabs))
	if p.Index != nil {
		index = *p.Index
	}
	if err := insertTabAt(&state.Root, p.TargetLeaf, p.Tab, index); err != nil {
		return Panel{}, err
	}
	return state, nil
}

// Handle removes the tab with the given key. When the containing leaf is left
// empty and has a non-empty sibling under a split, the split is collapsed:
// the sibling subtree takes the parent split's position in the tree. Returns
// ErrTabNotFound when no tab matches the key.
func (p RemoveTabPayload) Handle(state Panel) (Panel, error) {
	next, _, ok := removeTabReturning(&state.Root, p.Key)
	if !ok {
		return Panel{}, ErrTabNotFound
	}
	state.Root = *next
	return state, nil
}

// Handle moves a tab to a position within the panel. The target leaf is resolved
// before the remove so that a source-leaf collapse (which shifts path keys) does
// not invalidate the destination reference. Returns ErrTabNotFound when no tab
// matches the key, ErrInvalidPath / ErrNotALeaf when the target path is bad, or
// ErrIndexOutOfRange when index is outside [0, len(targetLeaf.Tabs)] after the
// remove (the count may shrink by one when moving within the same leaf).
func (p MoveTabPayload) Handle(state Panel) (Panel, error) {
	target, err := walkLeaf(&state.Root, p.TargetLeaf)
	if err != nil {
		return Panel{}, err
	}
	next, removed, ok := removeTabReturning(&state.Root, p.Key)
	if !ok {
		return Panel{}, ErrTabNotFound
	}
	state.Root = *next
	idx := len(target.Tabs)
	if p.Index != nil {
		idx = int(*p.Index)
	}
	if idx < 0 || idx > len(target.Tabs) {
		return Panel{}, ErrIndexOutOfRange
	}
	target.Tabs = append(target.Tabs[:idx], append([]Tab{removed}, target.Tabs[idx:]...)...)
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
