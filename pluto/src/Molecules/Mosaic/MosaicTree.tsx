import { Tab } from "@/atoms";
import { Direction, Location, Order } from "@/util";

const TabNotFound = new Error("Tab not found");

export interface MosaicLeaf {
  key: number;
  level: number;
  tabs?: Tab[];
  direction?: Direction;
  first?: MosaicLeaf;
  last?: MosaicLeaf;
  selected?: string;
  size?: number;
}

/**
 * Inserts a tab into a node in the tree. If the given key is not found,
 * the tab is inserted into the closest ancestor. This is to deal
 * with tree garbage collection.
 * @param key - The key of the node to insert the tab into.
 * @param tab - The tab to insert.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 */
export const insertMosaicTab = (
  root: MosaicLeaf,
  tab: Tab,
  key?: number,
  loc?: Location
): MosaicLeaf => {
  if (!loc) loc = "center";
  if (key === undefined) {
    insertAnywhere(root, tab);
    return root;
  }
  const node = findOrAncestor(root, key);
  if (loc === "center") {
    node.tabs?.push(tab);
    return root;
  }
  const [insertTo, sibling, dir] = splitArrangement(loc);
  node[insertTo] = {
    key: 0,
    tabs: [tab],
    level: node.level + 1,
    selected: tab.tabKey,
  };

  const siblingSelected =
    node.tabs?.find((t) => t.tabKey === node.selected)?.tabKey ??
    node.tabs?.[0]?.tabKey;

  node[sibling] = {
    key: 0,
    tabs: node.tabs,
    level: node.level + 1,
    selected: siblingSelected,
  };
  if (!node.first || !node.last) throw new Error("Invalid tree");
  /** Assigning these values to start and end keeps the tree sorted */
  node.first.key = node.key * 2;
  node.last.key = node.key * 2 + 1;
  node.direction = dir;
  node.tabs = undefined;
  node.size = undefined;
  node.selected = "";
  return root;
};

const insertAnywhere = (node: MosaicLeaf, tab: Tab): void => {
  if (node.tabs !== undefined) {
    node.tabs.push(tab);
    node.selected = tab.tabKey;
    return;
  }
  if (node.first) return insertAnywhere(node.first, tab);
  if (node.last) return insertAnywhere(node.last, tab);
};

/**
 * Removes a tab from the tree and performs any necessary garbage collection.
 * @param tabKey - The key of the tab to remove. This tab must exist in the tree.
 */
export const removeMosaicTab = (
  root: MosaicLeaf,
  tabKey: string
): MosaicLeaf => {
  const [, entry] = findMosaicTab(root, tabKey);
  if (!entry) throw TabNotFound;
  entry.tabs = entry.tabs?.filter((t) => t.tabKey !== tabKey);
  if (!entry.tabs?.find((t) => t.tabKey === entry.selected)) {
    entry.selected = entry.tabs?.[0]?.tabKey;
  }

  return gc(root);
};

/**
 * Marks the given tab as selected.
 */
export const selectMosaicTab = (
  root: MosaicLeaf,
  tabKey: string
): MosaicLeaf => {
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (!tab || !entry) throw TabNotFound;
  entry.selected = tabKey;
  return root;
};

/**
 * Moves a tab from one node to another.
 * @param to - The key of the node to move the tab to.
 * @param tabKey - The key of the tab to move. This tab must exist in the tree.
 * @param loc - The location where the tab was 'dropped' relative to the node.
 */
export const moveMosaicTab = (
  root: MosaicLeaf,
  tabKey: string,
  to: number,
  loc: Location
): MosaicLeaf => {
  const [tab, entry] = findMosaicTab(root, tabKey);
  if (!tab || !entry) throw TabNotFound;
  const r2 = removeMosaicTab(root, tabKey);
  const r3 = insertMosaicTab(r2, tab, to, loc);
  return r3;
};

export const resizeMosaicLeaf = (
  root: MosaicLeaf,
  key: number,
  size: number
) => {
  const node = findMosaicLeaf(root, key);
  if (!node) throw new Error("Node not found");
  else node.size = size;
  return root;
};

/**
 * Finds the node with the given key or its closest ancestor.
 * @param key  - The key of the node to find.
 * @returns The node with the given key, or the closest ancestor.
 */
const findOrAncestor = (root: MosaicLeaf, key: number): MosaicLeaf => {
  const node = findMosaicLeaf(root, key);
  if (node) return node;
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
  if (!node.first || !node.last) return [node, false];
  if (shouldGc(node.first)) return [liftUp(node.last, false), true];
  if (shouldGc(node.last)) return [liftUp(node.first, true), true];
  let [sGC, eGC] = [false, false];
  [node.first, sGC] = _gc(node.first);
  [node.last, eGC] = _gc(node.last);
  return [node, sGC || eGC];
};

const liftUp = (node: MosaicLeaf, isFirst: boolean): MosaicLeaf => {
  node.size = undefined;
  return node;
};

const shouldGc = (node: MosaicLeaf): boolean => {
  return !node.first && !node.last && (!node.tabs || node.tabs.length === 0);
};

const findMosaicTab = (
  node: MosaicLeaf,
  tabKey: string
): [Tab | undefined, MosaicLeaf | undefined] => {
  if (node.tabs) {
    const tab = node.tabs.find((t) => t.tabKey === tabKey);
    if (tab) return [tab, node];
  }
  if (!node.first || !node.last) return [undefined, undefined];
  const [t1Tab, t2Tree] = findMosaicTab(node.first, tabKey);
  if (t1Tab && t2Tree) return [t1Tab, t2Tree];
  const [t2Tab, t2Tree2] = findMosaicTab(node.last, tabKey);
  return [t2Tab, t2Tree2];
};

const findMosaicLeaf = (
  node: MosaicLeaf,
  key: number
): MosaicLeaf | undefined => {
  if (node.key === key) return node;
  if (!node.first || !node.last) return undefined;
  const n1 = findMosaicLeaf(node.first, key);
  if (n1) return n1;
  return findMosaicLeaf(node.last, key);
};

const splitArrangement = (
  insertPosition: Location
): [Order, Order, Direction] => {
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
