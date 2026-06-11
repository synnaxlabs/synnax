// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";

import { type Node, type Tab } from "@/panel/types.gen";

// Nodes in the panel tree are identified by path-derived numeric keys: the root
// is ROOT_PATH, and a split's children are childPath(key, "first" | "last").
// The scheme is shared with the Go and TypeScript reducers; every consumer that
// addresses nodes (dispatching InsertTab/SplitLeaf, adapting the tree for
// rendering) must use these helpers rather than re-deriving the math.
export const ROOT_PATH = 1;

/** childPath returns the path key of a split's child on the given side. */
export const childPath = (pathKey: number, side: "first" | "last"): number =>
  side === "first" ? pathKey * 2 : pathKey * 2 + 1;

/**
 * splitSide returns the child slot the new empty leaf occupies when a leaf is
 * split at the given location: "first" for left/top, "last" otherwise. The
 * convention is shared by SplitLeaf and location-bearing InsertTab/MoveTab.
 */
export const splitSide = (loc: location.Location): "first" | "last" =>
  loc === "left" || loc === "top" ? "first" : "last";

const pathDirections = (pathKey: number): boolean[] => {
  if (pathKey <= ROOT_PATH) return [];
  const bits: boolean[] = [];
  while (pathKey > ROOT_PATH) {
    bits.unshift((pathKey & 1) === 1);
    pathKey >>= 1;
  }
  return bits;
};

/**
 * walkPath returns the node at the given path key, or null when the path does
 * not exist in the tree.
 */
export const walkPath = <N extends Node>(
  root: N | undefined | null,
  pathKey: number,
): N | null => {
  if (root == null) return null;
  let n: N = root;
  for (const isLast of pathDirections(pathKey)) {
    if (n.split == null) return null;
    const next = isLast ? n.split.last : n.split.first;
    if (next == null) return null;
    n = next as N;
  }
  return n;
};

/** findTab returns the tab with the given key, or null when absent. */
export const findTab = (node: Node | undefined | null, key: string): Tab | null => {
  if (node == null) return null;
  if (node.leaf != null) return node.leaf.tabs.find((t) => t.key === key) ?? null;
  if (node.split != null)
    return findTab(node.split.first, key) ?? findTab(node.split.last, key);
  return null;
};

/** firstTab returns the first tab in traversal order, or null for an empty tree. */
export const firstTab = (node: Node | undefined | null): Tab | null => {
  if (node == null) return null;
  if (node.leaf != null) return node.leaf.tabs[0] ?? null;
  if (node.split != null)
    return firstTab(node.split.first) ?? firstTab(node.split.last);
  return null;
};

const findLeafPath = (
  node: Node | undefined | null,
  path: number,
  match: (tabs: Tab[]) => boolean,
): number | null => {
  if (node == null) return null;
  if (node.leaf != null) return match(node.leaf.tabs) ? path : null;
  if (node.split != null)
    return (
      findLeafPath(node.split.first, childPath(path, "first"), match) ??
      findLeafPath(node.split.last, childPath(path, "last"), match)
    );
  return null;
};

/** tabLeafPath returns the path key of the leaf holding the given tab, or null. */
export const tabLeafPath = (
  root: Node | undefined | null,
  tabKey: string,
): number | null =>
  findLeafPath(root, ROOT_PATH, (tabs) => tabs.some((t) => t.key === tabKey));

/** firstLeafPath returns the path key of the first leaf in traversal order. */
export const firstLeafPath = (root: Node | undefined | null): number | null =>
  findLeafPath(root, ROOT_PATH, () => true);
