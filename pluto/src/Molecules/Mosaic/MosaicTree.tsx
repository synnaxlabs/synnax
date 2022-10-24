import { TabEntry } from "../../Atoms/Tabs";
import { Direction, Location, Order } from "../../util/spatial";

const TabNotFound = new Error("Tab not found");

export interface MosaicNode {
  key: number;
  level: number;
  tabs?: TabEntry[];
  direction?: Direction;
  first?: MosaicNode;
  last?: MosaicNode;
  selected?: string;
  size?: number;
}

export default class MosaicTree {
  private root: MosaicNode;

  constructor(root: MosaicNode) {
    this.root = root;
  }

  /**
   * @returns A shallow copy of the root node.
   */
  shallowCopy(): MosaicNode {
    return { ...this.root };
  }

  /**
   * Inserts a tab into a node in the tree. If the given key is not found,
   * the tab is inserted into the closest ancestor. This is to deal
   * with tree garbage collection.
   * @param key - The key of the node to insert the tab into.
   * @param entry - The tab to insert.
   * @param loc - The location where the tab was 'dropped' relative to the node.
   */
  insert(key: number, entry: TabEntry, loc: Location) {
    const node = this.findOrAncestor(key);
    if (loc === "center") return node.tabs?.push(entry);
    const [insertTo, sibling, dir] = splitArrangement(loc);
    node[insertTo] = { key: 0, tabs: [entry], level: node.level + 1 };
    node[sibling] = { key: 0, tabs: node.tabs, level: node.level + 1 };
    if (!node.first || !node.last) throw new Error("Invalid tree");
    /** Assigning these values to start and end keeps the tree sorted */
    node.first.key = node.key * 2;
    node.last.key = node.key * 2 + 1;
    node.direction = dir;
    node.tabs = undefined;
  }

  insertAnywhere(entry: TabEntry) {
    return this._insertAnywhere(this.root, entry);
  }

  private _insertAnywhere(node: MosaicNode, entry: TabEntry): void {
    if (node.tabs) {
      node.tabs.push(entry);
      return;
    }
    if (node.first) return this._insertAnywhere(node.first, entry);
    if (node.last) return this._insertAnywhere(node.last, entry);
  }

  /**
   * Removes a tab from the tree and performs any necessary garbage collection.
   * @param tabKey - The key of the tab to remove. This tab must exist in the tree.
   */
  remove(tabKey: string) {
    const [, entry] = this.findTab(tabKey);
    if (!entry) throw TabNotFound;
    entry.tabs = entry.tabs?.filter((t) => t.tabKey !== tabKey);
    this.gc();
  }

  /**
   * Marks the given tab as selected.
   */
  select(tabKey: string) {
    const [tab, entry] = this.findTab(tabKey);
    if (!tab || !entry) throw TabNotFound;
    entry.selected = tabKey;
  }

  /**
   * Moves a tab from one node to another.
   * @param to - The key of the node to move the tab to.
   * @param tabKey - The key of the tab to move. This tab must exist in the tree.
   * @param loc - The location where the tab was 'dropped' relative to the node.
   */
  move(to: number, tabKey: string, loc: Location) {
    const [tab, entry] = this.findTab(tabKey);
    if (!tab || !entry) throw TabNotFound;
    this.remove(tabKey);
    this.insert(to, tab, loc);
  }

  resize(key: number, size: number) {
    const node = this.find(key);
    if (!node) throw new Error("Node not found");
    else node.size = size;
  }

  /**
   * Finds the node with the given key or its closest ancestor.
   * @param key  - The key of the node to find.
   * @returns The node with the given key, or the closest ancestor.
   */
  findOrAncestor(key: number): MosaicNode {
    const node = this.find(key);
    if (node) return node;
    return this.findOrAncestor(Math.floor(key / 2));
  }

  /**
   * Finds the node with the given key.
   * @param key  - The key of the node to find.
   * @returns The node with the given key, or undefined if it does not exist.
   */
  find(key: number): MosaicNode | undefined {
    return this._find(this.root, key);
  }

  findTab(tabKey: string): [TabEntry | undefined, MosaicNode | undefined] {
    return this._findTab(this.root, tabKey);
  }

  /**
   * Garbage collects the tree, removing any redundant nodes.
   */
  gc() {
    let gced = true;
    while (gced) {
      [this.root, gced] = this._gc(this.root);
    }
  }

  private _gc(node: MosaicNode): [MosaicNode, boolean] {
    if (!node.first || !node.last) return [node, false];
    if (this.shouldGc(node.first)) return [node.last, true];
    if (this.shouldGc(node.last)) return [node.first, true];
    let [sGC, eGC] = [false, false];
    [node.first, sGC] = this._gc(node.first);
    [node.last, eGC] = this._gc(node.last);
    return [node, sGC || eGC];
  }

  private shouldGc(node: MosaicNode): boolean {
    return !node.first && !node.last && (!node.tabs || node.tabs.length === 0);
  }

  private _findTab(
    node: MosaicNode,
    tabKey: string
  ): [TabEntry | undefined, MosaicNode | undefined] {
    if (node.tabs) {
      const tab = node.tabs.find((t) => t.tabKey === tabKey);
      if (tab) return [tab, node];
    }
    if (!node.first || !node.last) return [undefined, undefined];
    const [t1Tab, t2Tree] = this._findTab(node.first, tabKey);
    if (t1Tab && t2Tree) return [t1Tab, t2Tree];
    const [t2Tab, t2Tree2] = this._findTab(node.last, tabKey);
    return [t2Tab, t2Tree2];
  }

  private _find(node: MosaicNode, key: number): MosaicNode | undefined {
    if (node.key === key) return node;
    if (!node.first || !node.last) return undefined;
    const n1 = this._find(node.first, key);
    if (n1) return n1;
    return this._find(node.last, key);
  }
}

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
