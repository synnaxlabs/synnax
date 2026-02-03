// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type compare, type record, unique } from "@synnaxlabs/x";

export interface Node<K extends record.Key = string> {
  key: K;
  children?: Node<K>[];
}

export interface NodeShape {
  depth: number;
  expanded: boolean;
  hasChildren: boolean;
}

export interface Shape<K extends record.Key = string> {
  keys: K[];
  nodes: NodeShape[];
}

export const shouldExpand = <K extends record.Key = string>(
  node: Node<K>,
  expanded: K[],
): boolean => expanded.includes(node.key);

export interface FlattenProps<K extends record.Key = string> {
  nodes: Node<K>[];
  expanded: K[];
  sort?: compare.Comparator<Node<K>>;
  depth?: number;
  path?: string;
}

export const flatten = <K extends record.Key = string>({
  nodes,
  expanded,
  sort,
  depth = 0,
}: FlattenProps<K>): Shape<K> => {
  const flattened: Shape<K> = { keys: [], nodes: [] };
  if (sort != null) nodes.sort(sort);
  nodes.forEach((node) => {
    const expand = shouldExpand(node, expanded);
    flattened.keys.push(node.key);
    flattened.nodes.push({
      depth,
      expanded: expand,
      hasChildren: node.children != null,
    });
    if (expand && node.children != null) {
      const { keys, nodes } = flatten({
        nodes: node.children,
        expanded,
        sort,
        depth: depth + 1,
      });
      flattened.keys.push(...keys);
      flattened.nodes.push(...nodes);
    }
  });
  return flattened;
};

export interface MoveNodeProps<K extends record.Key = string> {
  tree: Node<K>[];
  destination: K | null;
  keys: K | K[];
}

export const moveNode = <K extends record.Key = string>({
  tree,
  destination,
  keys,
}: MoveNodeProps<K>): Node<K>[] => {
  keys = array.toArray(keys);
  if (destination == null) {
    const nodes = findNodes({ tree, keys });
    return [...nodes, ...tree.filter((node) => !keys.includes(node.key))];
  }
  keys.forEach((key) => {
    const node = findNode({ tree, key });
    if (node == null) return;
    removeNode({ tree, keys: key });
    setNode({ tree, destination, additions: node });
  });
  return tree;
};

export interface RemoveNodeProps<K extends record.Key = string> {
  tree: Node<K>[];
  keys: K | K[];
  parent?: K | null;
}

export const removeNode = <K extends record.Key = string>({
  tree,
  keys,
  parent,
}: RemoveNodeProps<K>): Node<K>[] => {
  keys = array.toArray(keys);
  if (parent !== undefined) {
    if (parent === null) return tree.filter((node) => !keys.includes(node.key));
    const parentNode = findNode({ tree, key: parent });
    if (parentNode == null) return tree;
    parentNode.children = parentNode.children?.filter(
      (child) => !keys.includes(child.key),
    );
    return tree;
  }
  keys.forEach((key) => {
    const index = tree.findIndex((node) => node.key === key);
    if (index !== -1) tree.splice(index, 1);
    else {
      const parent = findNodeParent({ tree, key });
      if (parent != null)
        parent.children = parent.children?.filter((child) => child.key !== key);
    }
  });
  return tree;
};

export interface SetNodeProps<K extends record.Key = string> {
  tree: Node<K>[];
  destination: K | null;
  additions: Node<K> | Node<K>[];
  throwOnMissing?: boolean;
}

export const setNode = <K extends record.Key = string>({
  tree,
  destination,
  additions,
  throwOnMissing = true,
}: SetNodeProps<K>): Node<K>[] => {
  additions = array.toArray(additions);
  const uniqueAdditions = unique.by(additions, (node) => node.key, false);
  const addedKeys = uniqueAdditions.map((node) => node.key);
  if (destination == null)
    return [
      ...uniqueAdditions,
      ...tree.filter((node) => !addedKeys.includes(node.key)),
    ];

  const node = findNode({ tree, key: destination });
  if (node == null) {
    if (throwOnMissing) throw new Error(`Could not find node with key ${destination}`);
    return tree;
  }
  node.children ??= [];

  node.children = [
    ...uniqueAdditions,
    ...node.children.filter((child) => !addedKeys.includes(child.key)),
  ];
  return tree;
};

export interface UpdateNodeProps<K extends record.Key = string> {
  tree: Node<K>[];
  key: K;
  updater: (node: Node<K>) => Node<K>;
  throwOnMissing?: boolean;
}

export const updateNode = <K extends record.Key = string>({
  tree,
  key,
  updater,
  throwOnMissing = true,
}: UpdateNodeProps<K>): Node<K>[] => {
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

interface UpdateNodeChildren<K extends record.Key = string> {
  tree: Node<K>[];
  parent: K;
  updater: (nodes: Node<K>[]) => Node<K>[];
  throwOnMissing?: boolean;
}

export const updateNodeChildren = <K extends record.Key = string>({
  tree,
  parent,
  updater,
  throwOnMissing = true,
}: UpdateNodeChildren<K>): Node<K>[] => {
  const parentNode = findNode({ tree, key: parent });
  if (parentNode == null) {
    if (throwOnMissing) throw new Error(`Could not find node with key ${parent}`);
    return tree;
  }
  parentNode.children = updater(parentNode.children ?? []);
  return tree;
};

export interface FindNodeProps<K extends record.Key = string> {
  tree: Node<K>[];
  key: K;
  depth?: number;
}

export const findNode = <K extends record.Key = string>({
  tree,
  key,
  depth = 0,
}: FindNodeProps<K>): Node<K> | null => {
  for (let i = 0; i < tree.length; i++) {
    const node = tree[i];
    if (node.key === key) return node;
    if (node.children != null) {
      const found = findNode({ tree: node.children, key, depth: depth + 1 });
      if (found != null) return found;
    }
  }
  return null;
};

export interface FindNodesProps<K extends record.Key = string> {
  tree: Node<K>[];
  keys: K[];
}

export const findNodes = <K extends record.Key = string>({
  tree,
  keys,
}: FindNodesProps<K>): Node<K>[] => {
  const nodes: Node<K>[] = [];
  for (const key of keys) {
    const node = findNode({ tree, key });
    if (node != null) nodes.push(node);
  }
  return nodes;
};

export interface FindNodeParentProps<K extends record.Key = string> {
  tree: Node<K>[];
  key: K;
}

export const findNodeParent = <K extends record.Key = string>({
  tree,
  key,
}: FindNodeParentProps<K>): Node<K> | null => {
  for (const node of tree)
    if (node.children != null) {
      if (node.children.some((child) => child.key === key)) return node;
      const found = findNodeParent({ tree: node.children, key });
      if (found != null) return found;
    }
  return null;
};

export const deepCopy = <K extends record.Key = string>(nodes: Node<K>[]): Node<K>[] =>
  nodes.map((node) => ({
    ...node,
    children: node.children != null ? deepCopy(node.children) : undefined,
  }));

export const getDescendants = <K extends record.Key = string>(
  ...node: Node<K>[]
): Node<K>[] => {
  const descendants: Node<K>[] = [];
  node.forEach((n) => {
    descendants.push(n);
    if (n.children != null) descendants.push(...getDescendants(...n.children));
  });
  return descendants;
};

export const filterShape = <K extends record.Key = string>(
  shape: Shape<K>,
  match: (key: K, depth: number) => boolean,
): Shape<K> => {
  const filtered: Shape<K> = { keys: [], nodes: [] };
  shape.keys.forEach((key, index) => {
    if (!match(key, shape.nodes[index].depth)) return;
    filtered.keys.push(key);
    filtered.nodes.push(shape.nodes[index]);
  });
  return filtered;
};

export const getAllNodesOfMinDepth = <K extends record.Key = string>(
  data: Shape<K>,
): K[] => {
  const minDepth = Math.min(...data.nodes.map((node) => node.depth));
  return data.keys.filter((_, index) => data.nodes[index].depth === minDepth);
};

export const getDepth = (key: string, state: Shape<string>) => {
  const index = state.keys.findIndex((k) => k === key);
  return state.nodes[index].depth;
};

export const getNodeShape = <K extends record.Key = string>(
  shape: Shape<K>,
  key: K,
): NodeShape | null => {
  const index = shape.keys.findIndex((k) => k === key);
  return shape.nodes[index];
};
