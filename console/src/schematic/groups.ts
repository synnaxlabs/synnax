// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Diagram } from "@synnaxlabs/pluto";
import { type xy } from "@synnaxlabs/x";

import { type NodeProps } from "@/schematic/types";

const GROUP_PADDING = 30;

/** Returns the keys of selected nodes that are group containers. */
export const selectedGroupKeys = (
  nodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): Set<string> =>
  new Set(
    nodes.filter((n) => n.selected && props[n.key]?.key === "group").map((n) => n.key),
  );

/** Returns the group key a node belongs to, or undefined if ungrouped. */
export const groupKeyOf = (nodeKey: string, props: NodeProps): string | undefined =>
  props.key === "group" ? nodeKey : props.groupId;

/** Applies the drag delta of a single group member to all other members. */
export const propagateGroupDrag = (
  nodes: Diagram.Node[],
  prevNodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): Diagram.Node[] => {
  const dragging = nodes.filter((n) => "dragging" in n && !!n.dragging);
  if (dragging.length === 0) return nodes;

  const prevByKey = new Map(prevNodes.map((n) => [n.key, n]));
  let delta: xy.XY | null = null;
  let activeGroupKey: string | null = null;

  for (const node of dragging) {
    const p = props[node.key];
    if (p == null) continue;
    const gk = groupKeyOf(node.key, p);
    if (gk == null) continue;
    const prev = prevByKey.get(node.key);
    if (prev == null) continue;
    delta = {
      x: node.position.x - prev.position.x,
      y: node.position.y - prev.position.y,
    };
    activeGroupKey = gk;
    break;
  }

  if (delta == null || activeGroupKey == null || (delta.x === 0 && delta.y === 0))
    return nodes;

  const dx = delta.x;
  const dy = delta.y;
  const draggingKeys = new Set(dragging.map((n) => n.key));
  let changed = false;
  const result = nodes.map((node) => {
    if (draggingKeys.has(node.key)) return node;
    const p = props[node.key];
    if (p == null) return node;
    if (groupKeyOf(node.key, p) !== activeGroupKey) return node;
    changed = true;
    return {
      ...node,
      position: { x: node.position.x + dx, y: node.position.y + dy },
    };
  });
  return changed ? result : nodes;
};

/** remapGroupIds updates groupId references in props after a paste operation. */
export const remapGroupIds = (
  props: Record<string, NodeProps>,
  keyMap: Record<string, string>,
): void => {
  for (const newKey of Object.values(keyMap)) {
    const p = props[newKey];
    if (p?.groupId != null && keyMap[p.groupId] != null) p.groupId = keyMap[p.groupId];
  }
};

/** Resolves the alignment key for a node (parent if grouped, itself otherwise. */
export const resolveAlignmentKey = (
  elKey: string,
  props: Record<string, NodeProps>,
  nodes: Diagram.Node[],
  elPosition: xy.XY,
): { key: string; position: xy.XY } => {
  const p = props[elKey];
  const gk = p != null ? groupKeyOf(elKey, p) : undefined;
  if (gk != null && gk !== elKey) {
    const groupNode = nodes.find((n) => n.key === gk);
    if (groupNode != null) return { key: gk, position: groupNode.position };
  }
  return { key: elKey, position: elPosition };
};

/** Expands group box positions into positions for all member nodes. */
export const expandGroupPositions = (
  positions: [string, xy.XY][],
  nodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): [string, xy.XY][] => {
  const result: [string, xy.XY][] = [];
  for (const [key, newPos] of positions) {
    const p = props[key];
    if (p?.key === "group") {
      const groupNode = nodes.find((n) => n.key === key);
      if (groupNode == null) continue;
      const delta = {
        x: newPos.x - groupNode.position.x,
        y: newPos.y - groupNode.position.y,
      };
      result.push([key, newPos]);
      for (const node of nodes)
        if (props[node.key]?.groupId === key)
          result.push([
            node.key,
            { x: node.position.x + delta.x, y: node.position.y + delta.y },
          ]);
    } else result.push([key, newPos]);
  }
  return result;
};

/** calculateGroupBoundingBox computes the bounding box for a set of member nodes. */
export const calculateGroupBoundingBox = (
  memberNodes: Diagram.Node[],
): { position: xy.XY; dimensions: { width: number; height: number } } => {
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const node of memberNodes) {
    const w = node.measured?.width ?? 0;
    const h = node.measured?.height ?? 0;
    minX = Math.min(minX, node.position.x);
    minY = Math.min(minY, node.position.y);
    maxX = Math.max(maxX, node.position.x + w);
    maxY = Math.max(maxY, node.position.y + h);
  }
  return {
    position: { x: minX - GROUP_PADDING, y: minY - GROUP_PADDING },
    dimensions: {
      width: maxX - minX + 2 * GROUP_PADDING,
      height: maxY - minY + 2 * GROUP_PADDING,
    },
  };
};
