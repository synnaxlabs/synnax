// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location, type spatial } from "@synnaxlabs/x";

import { type ontology } from "@/ontology";
import { type Node, type NodeLeaf, type Tab } from "@/panel/types.gen";

// Nodes in the panel tree are identified by path-derived numeric keys: the root
// is ROOT_NODE_KEY, and a split's children are childNodeKey(key, "first" | "last").
// The scheme is shared with the Go and TypeScript reducers; every consumer that
// addresses nodes (dispatching InsertTab/MoveTab, adapting the tree for
// rendering) must use these helpers rather than re-deriving the math.
export const ROOT_NODE_KEY = 1;

/** childNodeKey returns the path key of a split's child on the given side. */
export const childNodeKey = (pathKey: number, side: spatial.Order): number =>
  side === "first" ? pathKey * 2 : pathKey * 2 + 1;

/**
 * splitSide returns the child slot the new empty leaf occupies when a leaf is
 * split at the given location: "first" for left/top, "last" otherwise. The
 * convention is shared by location-bearing InsertTab/MoveTab and SplitTab.
 */
export const splitSide = (loc: location.Outer): spatial.Order =>
  loc === "left" || loc === "top" ? "first" : "last";

const pathDirections = (pathKey: number): boolean[] => {
  if (pathKey <= ROOT_NODE_KEY) return [];
  const bits: boolean[] = [];
  while (pathKey > ROOT_NODE_KEY) {
    bits.unshift((pathKey & 1) === 1);
    pathKey >>= 1;
  }
  return bits;
};

/**
 * findNode returns the node at the given path key, or null when the path does
 * not exist in the tree.
 */
export const findNode = (root: Node | undefined, pathKey: number): Node | undefined => {
  if (root == null) return undefined;
  let n: Node = root;
  for (const isLast of pathDirections(pathKey)) {
    if (n.variant !== "split") return undefined;
    n = isLast ? n.last : n.first;
  }
  return n;
};

/** findTab returns the tab with the given key, or null when absent. */
export const findTab = (node: Node | undefined, key?: string): Tab | undefined => {
  if (node == null || key == null) return undefined;
  if (node.variant === "leaf") return node.tabs.find((t) => t.key === key);
  return findTab(node.first, key) ?? findTab(node.last, key);
};

/**
 * findTabByResource returns the tab whose resource variant displays the given
 * ontology ID, or null when no tab backs that resource. A resource may back at
 * most one tab per panel, so openers use this to select the existing tab
 * instead of inserting a duplicate.
 */
export const findTabByResource = (
  node: Node | undefined,
  resource: ontology.ID,
): Tab | undefined => {
  if (node == null) return undefined;
  if (node.variant === "leaf")
    return node.tabs.find(
      (t) =>
        t.variant === "resource" &&
        t.resource.type === resource.type &&
        t.resource.key === resource.key,
    );
  return (
    findTabByResource(node.first, resource) ?? findTabByResource(node.last, resource)
  );
};

/**
 * findTabByType returns the tab whose view variant carries the given type, or
 * null when no view of that type is present. Openers of singleton views use this
 * to select the existing tab instead of inserting a duplicate.
 */
export const findTabByType = (
  node: Node | undefined,
  type: string,
): Tab | undefined => {
  if (node == null) return undefined;
  if (node.variant === "leaf")
    return node.tabs.find((t) => t.variant === "view" && t.type === type);
  return findTabByType(node.first, type) ?? findTabByType(node.last, type);
};

/** firstTab returns the first tab in traversal order, or null for an empty tree. */
export const firstTab = (node: Node | undefined): Tab | undefined => {
  if (node == null) return undefined;
  if (node.variant === "leaf") return node.tabs[0];
  return firstTab(node.first) ?? firstTab(node.last);
};

const findLeafPath = (
  node: Node | undefined,
  path: number,
  match: (tabs: Tab[]) => boolean,
): number | undefined => {
  if (node == null) return undefined;
  if (node.variant === "leaf") return match(node.tabs) ? path : undefined;
  return (
    findLeafPath(node.first, childNodeKey(path, "first"), match) ??
    findLeafPath(node.last, childNodeKey(path, "last"), match)
  );
};

/** tabLeafPath returns the path key of the leaf holding the given tab, or null. */
export const tabLeafPath = (
  root: Node | undefined,
  tabKey: string,
): number | undefined =>
  findLeafPath(root, ROOT_NODE_KEY, (tabs) => tabs.some((t) => t.key === tabKey));

/** findTabLeaf returns the leaf node holding the given tab, or null when absent. */
export const findTabLeaf = (
  root: Node | undefined,
  tabKey: string,
): NodeLeaf | undefined => {
  const leafPath = tabLeafPath(root, tabKey);
  if (leafPath == null) return undefined;
  const leaf = findNode(root, leafPath);
  return leaf?.variant === "leaf" ? leaf : undefined;
};

/**
 * canSplitTab reports whether the tab can be split off into its own pane: true
 * when the tab shares its leaf with at least one other tab. Splitting the only
 * tab in a leaf is a no-op (the SplitTab action rejects it), so menu
 * affordances for the split should gate on this.
 */
export const canSplitTab = (root: Node | undefined, tabKey: string): boolean => {
  const leafPath = tabLeafPath(root, tabKey);
  if (leafPath == null) return false;
  const leaf = findNode(root, leafPath);
  return leaf?.variant === "leaf" && leaf.tabs.length >= 2;
};

/** firstLeafPath returns the path key of the first leaf in traversal order. */
export const firstLeafPath = (root: Node | undefined): number | undefined =>
  findLeafPath(root, ROOT_NODE_KEY, () => true);
