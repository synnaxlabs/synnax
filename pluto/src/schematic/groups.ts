// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type record, type xy } from "@synnaxlabs/x";

export const GROUP_VARIANT = "groupBox";
export const GROUP_PADDING = 30;

const isGroupContainer = (
  key: string,
  configs: Record<string, record.Unknown>,
): boolean =>
  (configs[key] as { variant?: string } | undefined)?.variant === GROUP_VARIANT;

/** Group container's own key; member's groupId; undefined if loose. */
export const groupKeyOf = (
  node: schematic.Node,
  configs: Record<string, record.Unknown>,
): string | undefined => {
  if (isGroupContainer(node.key, configs)) return node.key;
  return node.groupId ?? undefined;
};

/** Filters selectedKeys to just the group container keys. */
export const selectedGroupKeys = (
  selectedKeys: readonly string[],
  configs: Record<string, record.Unknown>,
): Set<string> => {
  const out = new Set<string>();
  for (const k of selectedKeys) if (isGroupContainer(k, configs)) out.add(k);
  return out;
};

/** Keys of nodes whose groupId matches groupKey. */
export const findGroupMembers = (
  groupKey: string,
  allNodes: readonly schematic.Node[],
): string[] => allNodes.filter((n) => n.groupId === groupKey).map((n) => n.key);

/** Adds members for every selected container, and containers for every selected
 * member. Idempotent. */
export const expandSelectionToGroups = (
  selectedKeys: readonly string[],
  allNodes: readonly schematic.Node[],
  configs: Record<string, record.Unknown>,
): string[] => {
  const activeGroupKeys = new Set<string>();
  const nodeByKey = new Map(allNodes.map((n) => [n.key, n]));
  for (const k of selectedKeys) {
    if (isGroupContainer(k, configs)) activeGroupKeys.add(k);
    const node = nodeByKey.get(k);
    if (node?.groupId != null) activeGroupKeys.add(node.groupId);
  }
  if (activeGroupKeys.size === 0) return [...selectedKeys];
  const result = new Set(selectedKeys);
  for (const node of allNodes) {
    if (result.has(node.key)) continue;
    if (isGroupContainer(node.key, configs)) {
      if (activeGroupKeys.has(node.key)) result.add(node.key);
    } else if (node.groupId != null && activeGroupKeys.has(node.groupId))
      result.add(node.key);
  }
  return [...result];
};

/** Group containers with fewer than minMembers members; cascade-delete targets. */
export const findOrphanedGroups = (
  allNodes: readonly schematic.Node[],
  configs: Record<string, record.Unknown>,
  minMembers = 1,
): string[] => {
  const memberCounts = new Map<string, number>();
  for (const n of allNodes)
    if (n.groupId != null)
      memberCounts.set(n.groupId, (memberCounts.get(n.groupId) ?? 0) + 1);
  const out: string[] = [];
  for (const n of allNodes) {
    if (!isGroupContainer(n.key, configs)) continue;
    if ((memberCounts.get(n.key) ?? 0) < minMembers) out.push(n.key);
  }
  return out;
};

/** Bounding box (top-left + dimensions) enclosing members with GROUP_PADDING. */
export const computeGroupBoundingBox = (
  memberNodes: readonly schematic.Node[],
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

/** Extends position changes to co-grouped non-dragged members so groups drag as
 * a unit. */
export const propagateGroupDrag = (
  changes: readonly { key: string; position: xy.XY }[],
  allNodes: readonly schematic.Node[],
  configs: Record<string, record.Unknown>,
): { key: string; position: xy.XY }[] => {
  if (changes.length === 0) return [];
  const nodeByKey = new Map(allNodes.map((n) => [n.key, n]));
  const changedKeys = new Set(changes.map((c) => c.key));
  let dx = 0;
  let dy = 0;
  const activeGroupKeys = new Set<string>();
  for (const change of changes) {
    const prev = nodeByKey.get(change.key);
    if (prev == null) continue;
    if (dx === 0 && dy === 0) {
      dx = change.position.x - prev.position.x;
      dy = change.position.y - prev.position.y;
    }
    const gk = groupKeyOf(prev, configs);
    if (gk != null) activeGroupKeys.add(gk);
  }
  if (activeGroupKeys.size === 0 || (dx === 0 && dy === 0)) return [...changes];

  const result: { key: string; position: xy.XY }[] = [...changes];
  for (const node of allNodes) {
    if (changedKeys.has(node.key)) continue;
    const gk = groupKeyOf(node, configs);
    if (gk == null || !activeGroupKeys.has(gk)) continue;
    result.push({
      key: node.key,
      position: { x: node.position.x + dx, y: node.position.y + dy },
    });
  }
  return result;
};

/** Remaps node.groupId via keyMap. Drops groupId if container is not in keyMap
 * (avoids dangling references when only members are pasted). */
export const remapGroupId = (
  node: schematic.Node,
  keyMap: Readonly<Record<string, string>>,
): schematic.Node => {
  if (node.groupId == null) return node;
  const mapped = keyMap[node.groupId];
  if (mapped != null) return { ...node, groupId: mapped };
  const { groupId: _drop, ...rest } = node;
  return rest;
};
