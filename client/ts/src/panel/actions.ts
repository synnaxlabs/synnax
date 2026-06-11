// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type spatial } from "@synnaxlabs/x";
import { type Draft } from "immer";

import {
  type Action,
  createReduceAll,
  type HandlerResult,
  type Handlers,
} from "@/panel/actions.gen";
import { findTab, walkPath } from "@/panel/tree";
import { type Leaf, type Node, type Tab } from "@/panel/types.gen";

const NO_OP: HandlerResult = { inverse: [], targets: [] };

const walkLeaf = (root: Draft<Node>, pathKey: number): Draft<Leaf> | null => {
  const n = walkPath(root, pathKey);
  if (n == null || n.leaf == null) return null;
  return n.leaf;
};

// removeTab mirrors removeTab in core/pkg/service/panel/tree.go, removing a tab
// and leaving any emptied leaf in place. Collapsing empty leaves is the caller's
// responsibility (collapseEmptyLeaves), deferred so that a composed action
// sequence (e.g. SplitLeaf followed by MoveTab) can target a freshly created
// empty sibling before the tree is tidied.
const removeTab = (n: Draft<Node>, key: string): Tab | null => {
  if (n.leaf != null) {
    const idx = n.leaf.tabs.findIndex((t) => t.key === key);
    if (idx < 0) return null;
    const [removed] = n.leaf.tabs.splice(idx, 1);
    return removed ?? null;
  }
  if (n.split == null) return null;
  const fromFirst = n.split.first != null ? removeTab(n.split.first, key) : null;
  if (fromFirst != null) return fromFirst;
  return n.split.last != null ? removeTab(n.split.last, key) : null;
};

// collapseEmptyLeaves mirrors collapseEmptyLeaves in core/pkg/service/panel/tree.go,
// rewriting the tree bottom-up: every split with exactly one empty-leaf side is
// replaced in place by its surviving sibling subtree.
const collapseEmptyLeaves = (n: Draft<Node>): void => {
  if (n.split == null) return;
  const first = n.split.first;
  const last = n.split.last;
  if (first != null) collapseEmptyLeaves(first);
  if (last != null) collapseEmptyLeaves(last);
  const firstEmpty = first?.leaf != null && first.leaf.tabs.length === 0;
  const lastEmpty = last?.leaf != null && last.leaf.tabs.length === 0;
  let survivor: Draft<Node> | undefined;
  if (firstEmpty && !lastEmpty) survivor = last;
  else if (lastEmpty && !firstEmpty) survivor = first;
  if (survivor == null) return;
  n.leaf = survivor.leaf;
  n.split = survivor.split;
};

// directionAndSideForLocation maps a spatial.Location onto the (direction, side)
// pair that places a new empty leaf on that side of the original. Returns null
// for locations that do not divide the area in two (e.g., "center").
const directionAndSideForLocation = (
  loc: spatial.Location,
): { direction: spatial.Direction; side: spatial.Order } | null => {
  switch (loc) {
    case "left":
      return { direction: "x", side: "first" };
    case "right":
      return { direction: "x", side: "last" };
    case "top":
      return { direction: "y", side: "first" };
    case "bottom":
      return { direction: "y", side: "last" };
    default:
      return null;
  }
};

// splitLeafAt mirrors splitLeafAt + splitLeafForPlacement in
// core/pkg/service/panel/tree.go, replacing the leaf at the given path with a
// split holding the original leaf and a new empty sibling on the given side.
// Returns the path key of the new empty leaf, or null when the path does not
// resolve to a leaf or the location cannot produce a split.
const splitLeafAt = (
  root: Draft<Node>,
  leafPath: number,
  loc: spatial.Location,
  size: number,
): number | null => {
  const node = walkPath(root, leafPath);
  if (node == null || node.leaf == null) return null;
  const ds = directionAndSideForLocation(loc);
  if (ds == null) return null;
  const original = node.leaf;
  const empty: Leaf = { tabs: [] };
  node.leaf = undefined;
  node.split = {
    direction: ds.direction,
    size,
    first: { leaf: ds.side === "first" ? empty : original },
    last: { leaf: ds.side === "first" ? original : empty },
  };
  return ds.side === "first" ? leafPath * 2 : leafPath * 2 + 1;
};

const handlers: Handlers = {
  rename: (state, payload) => {
    state.name = payload.name;
    return { inverse: [], targets: [state.key] };
  },

  insertTab: (state, payload) => {
    if (state.root == null) return NO_OP;
    let targetLeaf = payload.targetLeaf;
    if (payload.location != null) {
      const placed = splitLeafAt(state.root, targetLeaf, payload.location, 0.5);
      if (placed == null) return NO_OP;
      targetLeaf = placed;
    }
    const leaf = walkLeaf(state.root, targetLeaf);
    if (leaf == null) return NO_OP;
    const idx = payload.index ?? leaf.tabs.length;
    if (idx < 0 || idx > leaf.tabs.length) return NO_OP;
    leaf.tabs.splice(idx, 0, payload.tab);
    return { inverse: [], targets: [payload.tab.key] };
  },

  removeTab: (state, payload) => {
    if (state.root == null) return NO_OP;
    const removed = removeTab(state.root, payload.key);
    if (removed == null) return NO_OP;
    collapseEmptyLeaves(state.root);
    return { inverse: [], targets: [payload.key] };
  },

  // The target leaf is resolved before the remove, and empty-leaf collapse is
  // deferred until after the insert, so the destination may be a leaf the
  // source's removal would otherwise collapse away (e.g. the empty sibling
  // created by a preceding SplitLeaf).
  moveTab: (state, payload) => {
    if (state.root == null) return NO_OP;
    let targetLeaf = payload.targetLeaf;
    if (payload.location != null) {
      const current = walkLeaf(state.root, targetLeaf);
      if (current == null) return NO_OP;
      // Moving a leaf's only tab to an edge of its own leaf is degenerate: the
      // result would be the tab beside an empty pane.
      if (current.tabs.length === 1 && current.tabs[0].key === payload.key)
        return NO_OP;
      const placed = splitLeafAt(state.root, targetLeaf, payload.location, 0.5);
      if (placed == null) return NO_OP;
      targetLeaf = placed;
    }
    const target = walkLeaf(state.root, targetLeaf);
    if (target == null) return NO_OP;
    const removed = removeTab(state.root, payload.key);
    if (removed == null) return NO_OP;
    const idx = payload.index ?? target.tabs.length;
    if (idx < 0 || idx > target.tabs.length) target.tabs.push(removed);
    else target.tabs.splice(idx, 0, removed);
    collapseEmptyLeaves(state.root);
    return { inverse: [], targets: [payload.key] };
  },

  splitLeaf: (state, payload) => {
    if (state.root == null) return NO_OP;
    const placed = splitLeafAt(
      state.root,
      payload.leaf,
      payload.location,
      payload.size ?? 0.5,
    );
    if (placed == null) return NO_OP;
    return { inverse: [], targets: [String(payload.leaf)] };
  },

  resizeSplit: (state, payload) => {
    if (state.root == null) return NO_OP;
    const node = walkPath(state.root, payload.split);
    if (node == null || node.split == null) return NO_OP;
    node.split.size = payload.size;
    return { inverse: [], targets: [String(payload.split)] };
  },

  setTabResource: (state, payload) => {
    if (state.root == null) return NO_OP;
    const tab = findTab(state.root, payload.key);
    if (tab == null) return NO_OP;
    tab.resource = payload.resource;
    tab.view = undefined;
    return { inverse: [], targets: [payload.key] };
  },

  setTabView: (state, payload) => {
    if (state.root == null) return NO_OP;
    const tab = findTab(state.root, payload.key);
    if (tab == null) return NO_OP;
    tab.view = payload.view;
    tab.resource = undefined;
    return { inverse: [], targets: [payload.key] };
  },
};

export const reduceAll = createReduceAll(handlers);

// All current panel actions are eligible for the undo stack from the substrate's
// perspective. Inverse vectors are returned empty for Phase 1 so the substrate
// records targets and remote-touched timestamps correctly without effecting
// undoable replay; richer inverses follow once tree-collapse round-tripping is
// implemented.
export const isUndoable = (_action: Action): boolean => true;
