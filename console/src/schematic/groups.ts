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

/** Expands a set of selected nodes to include all members and containers of any
 * groups touched by the selection. */
export const expandSelectionToGroups = (
  selectedNodes: Diagram.Node[],
  allNodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): Diagram.Node[] => {
  const activeGroupKeys = new Set<string>();
  for (const node of selectedNodes) {
    const p = props[node.key];
    if (p == null) continue;
    if (p.key === "group") activeGroupKeys.add(node.key);
    if (p.groupId != null) activeGroupKeys.add(p.groupId);
  }
  if (activeGroupKeys.size === 0) return selectedNodes;
  const selectedKeys = new Set(selectedNodes.map((n) => n.key));
  const extra = allNodes.filter((node) => {
    if (selectedKeys.has(node.key)) return false;
    const p = props[node.key];
    if (p?.key === "group") return activeGroupKeys.has(node.key);
    return p?.groupId != null && activeGroupKeys.has(p.groupId);
  });
  return extra.length === 0 ? selectedNodes : [...selectedNodes, ...extra];
};

/** When a group container is selected, selects all its member nodes. */
export const propagateGroupSelection = (
  nodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): Diagram.Node[] => {
  const groupKeys = selectedGroupKeys(nodes, props);
  if (groupKeys.size === 0) return nodes;
  let changed = false;
  const result = nodes.map((node) => {
    if (node.selected) return node;
    const p = props[node.key];
    if (p?.groupId != null && groupKeys.has(p.groupId)) {
      changed = true;
      return { ...node, selected: true };
    }
    return node;
  });
  return changed ? result : nodes;
};

const HIGHLIGHTED_CLASS = "pluto-group-box--highlighted";

/** Toggles a highlight class on group container DOM nodes when any of their
 * children are selected. Uses direct DOM class toggling instead of propagating
 * selection state to avoid interfering with multi-select behavior within groups. */
export const syncGroupHighlight = (
  nodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): void => {
  const highlightedGroups = new Set<string>();
  for (const node of nodes) {
    if (!node.selected) continue;
    const p = props[node.key];
    if (p?.groupId != null) highlightedGroups.add(p.groupId);
  }
  for (const node of nodes) {
    const p = props[node.key];
    if (p?.key !== "group") continue;
    const el = document.querySelector(`[data-id="${node.key}"]`);
    if (el == null) continue;
    if (highlightedGroups.has(node.key)) el.classList.add(HIGHLIGHTED_CLASS);
    else el.classList.remove(HIGHLIGHTED_CLASS);
  }
};

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

/** Expands deleted nodes to full groups, then removes them and cleans up props. */
export const cascadeGroupDeletes = (
  prevNodes: Diagram.Node[],
  nextNodes: Diagram.Node[],
  props: Record<string, NodeProps>,
): Diagram.Node[] => {
  const nextKeys = new Set(nextNodes.map((n) => n.key));
  const removed = prevNodes.filter((n) => !nextKeys.has(n.key));
  if (removed.length === 0) return nextNodes;
  const allRemoved = expandSelectionToGroups(removed, prevNodes, props);
  if (allRemoved.length === removed.length) return nextNodes;
  const removeKeys = new Set(allRemoved.map((n) => n.key));
  for (const key of removeKeys) delete props[key];
  return nextNodes.filter((n) => !removeKeys.has(n.key));
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
