// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { compare, toArray } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Haul } from "@/haul";

export interface Node {
  key: string;
  name: string;
  renaming?: boolean;
  forcePosition?: number;
  icon?: ReactElement;
  allowRename?: boolean;
  hasChildren?: boolean;
  children?: Node[];
  haulItems?: Haul.Item[];
  canDrop?: (items: Haul.Item[]) => boolean;
  href?: string;
}

export interface NodeWithPosition extends Node {
  depth: number;
  position: number;
}

export interface FlattenedNode extends Node {
  index: number;
  depth: number;
  expanded: boolean;
  path: string;
}

export const shouldExpand = (node: Node, expanded: string[]): boolean =>
  expanded.includes(node.key);

export interface FlattenProps {
  nodes: Node[];
  expanded: string[];
  depth?: number;
  sort?: boolean;
  path?: string;
}

export const sortAndSplice = (nodes: Node[], sort: boolean): Node[] => {
  if (sort) nodes.sort((a, b) => compare.stringsWithNumbers(a.name, b.name));
  let found = false;
  for (let i = 0; i < nodes.length; i++) {
    const node = nodes[i];
    if (node.forcePosition != null && i !== node.forcePosition) {
      found = true;
      // remove the node from its current position
      nodes.splice(i, 1);
      // splice it into the forced position
      nodes.splice(node.forcePosition, 0, node);
    }
  }
  if (found) return sortAndSplice(nodes, false);
  return nodes;
};

export const flatten = ({
  nodes,
  expanded,
  depth = 0,
  sort = true,
  path = "",
}: FlattenProps): FlattenedNode[] => {
  // Sort the first level of the tree independently of the rest
  if (depth === 0 && sort) nodes = nodes.sort((a, b) => a.name.localeCompare(b.name));
  const flattened: FlattenedNode[] = [];
  nodes.forEach((node, index) => {
    const nextPath = `${path}${node.key}/`;
    const expand = shouldExpand(node, expanded);
    flattened.push({ ...node, depth, expanded: expand, index, path: nextPath });
    if (expand && node.children != null) {
      node.children = sortAndSplice(node.children, sort);
      flattened.push(
        ...flatten({
          nodes: node.children,
          expanded,
          depth: depth + 1,
          sort,
          path: nextPath,
        }),
      );
    }
  });
  return flattened;
};

export interface MoveNodeProps {
  tree: Node[];
  destination: string;
  keys: string | string[];
}

export const moveNode = ({ tree, destination, keys }: MoveNodeProps): Node[] => {
  keys = toArray(keys);
  keys.forEach((key) => {
    const node = findNode({ tree, key });
    if (node == null) return;
    removeNode({ tree, keys: key });
    setNode({ tree, destination, additions: node });
  });
  return tree;
};

export interface RemoveNodeProps {
  tree: Node[];
  keys: string | string[];
}

export const removeNode = ({ tree, keys }: RemoveNodeProps): Node[] => {
  keys = toArray(keys);
  const treeKeys = tree.map((node) => node.key);
  keys.forEach((key) => {
    const index = treeKeys.indexOf(key);
    if (index !== -1) tree.splice(index, 1);
    else {
      const parent = findNodeParent({ tree, key });
      if (parent != null)
        parent.children = parent.children?.filter((child) => child.key !== key);
    }
  });
  return tree;
};

export interface SetNodeProps {
  tree: Node[];
  destination: string;
  additions: Node | Node[];
}

export const setNode = ({ tree, destination, additions }: SetNodeProps): Node[] => {
  additions = toArray(additions);
  const node = findNode({ tree, key: destination });
  if (node == null) throw new Error(`Could not find node with key ${destination}`);
  node.children ??= [];
  const addedKeys = additions.map((node) => node.key);
  node.children = [
    ...additions,
    ...node.children.filter((child) => !addedKeys.includes(child.key)),
  ];
  return tree;
};

export interface UpdateNodeProps {
  tree: Node[];
  key: string;
  updater: (node: Node) => Node;
  throwOnMissing?: boolean;
}

export const updateNode = ({
  tree,
  key,
  updater,
  throwOnMissing = true,
}: UpdateNodeProps): Node[] => {
  const node = findNode({ tree, key });
  if (node == null) {
    if (throwOnMissing) throw new Error(`Could not find node with key ${key}`);
    return tree;
  }
  const parent = findNodeParent({ tree, key });
  if (parent != null) {
    // splice the updated node into the parent's children
    const index = parent.children?.findIndex((child) => child.key === key);
    if (index != null && index !== -1) parent.children?.splice(index, 1, updater(node));
  }
  // we're in the root, so just update the node
  else
    tree.splice(
      tree.findIndex((node) => node.key === key),
      1,
      updater(node),
    );
  return tree;
};

interface UpdateNodeChildren {
  tree: Node[];
  parent: string;
  updater: (nodes: Node[]) => Node[];
  throwOnMissing?: boolean;
}

export const updateNodeChildren = ({
  tree,
  parent,
  updater,
  throwOnMissing = true,
}: UpdateNodeChildren): Node[] => {
  const parentNode = findNode({ tree, key: parent });
  if (parentNode == null) {
    if (throwOnMissing) throw new Error(`Could not find node with key ${parent}`);
    return tree;
  }
  parentNode.children = updater(parentNode.children ?? []);
  return tree;
};

export interface FindNodeProps {
  tree: Node[];
  key: string;
  depth?: number;
}

export const findNode = ({
  tree,
  key,
  depth = 0,
}: FindNodeProps): NodeWithPosition | null => {
  for (let i = 0; i < tree.length; i++) {
    const node = tree[i];
    if (node.key === key) {
      const n = node as NodeWithPosition;
      n.depth = depth;
      n.position = i;
      return n;
    }
    if (node.children != null) {
      const found = findNode({ tree: node.children, key, depth: depth + 1 });
      if (found != null) return found;
    }
  }
  return null;
};

export interface FindNodesProps {
  tree: Node[];
  keys: string[];
}

export const findNodes = ({ tree, keys }: FindNodesProps): NodeWithPosition[] => {
  const nodes: NodeWithPosition[] = [];
  for (const key of keys) {
    const node = findNode({ tree, key });
    if (node != null) nodes.push(node);
  }
  return nodes;
};

export interface FindNodeParentProps {
  tree: Node[];
  key: string;
}

export const findNodeParent = ({ tree, key }: FindNodeParentProps): Node | null => {
  for (const node of tree)
    if (node.children != null) {
      if (node.children.some((child) => child.key === key)) return node;
      const found = findNodeParent({ tree: node.children, key });
      if (found != null) return found;
    }
  return null;
};

export const deepCopy = (nodes: Node[]): Node[] =>
  nodes.map((node) => ({ ...node, children: deepCopy(node.children ?? []) }));

export const getDescendants = (...node: Node[]): Node[] => {
  const descendants: Node[] = [];
  node.forEach((n) => {
    descendants.push(n);
    if (n.children != null) descendants.push(...getDescendants(...n.children));
  });
  return descendants;
};
