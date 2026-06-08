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
import { type Leaf, type Node, type Tab } from "@/panel/types.gen";

const NO_OP: HandlerResult = { inverse: [], targets: [] };

// Path-derived numeric keys mirror the Go reducer: root = 1, first child = 2k,
// last child = 2k+1.

const pathDirections = (pathKey: number): boolean[] => {
  if (pathKey <= 1) return [];
  const bits: boolean[] = [];
  while (pathKey > 1) {
    bits.unshift((pathKey & 1) === 1);
    pathKey >>= 1;
  }
  return bits;
};

const walk = (root: Draft<Node> | undefined, pathKey: number): Draft<Node> | null => {
  if (root == null) return null;
  let n: Draft<Node> = root;
  for (const isLast of pathDirections(pathKey)) {
    if (n.split == null) return null;
    const next = isLast ? n.split.last : n.split.first;
    if (next == null) return null;
    n = next;
  }
  return n;
};

const walkLeaf = (root: Draft<Node>, pathKey: number): Draft<Leaf> | null => {
  const n = walk(root, pathKey);
  if (n == null || n.leaf == null) return null;
  return n.leaf;
};

// removeTabFromNode mirrors removeTabFromNode in core/pkg/service/panel/tree.go,
// removing a tab and collapsing the parent split when a leaf side empties out.
const removeTabFromNode = (n: Draft<Node>, key: string): Tab | null => {
  if (n.leaf != null) {
    const idx = n.leaf.tabs.findIndex((t) => t.key === key);
    if (idx < 0) return null;
    const [removed] = n.leaf.tabs.splice(idx, 1);
    return removed ?? null;
  }
  if (n.split == null) return null;
  const first = n.split.first;
  const last = n.split.last;
  if (first != null) {
    const removed = removeTabFromNode(first, key);
    if (removed != null) {
      collapseIfEmptySide(n);
      return removed;
    }
  }
  if (last != null) {
    const removed = removeTabFromNode(last, key);
    if (removed != null) {
      collapseIfEmptySide(n);
      return removed;
    }
  }
  return null;
};

// findTab walks the tree to return the tab with the given key, or null if absent.
const findTab = (n: Draft<Node>, key: string): Draft<Tab> | null => {
  if (n.leaf != null) return n.leaf.tabs.find((t) => t.key === key) ?? null;
  if (n.split == null) return null;
  return (
    (n.split.first != null ? findTab(n.split.first, key) : null) ??
    (n.split.last != null ? findTab(n.split.last, key) : null)
  );
};

// collapseIfEmptySide rewrites n in place: when one side of n's split is an empty
// leaf, n becomes the surviving sibling subtree.
const collapseIfEmptySide = (n: Draft<Node>): void => {
  if (n.split == null) return;
  const first = n.split.first;
  const last = n.split.last;
  const firstEmpty = first?.leaf != null && first.leaf.tabs.length === 0;
  const lastEmpty = last?.leaf != null && last.leaf.tabs.length === 0;
  if (firstEmpty && !lastEmpty && last != null) {
    n.leaf = last.leaf;
    n.split = last.split;
    return;
  }
  if (lastEmpty && !firstEmpty && first != null) {
    n.leaf = first.leaf;
    n.split = first.split;
  }
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

const handlers: Handlers = {
  rename: (state, payload) => {
    state.name = payload.name;
    return { inverse: [], targets: [state.key] };
  },

  insertTab: (state, payload) => {
    if (state.root == null) return NO_OP;
    const leaf = walkLeaf(state.root, payload.targetLeaf);
    if (leaf == null) return NO_OP;
    const idx = payload.index ?? leaf.tabs.length;
    if (idx < 0 || idx > leaf.tabs.length) return NO_OP;
    leaf.tabs.splice(idx, 0, payload.tab);
    return { inverse: [], targets: [payload.tab.key] };
  },

  removeTab: (state, payload) => {
    if (state.root == null) return NO_OP;
    const removed = removeTabFromNode(state.root, payload.key);
    if (removed == null) return NO_OP;
    return { inverse: [], targets: [payload.key] };
  },

  moveTab: (state, payload) => {
    if (state.root == null) return NO_OP;
    const target = walkLeaf(state.root, payload.targetLeaf);
    if (target == null) return NO_OP;
    const removed = removeTabFromNode(state.root, payload.key);
    if (removed == null) return NO_OP;
    const idx = payload.index ?? target.tabs.length;
    if (idx < 0 || idx > target.tabs.length) target.tabs.push(removed);
    else target.tabs.splice(idx, 0, removed);

    return { inverse: [], targets: [payload.key] };
  },

  splitLeaf: (state, payload) => {
    if (state.root == null) return NO_OP;
    const node = walk(state.root, payload.leaf);
    if (node == null || node.leaf == null) return NO_OP;
    const ds = directionAndSideForLocation(payload.location);
    if (ds == null) return NO_OP;
    const original = node.leaf;
    const empty: Leaf = { tabs: [] };
    const firstLeaf = ds.side === "first" ? empty : original;
    const lastLeaf = ds.side === "first" ? original : empty;
    const size = payload.size ?? 0.5;
    node.leaf = undefined;
    node.split = {
      direction: ds.direction,
      size,
      first: { leaf: firstLeaf },
      last: { leaf: lastLeaf },
    };
    return { inverse: [], targets: [String(payload.leaf)] };
  },

  resizeSplit: (state, payload) => {
    if (state.root == null) return NO_OP;
    const node = walk(state.root, payload.split);
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
