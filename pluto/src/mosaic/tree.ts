// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type direction, type location, type spatial } from "@synnaxlabs/x";

import { type Node } from "@/mosaic/types";
import { Tabs } from "@/tabs";

const TabNotFound = new Error("Tab not found");
const InvalidMosaic = new Error("Invalid Mosaic");

/**
 * Inserts a tab into a node in the mosaic. If the given key is not found,
 * the tab is inserted into the closest ancestor. This is to deal
 * with mosaic garbage collection.
 *
 * @param key - The key of the node to insert the tab into.
 * @param tab - The tab to insert.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 */
export const insertTab = (
  root: Node,
  tab: Tabs.Tab,
  loc: location.Location = "center",
  key?: number,
): Node => {
  root = shallowCopyNode(root);
  if (key === undefined) return insertAnywhere(root, tab);

  const node = findNodeOrAncestor(root, key);

  // In the case where we're dropping the node in the center,
  // simply add the tab, change the selection, and return.
  if (loc === "center") {
    node.tabs?.push(tab);
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

  // If we're not dropping the tab in the center,
  // and we have no tabs in the current node,
  // we can't split the node (because one side would be empty),
  // so we do nothing.
  if (node.tabs == null || node.tabs.length === 0) return root;

  const [insertOrder, siblingOrder, dir] = splitArrangement(loc);
  node.direction = dir;

  node[insertOrder] = { key: 0, tabs: [tab], selected: tab.tabKey };
  node[siblingOrder] = { key: 0, tabs: node.tabs, selected: node.selected };

  if (node.first == null || node.last == null) throw InvalidMosaic;

  // Assigning these keeps the mosaic sorted so we can do ancestor searches.
  node.first.key = firstChildKey;
  node.last.key = lastChildKey;

  // Clear the previous node, as it's now been and is not used
  // for rendering.
  node.tabs = undefined;
  node.size = undefined;
  node.selected = undefined;

  return root;
};

const insertAnywhere = (root: Node, tab: Tabs.Tab): Node => {
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
 * Automatically selects tabs for all nodes in the mosaic if they don't already have
 * a selection.
 *
 * @param root - The root of the mosaic.
 * @returns A shallow copy of the root with all nodes having a selection.
 */
export const autoSelectTabs = (root: Node): [Node, string[]] => {
  root = shallowCopyNode(root);
  const selected: string[] = [];
  if (root.tabs != null) {
    root.selected = Tabs.resetSelection(root.selected, root.tabs);
    if (root.selected != null) selected.push(root.selected);
  }
  if (root.first != null) {
    const [f, s] = autoSelectTabs(root.first);
    root.first = f;
    selected.push(...s);
  }
  if (root.last != null) {
    const [r, l] = autoSelectTabs(root.last);
    root.last = r;
    selected.push(...l);
  }
  return [root, selected];
};

/**
 * Removes a tab from the mosaic and performs any necessary garbage collection.
 *
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to remove. This tab must exist in the mosaic.
 */
export const removeTab = (root: Node, tabKey: string): [Node, string | null] => {
  root = shallowCopyNode(root);
  const [, node] = findMosaicTab(root, tabKey);
  if (node == null) return [root, null];
  node.tabs = node.tabs?.filter((t) => t.tabKey !== tabKey);
  node.selected = Tabs.resetSelection(node.selected, node.tabs);
  root = gc(root);
  const selected = node.selected ?? findSelected(root);
  return [root, selected];
};

export const findSelected = (root: Node): string | null => {
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
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (tab == null || entry == null) throw TabNotFound;
  entry.selected = tabKey;
  return root;
};

/**
 * Moves a tab from one node to another.
 *
 * @param root - The root of the mosaic.
 * @param to - The key of the node to move the tab to.
 * @param tabKey - The key of the tab to move. This tab must exist in the mosaic.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 * @returns A shallow copy of the root of the mosaic with the tab moved.
 */
export const moveTab = (
  root: Node,
  tabKey: string,
  loc: location.Location,
  to: number,
): [Node, string | null] => {
  root = shallowCopyNode(root);
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (tab == null || entry == null) throw TabNotFound;
  const [r2, selected] = removeTab(root, tabKey);
  const r3 = insertTab(r2, tab, loc, to);
  return [r3, selected];
};

/**
 * Resizes the given mosaic leaf.
 *
 * @param root - The root of the mosaic.
 * @param key  - The key of the leaf to resize.
 * @param size - The new size distribution for the leaf. Expressed as either a percentage
 * or a number of pixels of the first child.
 * @returns A shallow copy of the root of the mosaic with the leaf resized.
 */
export const resizeNode = (root: Node, key: number, size: number): Node => {
  const node = findMosaicNode(root, key);
  if (node == null) throw new Error("Node not found");
  else node.size = size;
  return root;
};

/**
 * Sets the title of a tab.
 *
 * @param root - The root of the mosaic.
 * @param tabKey  - The key of the tab to resize.
 * @param name - The new title of the tab.
 * @returns A shallow copy of the root of the mosaic with the tab title changed.
 */
export const renameTab = (root: Node, tabKey: string, name: string): Node => {
  root = shallowCopyNode(root);
  const [, leaf] = findMosaicTab(root, tabKey);
  if (leaf?.tabs == null) throw TabNotFound;
  leaf.tabs = Tabs.rename(tabKey, name, leaf?.tabs ?? []);
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
  if (root.first == null && root.last == null) root.key = 1;
  return root;
};

const _gc = (node: Node): [Node, boolean] => {
  if (node.first == null || node.last == null) return [node, false];
  if (shouldGc(node.first)) return [liftUp(node.last, true), true];
  if (shouldGc(node.last)) return [liftUp(node.first, false), true];
  let [sGC, eGC] = [false, false];
  [node.first, sGC] = _gc(node.first);
  [node.last, eGC] = _gc(node.last);
  return [node, sGC || eGC];
};

const liftUp = (node: Node, isLast: boolean): Node => {
  node.size = undefined;
  node.key = (node.key - Number(isLast)) / 2;
  return node;
};

const shouldGc = (node: Node): boolean =>
  node.first == null &&
  node.last == null &&
  (node.tabs == null || node.tabs.length === 0);

const findMosaicTab = (
  node: Node,
  tabKey: string,
): [Tabs.Tab | undefined, Node | undefined] => {
  if (node.tabs != null) {
    const tab = node.tabs.find((t) => t.tabKey === tabKey);
    if (tab != null) return [tab, node];
  }
  if (node.first == null || node.last == null) return [undefined, undefined];
  const [t1Tab, t2Tree] = findMosaicTab(node.first, tabKey);
  if (t1Tab != null && t2Tree != null) return [t1Tab, t2Tree];
  const [t2Tab, t2Tree2] = findMosaicTab(node.last, tabKey);
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

const shallowCopyNode = (node: Node): Node => deep.copy(node);

export const mapNodes = <O>(root: Node, fn: (node: Node) => O, acc: O[] = []): O[] => {
  acc.push(fn(root));
  if (root.first != null) mapNodes(root.first, fn, acc);
  if (root.last != null) mapNodes(root.last, fn, acc);
  return acc;
};

export const forEachNode = (root: Node, fn: (node: Node) => void): void => {
  fn(root);
  if (root.first != null) forEachNode(root.first, fn);
  if (root.last != null) forEachNode(root.last, fn);
};
