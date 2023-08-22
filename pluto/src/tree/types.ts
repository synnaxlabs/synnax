// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Haul } from "@/haul";

export interface Node {
  key: string;
  icon: ReactElement;
  children: Node[];
  canDrop: (items: Haul.Item[]) => boolean;
}

export interface FlattenedNode extends Node {
  depth: number;
}

// VIRTUALIZATION, DRAG AND DROP, CONTEXT MENU
// Basically the idea is that you make calculations in order to flatten the structure
// of the tree into a list of nodes and then pass it into a virtualizer.

// We have a selection state and an expansion state. We assume the key is unique within
// all the nodes, so we can just track both as arrays

export interface TreeProps {
  nodes: Node[];
  selected: string[];
  onSelect: (key: string) => void;
  expanded: string[];
  onExpand: (key: string) => void;
}

export const Tree = 

export const shouldExpand = (node: Node, expanded: string[]): boolean =>
  expanded.includes(node.key) ||
  node.children.some((child) => shouldExpand(child, expanded));

export const flatten = (
  nodes: Node[],
  expanded: string[],
  depth: number = 1
): FlattenedNode[] => {
  const flattened: FlattenedNode[] = [];
  for (const node of nodes) {
    flattened.push({ ...node, depth });
    if (shouldExpand(node, expanded))
      flattened.push(...flatten(node.children, expanded, depth + 1));
  }
  return flattened;
};
