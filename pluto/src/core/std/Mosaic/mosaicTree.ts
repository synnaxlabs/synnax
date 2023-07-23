// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  CrudeDirection,
  Location,
  CrudeLocation,
  LooseLocationT,
  CrudeOrder,
  Deep,
} from "@synnaxlabs/x";

import { MosaicNode } from "@/core/std/Mosaic/types";
import { Tab, Tabs } from "@/core/std/Tabs";

const TabNotFound = new Error("Tab not found");
const InvalidMosaic = new Error("Invalid Mosaic");

export const insertMosaicTab = (
  root: MosaicNode,
  tab: Tab,
  loc_: LooseLocationT = "center",
  key?: number
): MosaicNode => {
  root = shallowCopyNode(root);
  const loc = new Location(loc_);
  if (key === undefined) return insertAnywhere(root, tab);

  const node = findNodeOrAncestor(root, key);

  // In the case where we're dropping the node in the center,
  // simply add the tab, change the selection, and return.
  if (loc.equals("center")) {
    node.tabs?.push(tab);
    node.selected = tab.tabKey;
    return root;
  }

  // If we're not dropping the tab in the center,
  // and we have no tabs in the current node,
  // we can't split the node (because on side would be empty),
  // so we do nothing.
  if (node.tabs == null || node.tabs.length === 0) return root;

  const [insertOrder, siblingOrder, dir] = splitArrangement(loc.crude);
  node.direction = dir;

  node[insertOrder] = { key: 0, tabs: [tab], selected: tab.tabKey };
  node[siblingOrder] = { key: 0, tabs: node.tabs, selected: node.selected };

  if (node.first == null || node.last == null) throw InvalidMosaic;

  // Assigning these keeps the mosaic sorted so we can do ancestor searches.
  node.first.key = node.key * 2;
  node.last.key = node.key * 2 + 1;

  // Clear the previous node, as it's now been and is not used
  // for rendering.
  node.tabs = undefined;
  node.size = undefined;
  node.selected = undefined;

  return root;
};

const insertAnywhere = (root: MosaicNode, tab: Tab): MosaicNode => {
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

export const autoSelectTabs = (root: MosaicNode): [MosaicNode, string[]] => {
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

export const removeMosaicTab = (
  root: MosaicNode,
  tabKey: string
): [MosaicNode, string | null] => {
  root = shallowCopyNode(root);
  const [, node] = findMosaicTab(root, tabKey);
  if (node == null) throw TabNotFound;
  node.tabs = node.tabs?.filter((t) => t.tabKey !== tabKey);
  node.selected = Tabs.resetSelection(node.selected, node.tabs);
  const gced = gc(root);
  const selected = node.selected ?? findSelected(gced);
  return [gced, selected];
};

export const findSelected = (root: MosaicNode): string | null => {
  if (root.selected != null) return root.selected;
  if (root.first != null) return findSelected(root.first);
  if (root.last != null) return findSelected(root.last);
  return null;
};

export const selectMosaicTab = (root: MosaicNode, tabKey: string): MosaicNode => {
  root = shallowCopyNode(root);
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (tab == null || entry == null) throw TabNotFound;
  entry.selected = tabKey;
  return root;
};

export const moveMosaicTab = (
  root: MosaicNode,
  tabKey: string,
  loc: LooseLocationT,
  to: number
): [MosaicNode, string | null] => {
  root = shallowCopyNode(root);
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (tab == null || entry == null) throw TabNotFound;
  const [r2, selected] = removeMosaicTab(root, tabKey);
  const r3 = insertMosaicTab(r2, tab, loc, to);
  return [r3, selected];
};

export const resizeMosaicNode = (
  root: MosaicNode,
  key: number,
  size: number
): MosaicNode => {
  const node = findMosaicNode(root, key);
  if (node == null) throw new Error("Node not found");
  else node.size = size;
  return root;
};

export const renameMosaicTab = (
  root: MosaicNode,
  tabKey: string,
  name: string
): MosaicNode => {
  root = shallowCopyNode(root);
  const [, leaf] = findMosaicTab(root, tabKey);
  if (leaf == null || leaf.tabs == null) throw TabNotFound;
  leaf.tabs = Tabs.rename(tabKey, name, leaf?.tabs ?? []);
  return root;
};

/** Finds the node with the given key or its closest ancestor. */
const findNodeOrAncestor = (root: MosaicNode, key: number): MosaicNode => {
  const node = findMosaicNode(root, key);
  if (node != null) return node;
  const next = Math.floor(key / 2);
  return next === 0 ? root : findNodeOrAncestor(root, next);
};

const gc = (root: MosaicNode): MosaicNode => {
  let gced = true;
  while (gced) [root, gced] = _gc(root);
  return root;
};

const _gc = (node: MosaicNode): [MosaicNode, boolean] => {
  if (node.first == null || node.last == null) return [node, false];
  if (shouldGc(node.first)) return [liftUp(node.last, true), true];
  if (shouldGc(node.last)) return [liftUp(node.first, false), true];
  let [sGC, eGC] = [false, false];
  [node.first, sGC] = _gc(node.first);
  [node.last, eGC] = _gc(node.last);
  return [node, sGC || eGC];
};

const liftUp = (node: MosaicNode, isLast: boolean): MosaicNode => {
  node.size = undefined;
  node.key = (node.key - Number(isLast)) / 2;
  return node;
};

const shouldGc = (node: MosaicNode): boolean =>
  node.first == null &&
  node.last == null &&
  (node.tabs == null || node.tabs.length === 0);

const findMosaicTab = (
  node: MosaicNode,
  tabKey: string
): [Tab | undefined, MosaicNode | undefined] => {
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

const findMosaicNode = (node: MosaicNode, key: number): MosaicNode | undefined => {
  if (node.key === key) return node;
  if (node.first == null || node.last == null) return undefined;
  const n1 = findMosaicNode(node.first, key);
  if (n1 != null) return n1;
  return findMosaicNode(node.last, key);
};

const splitArrangement = (
  insertPosition: CrudeLocation
): [CrudeOrder, CrudeOrder, CrudeDirection] => {
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

const shallowCopyNode = (node: MosaicNode): MosaicNode => Deep.copy(node);
