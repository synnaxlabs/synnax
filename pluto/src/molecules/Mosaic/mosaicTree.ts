import { MosaicLeaf } from "./types";

import { Tab, Tabs } from "@/atoms";
import { Direction, Location, Order } from "@/util/spatial";

const TabNotFound = new Error("Tab not found");
const InvalidMosaic = new Error("Invalid Mosaic");

/**
 * Inserts a tab into a node in the mosaic. If the given key is not found,
 * the tab is inserted into the closest ancestor. This is to deal
 * with mosaic garbage collection.
 * @param key - The key of the node to insert the tab into.
 * @param tab - The tab to insert.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 */
export const insertMosaicTab = (
  root: MosaicLeaf,
  tab: Tab,
  loc: Location = "center",
  key?: number
): MosaicLeaf => {
  if (key === undefined) return insertAnywhere(root, tab);

  const node = findOrAncestor(root, key);

  // In the case where we're dropping the node in the center,
  // simply add the tab, change the selection, and return.
  if (loc === "center") {
    node.tabs?.push(tab);
    node.selected = tab.tabKey;
    return root;
  }

  // If we're not dropping the tab in the center,
  // and we have no tabs in the current node,
  // we can't split the node (because on side would be empty),
  // so we do nothing.
  if (node.tabs == null || node.tabs.length === 0) return root;

  const [insertOrder, siblingOrder, dir] = splitArrangement(loc);
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

  return shallowCopyLeaf(root);
};

const insertAnywhere = (node: MosaicLeaf, tab: Tab): MosaicLeaf => {
  if (node.tabs != null) {
    node.tabs.push(tab);
    node.selected = tab.tabKey;
    return node;
  }
  if (node.first != null) node.first = insertAnywhere(node.first, tab);
  else if (node.last != null) node.last = insertAnywhere(node.last, tab);
  return node;
};

/**
 * Automatically selects tabs for all nodes in the mosaic if they don't already have a selection.
 * @param root - The root of the mosaic.
 * @returns A shallow copy of the root with all nodes having a selection.
 */
export const autoSelectTabs = (root: MosaicLeaf): MosaicLeaf => {
  // recursively iterate through mosaic and call Tabs.resetSelection on each nodes tabs
  if (root.tabs != null) root.selected = Tabs.resetSelection(root.selected, root.tabs);
  if (root.first != null) autoSelectTabs(root.first);
  if (root.last != null) autoSelectTabs(root.last);
  return shallowCopyLeaf(root);
};

/**
 * Removes a tab from the mosaic and performs any necessary garbage collection.
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to remove. This tab must exist in the mosaic.
 */
export const removeMosaicTab = (root: MosaicLeaf, tabKey: string): MosaicLeaf => {
  const [, node] = findMosaicTab(root, tabKey);
  if (node == null) throw TabNotFound;
  node.tabs = node.tabs?.filter((t) => t.tabKey !== tabKey);
  node.selected = Tabs.resetSelection(node.selected, node.tabs);
  return shallowCopyLeaf(gc(root));
};

/**
 * Marks the given tab as selected.
 * @param root - The root of the mosaic.
 * @param tabKey - The key of the tab to select. This tab must exist in the mosaic.
 * @returns A shallow copy of the root of the mosaic with the tab selected.
 */
export const selectMosaicTab = (root: MosaicLeaf, tabKey: string): MosaicLeaf => {
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (tab == null || entry == null) throw TabNotFound;
  entry.selected = tabKey;
  return shallowCopyLeaf(root);
};

/**
 * Moves a tab from one node to another.
 * @param root - The root of the mosaic.
 * @param to - The key of the node to move the tab to.
 * @param tabKey - The key of the tab to move. This tab must exist in the mosaic.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 * @returns A shallow copy of the root of the mosaic with the tab moved.
 */
export const moveMosaicTab = (
  root: MosaicLeaf,
  tabKey: string,
  loc: Location,
  to: number
): MosaicLeaf => {
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (tab == null || entry == null) throw TabNotFound;
  const r2 = removeMosaicTab(root, tabKey);
  const r3 = insertMosaicTab(r2, tab, loc, to);
  return shallowCopyLeaf(r3);
};

/**
 * Resizes the given mosaic leaf.
 * @param root - The root of the mosaic.
 * @param key  - The key of the leaf to resize.
 * @param size - The new size distribution for the leaf. Expressed as either a percentage
 * or a number of pixels of the first child.
 * @returns A shallow copy of the root of the mosaic with the leaf resized.
 */
export const resizeMosaicLeaf = (
  root: MosaicLeaf,
  key: number,
  size: number
): MosaicLeaf => {
  const node = findMosaicLeaf(root, key);
  if (node == null) throw new Error("Node not found");
  else node.size = size;
  return root;
};

/**
 * Sets the title of a tab.
 * @param root - The root of the mosaic.
 * @param tabKey  - The key of the tab to resize.
 * @param title - The new title of the tab.
 * @returns A shallow copy of the root of the mosaic with the tab title changed.
 */
export const renameMosaicTab = (
  root: MosaicLeaf,
  tabKey: string,
  title: string
): MosaicLeaf => {
  const [, leaf] = findMosaicTab(root, tabKey);
  if (leaf == null || leaf.tabs == null) throw TabNotFound;
  leaf.tabs = Tabs.rename(tabKey, title, leaf?.tabs ?? []);
  return shallowCopyLeaf(root);
};

/**
 * Finds the node with the given key or its closest ancestor.
 * @param root - The root of the mosaic.
 * @param key  - The key of the node to find.
 * @returns The node with the given key, or the closest ancestor.
 */
const findOrAncestor = (root: MosaicLeaf, key: number): MosaicLeaf => {
  const node = findMosaicLeaf(root, key);
  if (node != null) return node;
  return findOrAncestor(root, Math.floor(key / 2));
};

const gc = (root: MosaicLeaf): MosaicLeaf => {
  let gced = true;
  while (gced) {
    [root, gced] = _gc(root);
  }
  return root;
};

const _gc = (node: MosaicLeaf): [MosaicLeaf, boolean] => {
  if (node.first == null || node.last == null) return [node, false];
  if (shouldGc(node.first)) return [liftUp(node.last, true), true];
  if (shouldGc(node.last)) return [liftUp(node.first, false), true];
  let [sGC, eGC] = [false, false];
  [node.first, sGC] = _gc(node.first);
  [node.last, eGC] = _gc(node.last);
  return [node, sGC || eGC];
};

const liftUp = (node: MosaicLeaf, isLast: boolean): MosaicLeaf => {
  node.size = undefined;
  node.key = (node.key - Number(isLast)) / 2;
  return node;
};

const shouldGc = (node: MosaicLeaf): boolean => {
  return (
    node.first == null &&
    node.last == null &&
    (node.tabs == null || node.tabs.length === 0)
  );
};

const findMosaicTab = (
  node: MosaicLeaf,
  tabKey: string
): [Tab | undefined, MosaicLeaf | undefined] => {
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

const findMosaicLeaf = (node: MosaicLeaf, key: number): MosaicLeaf | undefined => {
  if (node.key === key) return node;
  if (node.first == null || node.last == null) return undefined;
  const n1 = findMosaicLeaf(node.first, key);
  if (n1 != null) return n1;
  return findMosaicLeaf(node.last, key);
};

const splitArrangement = (insertPosition: Location): [Order, Order, Direction] => {
  switch (insertPosition) {
    case "top":
      return ["first", "last", "vertical"];
    case "left":
      return ["first", "last", "horizontal"];
    case "bottom":
      return ["last", "first", "vertical"];
    case "right":
      return ["last", "first", "horizontal"];
    case "center":
      throw new Error("cannot split a center placed tab");
  }
};

const shallowCopyLeaf = (leaf: MosaicLeaf): MosaicLeaf => ({ ...leaf });
