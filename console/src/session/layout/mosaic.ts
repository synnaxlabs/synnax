// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction, type location, type spatial } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * tabZ is the persisted schema for a tab rendered in a layout mosaic. The shape
 * is part of the persisted slice state and must not change without a migration.
 */
export const tabZ = z.object({
  tabKey: z.string(),
  name: z.string(),
  closable: z.boolean().optional(),
  icon: z.unknown().optional(),
  editable: z.boolean().optional(),
  visible: z.boolean().optional(),
  unsavedChanges: z.boolean().optional(),
  loading: z.boolean().optional(),
  content: z.unknown().optional(),
});

export interface Tab extends z.infer<typeof tabZ> {}

/**
 * Base interface for a mosaic node in the tree. Used to make sure that zod type
 * inference works correctly with recursive types.
 */
interface BaseNode {
  key: number;
  tabs?: Tab[];
  selected?: string;
  direction?: direction.Direction;
  size?: number;
  first?: BaseNode;
  last?: BaseNode;
}

/**
 * nodeZ is the persisted schema for a node in the layout mosaic binary tree. See
 * the {@link Node} interface for the field semantics.
 */
export const nodeZ: z.ZodType<BaseNode> = z.object({
  key: z.number(),
  tabs: z.array(tabZ).optional(),
  selected: z.string().optional(),
  direction: direction.directionZ.optional(),
  size: z.number().optional(),
  get first() {
    return nodeZ.optional();
  },
  get last() {
    return nodeZ.optional();
  },
});

/**
 * Represents the data for a node in the mosaic binary tree. Nodes can be either
 * leaf or non-leaf nodes. Leaf nodes represent the edges of the tree, and render
 * tabs in the mosaic. Non-leaf nodes represent the internal nodes of the tree,
 * and manage the splitting/sizing of the tree.
 */
export interface Node {
  /**
   * Key assigns a unique identifier to the leaf. This key is used to identify the
   * location of the leaf in the tree. Keys are organized in a structure resembling
   * the following.
   *                   1
   *                 /   \
   *                 2    3
   *               /  \  / \
   *              4   5  6  7
   *
   * The first child of a node is assigned the key 2 * key, the second child is
   * assigned the key 2 * key + 1. The root node is assigned the key 1.
   */
  key: number;
  /**
   * A list of tabs for the node to render. If this value is defined, the node is
   * considered a leaf, and the tabs will be rendered.
   */
  tabs?: Tab[];
  /** The key of the selected tab. This value only needs to be set for leaf nodes. */
  selected?: string;
  /** The direction of the split. This value only needs to be set for non-leaf nodes. */
  direction?: direction.Direction;
  /**
   * The first child of the node. If the node is vertical, this is the top child.
   * If the node is horizontal, this is the left child. This value only needs to be
   * set for non-leaf nodes.
   */
  first?: Node;
  /**
   * The last child of the node. If the node is vertical, this is the bottom child.
   * If the node is horizontal, this is the right child. This value only needs to be
   * set for non-leaf nodes.
   */
  last?: Node;
  /**
   * The size of the node as a decimal between 0 and 1. This value only needs to be
   * set for non-leaf nodes.
   */
  size?: number;
}

/**
 * Checks if the selected tab key exists in the tabs array. If it does not, it
 * returns the last tab key in the array. If the array is empty, it returns
 * undefined. This function is useful for 'resetting' the selected tab when a tab
 * is removed that may be the currently selected tab.
 */
const resetSelection = (selected = "", tabs: Tab[] = []): string | undefined => {
  if (tabs.length === 0) return undefined;
  return tabs.find((t) => t.tabKey === selected) != null
    ? selected
    : tabs[tabs.length - 1]?.tabKey;
};

/**
 * Inserts a tab into a node in the mosaic. If the given key is not found, the tab
 * is inserted into the closest ancestor. This is to deal with mosaic garbage
 * collection.
 *
 * @param key - The key of the node to insert the tab into.
 * @param tab - The tab to insert.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 */
export const insertTab = (
  root: Node,
  tab: Tab,
  loc: location.Location = "center",
  key?: number,
  index?: number,
): Node => {
  root = shallowCopyNode(root);
  if (key === undefined) return insertAnywhere(root, tab);

  const node = findNodeOrAncestor(root, key);

  // In the case where we're dropping the node in the center,
  // simply add the tab at the specified index or append it
  if (loc === "center") {
    node.tabs ||= [];
    if (index !== undefined && index >= 0 && index <= node.tabs.length)
      node.tabs.splice(index, 0, tab);
    else node.tabs.push(tab);

    node.selected = tab.tabKey;
    return root;
  }

  const firstChildKey = node.key * 2;
  const lastChildKey = node.key * 2 + 1;
  const potentialChildKey =
    loc === "top" || loc === "left" ? firstChildKey : lastChildKey;

  // Allow for inserting into one of the existing children.
  if (findNodeOrAncestor(root, potentialChildKey).key !== node.key)
    return insertTab(root, tab, "center", potentialChildKey);

  // If we're not dropping the tab in the center, and we have no tabs in the current
  // node, we can't split the node, so we instead insert the tab in the center
  if (node.tabs == null || node.tabs.length === 0)
    return insertTab(root, tab, "center", key);

  const [insertOrder, siblingOrder, dir] = splitArrangement(loc);
  node.direction = dir;

  node[insertOrder] = { key: 0, tabs: [tab], selected: tab.tabKey };
  node[siblingOrder] = { key: 0, tabs: node.tabs, selected: node.selected };

  if (node.first == null || node.last == null) throw new Error("Invalid Mosaic");

  // Assigning these keeps the mosaic sorted so we can do ancestor searches.
  node.first.key = firstChildKey;
  node.last.key = lastChildKey;

  // Clear the previous node, as it's now been split and is not used
  // for rendering.
  node.tabs = undefined;
  node.size = undefined;
  node.selected = undefined;

  return root;
};

export const updateTab = (
  root: Node,
  tabKey: string,
  updater: (tab: Tab) => Tab,
): Node => {
  root = shallowCopyNode(root);
  const [tab, node] = findTab(root, tabKey);
  if (tab == null || node == null) throw new Error("Tab not found");
  node.tabs = node.tabs?.map((t) => (t.tabKey === tabKey ? updater(t) : t));
  return root;
};

const insertAnywhere = (root: Node, tab: Tab): Node => {
  root = shallowCopyNode(root);
  if (root.tabs != null) {
    root.tabs.push(tab);
    root.selected = tab.tabKey;
    return root;
  }
  if (root.first != null) root.first = insertAnywhere(root.first, tab);
  else if (root.last != null) root.last = insertAnywhere(root.last, tab);
  return root;
};

/**
 * Removes a tab from the mosaic and performs any necessary garbage collection.
 *
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to remove. This tab must exist in the mosaic.
 */
export const removeTab = (root: Node, tabKey: string): [Node, string | null] => {
  root = shallowCopyNode(root);
  const [, node] = findTab(root, tabKey);
  if (node == null) return [root, null];
  node.tabs = node.tabs?.filter((t) => t.tabKey !== tabKey);
  node.selected = resetSelection(node.selected, node.tabs);
  root = gc(root);
  const selected = node.selected ?? findSelected(root);
  return [root, selected];
};

const findSelected = (root: Node): string | null => {
  if (root.selected != null) return root.selected;
  if (root.first != null) return findSelected(root.first);
  if (root.last != null) return findSelected(root.last);
  return null;
};

/**
 * Marks the given tab as selected.
 *
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to select. This tab must exist in the mosaic.
 * @returns A shallow copy of the root of the mosaic with the tab selected.
 */
export const selectTab = (root: Node, tabKey: string): Node => {
  root = shallowCopyNode(root);
  const [tab, entry] = findTab(root, tabKey);
  if (tab == null || entry == null) throw new Error("Tab not found");
  entry.selected = tabKey;
  return root;
};

/**
 * Moves a tab from one node to another.
 *
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to move. This tab must exist in the mosaic.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 * @param to - The key of the node to move the tab to.
 * @param index - Optional index where to insert the tab in the target node.
 * @returns A shallow copy of the root of the mosaic with the tab moved.
 */
export const moveTab = (
  root: Node,
  tabKey: string,
  loc: location.Location,
  to: number,
  index?: number,
): [Node, string | null] => {
  root = shallowCopyNode(root);
  const [tab, entry] = findTab(root, tabKey);
  if (tab == null || entry == null) throw new Error("Tab not found");
  const [r2, selected] = removeTab(root, tabKey);
  const r3 = insertTab(r2, tab, loc, to, index);
  return [r3, selected];
};

export const findTabNode = (root: Node, tabKey: string): Node | undefined => {
  if (root.tabs != null) {
    if (root.tabs.some((t) => t.tabKey === tabKey)) return root;
    return undefined;
  }
  if (root.first != null) {
    const node = findTabNode(root.first, tabKey);
    if (node != null) return node;
  }
  if (root.last != null) {
    const node = findTabNode(root.last, tabKey);
    if (node != null) return node;
  }
  return undefined;
};

/**
 * Resizes the given mosaic leaf.
 *
 * @param root - The root of the mosaic.
 * @param key  - The key of the leaf to resize.
 * @param size - The new size distribution for the leaf. Expressed as either a
 * percentage or a number of pixels of the first child.
 * @returns A shallow copy of the root of the mosaic with the leaf resized.
 */
export const resizeNode = (root: Node, key: number, size: number): Node => {
  const node = findMosaicNode(root, key);
  if (node == null) throw new Error("Node not found");
  else node.size = size;
  return root;
};

/**
 * Splits the node containing the tab with the given `tabKey`, moving the tab to a
 * new child node in the specified direction.
 *
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to move to the new split.
 * @param dir - The direction to split ('x' for vertical, 'y' for horizontal).
 * @returns A shallow copy of the root of the mosaic with the node split.
 */
export const split = (root: Node, tabKey: string, dir: direction.Direction): Node => {
  root = shallowCopyNode(root);
  const node = findTabNode(root, tabKey);
  if (node == null) throw new Error("Tab not found");
  if (node.tabs == null || node.tabs.length === 0) throw new Error("Node has no tabs");

  const tabIndex = node.tabs.findIndex((t) => t.tabKey === tabKey);
  if (tabIndex === -1) throw new Error("Tab not found in node");

  const tab = node.tabs[tabIndex];
  node.tabs = node.tabs.filter((t) => t.tabKey !== tabKey);

  const firstChildKey = node.key * 2;
  const lastChildKey = node.key * 2 + 1;

  const childWithTab: Node = {
    key: lastChildKey,
    tabs: [tab],
    selected: tab.tabKey,
  };

  const childWithoutTab: Node = {
    key: firstChildKey,
    tabs: node.tabs,
    selected: node.selected,
  };
  childWithoutTab.selected = resetSelection(
    childWithoutTab.selected,
    childWithoutTab.tabs,
  );

  node.direction = dir;
  node.first = childWithoutTab;
  node.last = childWithTab;

  // Clear the node's tabs and selected since it's now an internal node
  node.tabs = undefined;
  node.selected = undefined;

  return root;
};

/**
 * Determines if the node containing the tab with the given `tabKey` can be split
 * without resulting in one of the new child nodes having no tabs.
 *
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to move to the new split.
 * @returns True if the split is possible, false otherwise.
 */
export const canSplit = (root: Node, tabKey: string): boolean => {
  const node = findTabNode(root, tabKey);
  if (node == null) return false;
  return node.tabs != null && node.tabs.length > 1;
};

/**
 * Sets the title of a tab.
 *
 * @param root - The root of the mosaic.
 * @param tabKey  - The key of the tab to rename.
 * @param name - The new title of the tab.
 * @returns A shallow copy of the root of the mosaic with the tab title changed.
 */
export const renameTab = (root: Node, tabKey: string, name: string): Node => {
  name = name.trim();
  root = shallowCopyNode(root);
  const [, leaf] = findTab(root, tabKey);
  if (leaf?.tabs == null) throw new Error("Tab not found");
  if (name.length > 0)
    leaf.tabs = leaf.tabs.map((t) => (t.tabKey === tabKey ? { ...t, name } : t));
  return root;
};

/***
 * @returns True if the given mosaic node is empty, false otherwise.
 */
export const isEmpty = (root: Node): boolean => {
  if (root.tabs != null) return root.tabs.length === 0;
  return root.first == null && root.last == null;
};

/** Finds the node with the given key or its closest ancestor. */
const findNodeOrAncestor = (root: Node, key: number): Node => {
  const node = findMosaicNode(root, key);
  if (node != null) return node;
  const next = Math.floor(key / 2);
  return next === 0 ? root : findNodeOrAncestor(root, next);
};

const gc = (root: Node): Node => {
  let gced = true;
  while (gced) [root, gced] = _gc(root);
  // After garbage collection, ensure the root has key 1 and
  // recursively fix all child keys
  root = normalizeKeys(root, 1);
  return root;
};

const _gc = (node: Node): [Node, boolean] => {
  if (node.first == null || node.last == null) return [node, false];
  if (shouldGc(node.first)) return [liftUp(node.last), true];
  if (shouldGc(node.last)) return [liftUp(node.first), true];
  let sGC: boolean, eGC: boolean;
  [node.first, sGC] = _gc(node.first);
  [node.last, eGC] = _gc(node.last);
  return [node, sGC || eGC];
};

const liftUp = (node: Node): Node => {
  node.size = undefined;
  return node;
};

const shouldGc = (node: Node): boolean =>
  node.first == null &&
  node.last == null &&
  (node.tabs == null || node.tabs.length === 0);

const findTab = (node: Node, tabKey: string): [Tab | undefined, Node | undefined] => {
  if (node.tabs != null) {
    const tab = node.tabs.find((t) => t.tabKey === tabKey);
    if (tab != null) return [tab, node];
  }
  if (node.first == null || node.last == null) return [undefined, undefined];
  const [t1Tab, t2Tree] = findTab(node.first, tabKey);
  if (t1Tab != null && t2Tree != null) return [t1Tab, t2Tree];
  const [t2Tab, t2Tree2] = findTab(node.last, tabKey);
  return [t2Tab, t2Tree2];
};

const findMosaicNode = (node: Node, key: number): Node | undefined => {
  if (node.key === key) return node;
  if (node.first == null || node.last == null) return undefined;
  const n1 = findMosaicNode(node.first, key);
  if (n1 != null) return n1;
  return findMosaicNode(node.last, key);
};

const splitArrangement = (
  insertPosition: location.Location,
): [spatial.Order, spatial.Order, direction.Direction] => {
  switch (insertPosition) {
    case "top":
      return ["first", "last", "y"];
    case "left":
      return ["first", "last", "x"];
    case "bottom":
      return ["last", "first", "y"];
    case "right":
      return ["last", "first", "x"];
    case "center":
      throw new Error("cannot split a center placed tab");
  }
};

const shallowCopyNode = (node: Node): Node => ({
  ...node,
  first: node.first != null ? shallowCopyNode(node.first) : undefined,
  last: node.last != null ? shallowCopyNode(node.last) : undefined,
});

export const forEachNode = (root: Node, fn: (node: Node) => void): void => {
  fn(root);
  if (root.first != null) forEachNode(root.first, fn);
  if (root.last != null) forEachNode(root.last, fn);
};

// New helper function to normalize keys throughout the tree
const normalizeKeys = (node: Node, key: number): Node => {
  node = shallowCopyNode(node);
  node.key = key;
  if (node.first != null) node.first = normalizeKeys(node.first, key * 2);
  if (node.last != null) node.last = normalizeKeys(node.last, key * 2 + 1);
  return node;
};
