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
  for (const node of dragging) {
    const prev = prevByKey.get(node.key);
    if (prev == null) continue;
    delta = {
      x: node.position.x - prev.position.x,
      y: node.position.y - prev.position.y,
    };
    break;
  }
  if (delta == null || (delta.x === 0 && delta.y === 0)) return nodes;

  const activeGroupKeys = new Set<string>();
  for (const node of dragging) {
    const p = props[node.key];
    if (p == null) continue;
    const gk = groupKeyOf(node.key, p);
    if (gk != null) activeGroupKeys.add(gk);
  }
  if (activeGroupKeys.size === 0) return nodes;

  const dx = delta.x;
  const dy = delta.y;
  const draggingKeys = new Set(dragging.map((n) => n.key));
  let changed = false;
  const result = nodes.map((node) => {
    if (draggingKeys.has(node.key)) return node;
    const p = props[node.key];
    if (p == null) return node;
    const gk = groupKeyOf(node.key, p);
    if (gk == null || !activeGroupKeys.has(gk)) return node;
    changed = true;
    return {
      ...node,
      position: { x: node.position.x + dx, y: node.position.y + dy },
    };
  });
  return changed ? result : nodes;
};

/** remapGroupIds updates groupId references in props after a paste operation. If a
 * member is pasted without its group container, the stale groupId is cleared so the
 * pasted node doesn't inadvertently join the original group. */
export const remapGroupIds = (
  props: Record<string, NodeProps>,
  keyMap: Record<string, string>,
): void => {
  for (const newKey of Object.values(keyMap)) {
    const p = props[newKey];
    if (p?.groupId == null) continue;
    if (keyMap[p.groupId] != null) p.groupId = keyMap[p.groupId];
    else delete p.groupId;
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

/** Removes children of deleted group nodes from the node list and cleans up props. */
export const cascadeGroupDeletes = (
  prevNodes: Diagram.Node[],
  nextNodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): Diagram.Node[] => {
  const nextKeys = new Set(nextNodes.map((n) => n.key));
  const removedGroupKeys = new Set<string>();
  for (const node of prevNodes)
    if (!nextKeys.has(node.key) && props[node.key]?.key === "group")
      removedGroupKeys.add(node.key);
  if (removedGroupKeys.size === 0) return nextNodes;
  for (const key of removedGroupKeys) delete props[key];
  return nextNodes.filter((n) => {
    const gid = props[n.key]?.groupId;
    if (gid != null && removedGroupKeys.has(gid)) {
      delete props[n.key];
      return false;
    }
    return true;
  });
};

/** Removes empty group containers, ungroups single-member groups, and recalculates
 * bounding boxes for surviving groups in one pass. */
export const auditGroups = (
  nodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): Diagram.Node[] => {
  const members = new Map<string, Diagram.Node[]>();
  for (const node of nodes) {
    const gid = props[node.key]?.groupId;
    if (gid == null) continue;
    const list = members.get(gid);
    if (list != null) list.push(node);
    else members.set(gid, [node]);
  }
  const toRemove = new Set<string>();
  const toResize = new Set<string>();
  for (const node of nodes) {
    if (props[node.key]?.key !== "group") continue;
    const count = members.get(node.key)?.length ?? 0;
    if (count <= 1) toRemove.add(node.key);
    else toResize.add(node.key);
  }
  if (toRemove.size === 0 && toResize.size === 0) return nodes;
  for (const node of nodes) {
    const p = props[node.key];
    if (p?.groupId != null && toRemove.has(p.groupId)) delete p.groupId;
  }
  for (const key of toRemove) delete props[key];
  const result = toRemove.size > 0 ? nodes.filter((n) => !toRemove.has(n.key)) : nodes;
  for (const groupKey of toResize) {
    const memberNodes = members.get(groupKey);
    if (memberNodes == null) continue;
    const { position, dimensions } = calculateGroupBoundingBox(memberNodes);
    const groupNode = result.find((n) => n.key === groupKey);
    if (groupNode != null) groupNode.position = position;
    const p = props[groupKey];
    if (p != null) (p as Record<string, unknown>).dimensions = dimensions;
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
