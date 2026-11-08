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
}

/**
 * Returns the visible rows of the tree: each node, then the rows under it when it is
 * expanded. Sorts copies of the levels, never the caller's arrays.
 * @throws if two nodes share a key.
 */
export const flatten = <K extends record.Key = string>({
  nodes,
  expanded,
  sort,
}: FlattenProps<K>): Shape<K> => {
  const flattened: Shape<K> = { keys: [], nodes: [] };
  const seen = new Set<K>();
  const visit = (level: Node<K>[], depth: number): void => {
    (sort == null ? level : level.toSorted(sort)).forEach((node) => {
      if (seen.has(node.key))
        throw new Error(`tree has two nodes with key ${node.key}`);
      seen.add(node.key);
      const expand = shouldExpand(node, expanded);
      flattened.keys.push(node.key);
      flattened.nodes.push({
        depth,
        expanded: expand,
        hasChildren: node.children != null,
      });
      if (expand && node.children != null) visit(node.children, depth + 1);
    });
  };
  visit(nodes, 0);
  return flattened;
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

/**
 * Returns the depth of the node with the given key, or null if the shape has no such
 * key. A shape holds only visible rows, so a node under a contracted parent is absent
 * even though it is still in the tree.
 */
export const getDepth = <K extends record.Key = string>(
  key: K,
  shape: Shape<K>,
): number | null => getNodeShape(shape, key)?.depth ?? null;

/** Orders keys shallowest-first. Keys the shape does not contain sort last. */
export const compareDepth =
  <K extends record.Key = string>(shape: Shape<K>): compare.Comparator<K> =>
  (a, b) => {
    const aDepth = getDepth(a, shape);
    const bDepth = getDepth(b, shape);
    if (aDepth == null) return bDepth == null ? 0 : 1;
    if (bDepth == null) return -1;
    return aDepth - bDepth;
  };

export const getNodeShape = <K extends record.Key = string>(
  shape: Shape<K>,
  key: K,
): NodeShape | null => {
  const index = shape.keys.findIndex((k) => k === key);
  return index === -1 ? null : shape.nodes[index];
};
