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
import {
  findNode,
  findTab,
  findTabByResource,
  firstLeafPath,
  ROOT_NODE_KEY,
  tabLeafPath,
} from "@/panel/tree";
import { type Node, type NodeLeaf, type Panel, type Tab } from "@/panel/types.gen";

const NO_OP: HandlerResult = { inverse: [], targets: [] };

const walkLeaf = (root: Draft<Node>, pathKey: number): Draft<NodeLeaf> | null => {
  const n = findNode(root, pathKey);
  if (n == null || n.variant !== "leaf") return null;
  return n;
};

// replaceNodeAt swaps the node at the given path key for next, assigning through
// the parent split (or the panel root for ROOT_PATH). Returns false when the
// path does not resolve to a node.
const replaceNodeAt = (state: Draft<Panel>, pathKey: number, next: Node): boolean => {
  if (pathKey === ROOT_NODE_KEY) {
    state.root = next;
    return true;
  }
  const parent = findNode(state.root, pathKey >> 1);
  if (parent == null || parent.variant !== "split") return false;
  if ((pathKey & 1) === 1) parent.last = next;
  else parent.first = next;
  return true;
};

// removeTab mirrors removeTab in core/pkg/service/panel/tree.go, removing a tab
// and leaving any emptied leaf in place. Collapsing empty leaves is the caller's
// responsibility (collapseEmptyLeaves), deferred so that a split-then-move
// sequence (MoveTab with an edge location, or SplitTab) can target the freshly
// created empty sibling before the tree is tidied.
const removeTab = (n: Draft<Node>, key: string): Tab | null => {
  if (n.variant === "leaf") {
    const idx = n.tabs.findIndex((t) => t.key === key);
    if (idx < 0) return null;
    const [removed] = n.tabs.splice(idx, 1);
    return removed ?? null;
  }
  return removeTab(n.first, key) ?? removeTab(n.last, key);
};

const isEmptyLeaf = (n: Draft<Node>): boolean =>
  n.variant === "leaf" && n.tabs.length === 0;

// collapseEmptyLeaves mirrors collapseEmptyLeaves in core/pkg/service/panel/tree.go,
// rewriting the tree bottom-up: every split with exactly one empty-leaf side is
// replaced by its surviving sibling subtree.
const collapseEmptyLeaves = (state: Draft<Panel>): void => {
  state.root = collapseNode(state.root);
};

const collapseNode = (n: Draft<Node>): Draft<Node> => {
  if (n.variant !== "split") return n;
  n.first = collapseNode(n.first);
  n.last = collapseNode(n.last);
  const firstEmpty = isEmptyLeaf(n.first);
  const lastEmpty = isEmptyLeaf(n.last);
  if (firstEmpty && !lastEmpty) return n.last;
  if (lastEmpty && !firstEmpty) return n.first;
  return n;
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
  state: Draft<Panel>,
  leafPath: number,
  loc: spatial.Location,
  size: number,
): number | null => {
  const node = findNode(state.root, leafPath);
  if (node == null || node.variant !== "leaf") return null;
  const ds = directionAndSideForLocation(loc);
  if (ds == null) return null;
  const empty: Node = { variant: "leaf", tabs: [] };
  const split: Node = {
    variant: "split",
    direction: ds.direction,
    size,
    first: ds.side === "first" ? empty : node,
    last: ds.side === "first" ? node : empty,
  };
  if (!replaceNodeAt(state, leafPath, split)) return null;
  return ds.side === "first" ? leafPath * 2 : leafPath * 2 + 1;
};

// replaceTab swaps the tab with the given key for next in place, keeping its
// position within its leaf. Returns false when no tab matches the key.
const replaceTab = (n: Draft<Node>, key: string, next: Tab): boolean => {
  if (n.variant === "leaf") {
    const idx = n.tabs.findIndex((t) => t.key === key);
    if (idx < 0) return false;
    n.tabs[idx] = next;
    return true;
  }
  return replaceTab(n.first, key, next) || replaceTab(n.last, key, next);
};

const handlers: Handlers = {
  rename: (state, payload) => {
    state.name = payload.name;
    return { inverse: [], targets: [state.key] };
  },

  // insertTab upserts payload.tab keyed on its tab key. A tab whose key is
  // already in the tree always has its content refreshed; it keeps its current
  // position unless the caller gives an explicit placement (targetTab,
  // targetLeaf, location, or index), in which case it is relocated. A tab whose
  // key is absent is inserted at the resolved destination. Inserting a resource
  // tab whose resource already backs a different tab is a no-op: a resource may
  // back at most one tab per panel, and callers select the existing tab instead.
  insertTab: (state, payload) => {
    const { tab } = payload;
    if (tab.variant === "resource") {
      const existing = findTabByResource(state.root, tab.resource);
      if (existing != null && existing.key !== tab.key) return NO_OP;
    }
    const placementGiven =
      payload.targetTab != null ||
      payload.targetLeaf != null ||
      payload.location != null ||
      payload.index != null;
    const exists = findTab(state.root, tab.key) != null;
    if (exists && !placementGiven) {
      replaceTab(state.root, tab.key, tab);
      return { inverse: [], targets: [tab.key] };
    }
    let targetLeaf: number | undefined;
    if (payload.targetTab != null)
      targetLeaf = tabLeafPath(state.root, payload.targetTab);
    else if (payload.targetLeaf != null) targetLeaf = payload.targetLeaf;
    else targetLeaf = firstLeafPath(state.root);
    if (targetLeaf == null) return NO_OP;
    if (payload.location != null && payload.location !== "center") {
      const placed = splitLeafAt(state, targetLeaf, payload.location, 0.5);
      if (placed == null) return NO_OP;
      targetLeaf = placed;
    }
    if (exists) removeTab(state.root, tab.key);
    const leaf = walkLeaf(state.root, targetLeaf);
    if (leaf == null) return NO_OP;
    const idx = payload.index ?? leaf.tabs.length;
    if (idx < 0 || idx > leaf.tabs.length) return NO_OP;
    leaf.tabs.splice(idx, 0, tab);
    collapseEmptyLeaves(state);
    return { inverse: [], targets: [tab.key] };
  },

  removeTab: (state, payload) => {
    const removed = removeTab(state.root, payload.key);
    if (removed == null) return NO_OP;
    collapseEmptyLeaves(state);
    return { inverse: [], targets: [payload.key] };
  },

  // The target leaf is resolved before the remove, and empty-leaf collapse is
  // deferred until after the insert, so the destination may be a leaf the
  // source's removal would otherwise collapse away (e.g. the empty sibling
  // created by the preceding edge split).
  moveTab: (state, payload) => {
    const srcLeaf = tabLeafPath(state.root, payload.key);
    const srcNode = srcLeaf != null ? findNode(state.root, srcLeaf) : undefined;
    const srcIdx =
      srcNode?.variant === "leaf"
        ? srcNode.tabs.findIndex((t) => t.key === payload.key)
        : -1;
    let targetLeaf = payload.targetLeaf;
    if (payload.location != null && payload.location !== "center") {
      const current = walkLeaf(state.root, targetLeaf);
      if (current == null) return NO_OP;
      // Moving a leaf's only tab to an edge of its own leaf is degenerate: the
      // result would be the tab beside an empty pane.
      if (current.tabs.length === 1 && current.tabs[0].key === payload.key)
        return NO_OP;
      const placed = splitLeafAt(state, targetLeaf, payload.location, 0.5);
      if (placed == null) return NO_OP;
      targetLeaf = placed;
    }
    const target = walkLeaf(state.root, targetLeaf);
    if (target == null) return NO_OP;
    const removed = removeTab(state.root, payload.key);
    if (removed == null) return NO_OP;
    let idx = payload.index ?? target.tabs.length;
    // The index counts the moved tab as still present, so a same-leaf move
    // decrements it past the tab's own slot to stay valid after the remove.
    if (payload.index != null && srcLeaf === targetLeaf && srcIdx < idx) idx--;
    if (idx < 0 || idx > target.tabs.length) target.tabs.push(removed);
    else target.tabs.splice(idx, 0, removed);
    collapseEmptyLeaves(state);
    return { inverse: [], targets: [payload.key] };
  },

  // splitTab resolves the tab's own leaf, splits it on the direction-mapped
  // edge (x -> right, y -> bottom), and moves the tab into the new sibling
  // pane. A leaf holding a single tab is a no-op (the result would be the tab
  // beside an empty pane).
  splitTab: (state, payload) => {
    const leafPath = tabLeafPath(state.root, payload.key);
    if (leafPath == null) return NO_OP;
    const source = walkLeaf(state.root, leafPath);
    if (source == null || source.tabs.length < 2) return NO_OP;
    const location = payload.direction === "y" ? "bottom" : "right";
    const placed = splitLeafAt(state, leafPath, location, 0.5);
    if (placed == null) return NO_OP;
    const removed = removeTab(state.root, payload.key);
    if (removed == null) return NO_OP;
    const target = walkLeaf(state.root, placed);
    if (target == null) return NO_OP;
    target.tabs.splice(0, 0, removed);
    collapseEmptyLeaves(state);
    return { inverse: [], targets: [payload.key] };
  },

  resizeSplit: (state, payload) => {
    const node = findNode(state.root, payload.split);
    if (node == null || node.variant !== "split") return NO_OP;
    // An equal size must not touch the draft: the dispatch substrate detects
    // no-op vectors by reference equality and skips the server send.
    if (node.size === payload.size) return NO_OP;
    node.size = payload.size;
    return { inverse: [], targets: [String(payload.split)] };
  },

  // setTabResource swaps the tab's content to the resource in place, clearing
  // any view. A no-op when the resource already backs a different tab: a
  // resource may back at most one tab per panel, and callers select the
  // existing tab instead.
  setTabResource: (state, payload) => {
    const existing = findTabByResource(state.root, payload.resource);
    if (existing != null && existing.key !== payload.key) return NO_OP;
    const next: Tab = {
      variant: "resource",
      key: payload.key,
      resource: payload.resource,
    };
    if (!replaceTab(state.root, payload.key, next)) return NO_OP;
    return { inverse: [], targets: [payload.key] };
  },

  // setTabView swaps the tab's content to the inline view in place, clearing
  // any resource.
  setTabView: (state, payload) => {
    const next: Tab = { variant: "view", key: payload.key, ...payload.view };
    if (!replaceTab(state.root, payload.key, next)) return NO_OP;
    return { inverse: [], targets: [payload.key] };
  },
};

export const reduceAll = createReduceAll(handlers);

export const kindOf = (actions: Action[]): string => {
  if (actions.length === 0) return "default";
  // Drag-resize streams ResizeSplit; coalesce them into a single undoable per
  // gesture so one ⌘Z reverses the entire drag.
  if (actions.every((a) => a.type === "resize_split")) return "resize";
  // Same for cross-leaf drags that produce a stream of MoveTab.
  if (actions.every((a) => a.type === "move_tab")) return "move";
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};
