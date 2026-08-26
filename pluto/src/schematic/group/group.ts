// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { compare, type dimensions, type record, uuid } from "@synnaxlabs/x";

import { Node } from "@/schematic/node";
import { type Diagram } from "@/vis/diagram";

const PADDING = 30;

/** Measure returns a symbol's rendered size, or null when it is not mounted. */
export type Measure = (key: string) => dimensions.Dimensions | null;

const isConfig = (c: record.Unknown | undefined): c is Node.GroupBox.Config =>
  c?.variant === Node.GroupBox.VARIANT;

// Membership forms a forest: a group's config lists its member symbols' keys, and
// a group is itself a symbol, so it can be a member of another group. parentOf is
// that relation inverted.
export const buildParentOf = (
  configs: Record<string, record.Unknown>,
): Map<string, string> => {
  const parentOf = new Map<string, string>();
  for (const [key, config] of Object.entries(configs)) {
    if (!isConfig(config)) continue;
    // First wins: corrupt data claiming a key for two groups keeps the first.
    for (const m of config.members) if (!parentOf.has(m)) parentOf.set(m, key);
  }
  return parentOf;
};

// The visited set guards against cycles in corrupt or hand-edited data.
const rootOf = (parentOf: Map<string, string>, key: string): string => {
  const visited = new Set<string>();
  let current = key;
  while (true) {
    const parent = parentOf.get(current);
    if (parent == null || visited.has(parent)) return current;
    visited.add(current);
    current = parent;
  }
};

// Selected keys resolve to their outermost group (or themselves when ungrouped):
// grouping nests whole groups, and no symbol ever gains two parents.
const resolveOutermost = (
  selected: readonly string[],
  nodes: readonly schematic.Node[],
  parentOf: Map<string, string>,
): schematic.Node[] => {
  const nodeByKey = new Map(nodes.map((n) => [n.key, n]));
  const keys = new Set<string>();
  for (const key of selected) if (nodeByKey.has(key)) keys.add(rootOf(parentOf, key));
  return [...keys].map((k) => nodeByKey.get(k)).filter((n) => n != null);
};

/** canGroup returns whether the selection resolves to two or more outermost symbols. */
export const canGroup = (
  selected: readonly string[],
  nodes: readonly schematic.Node[],
  parentOf: Map<string, string>,
): boolean => resolveOutermost(selected, nodes, parentOf).length >= 2;

export interface CreateParams {
  selected: readonly string[];
  nodes: readonly schematic.Node[];
  configs: Record<string, record.Unknown>;
  measure: Measure;
}

export interface CreateResult {
  actions: schematic.Action[];
  /** selection lists the keys to select: the new group first, then its members. */
  selection: string[];
}

/**
 * createActions builds the one-batch group action: a setNode inserting a group
 * whose members are the outermost groups (or ungrouped symbols) the selection
 * resolves to, positioned on their bounding box. Returns null when fewer than
 * two resolve or one is not measurable.
 */
export const createActions = ({
  selected,
  nodes,
  configs,
  measure,
}: CreateParams): CreateResult | null => {
  const memberNodes = resolveOutermost(selected, nodes, buildParentOf(configs));
  if (memberNodes.length < 2) return null;
  let minX = Infinity;
  let minY = Infinity;
  let maxX = -Infinity;
  let maxY = -Infinity;
  for (const node of memberNodes) {
    const dims = measure(node.key);
    if (dims == null) return null;
    minX = Math.min(minX, node.position.x);
    minY = Math.min(minY, node.position.y);
    maxX = Math.max(maxX, node.position.x + dims.width);
    maxY = Math.max(maxY, node.position.y + dims.height);
  }
  const key = uuid.create();
  const members = memberNodes.map((n) => n.key);
  const config: Node.GroupBox.Config = {
    ...Node.GroupBox.defaultConfig(),
    members,
    dimensions: {
      width: maxX - minX + 2 * PADDING,
      height: maxY - minY + 2 * PADDING,
    },
  };
  const node = {
    key,
    position: { x: minX - PADDING, y: minY - PADDING },
    zIndex: -1,
  };
  return {
    actions: [schematic.setNode({ node, config })],
    selection: [key, ...withMembers(members, configs)],
  };
};

const collectMembers = (
  key: string,
  configs: Record<string, record.Unknown>,
  out: Set<string>,
): void => {
  const config = configs[key];
  if (!isConfig(config)) return;
  for (const m of config.members) {
    if (out.has(m)) continue;
    out.add(m);
    collectMembers(m, configs, out);
  }
};

/**
 * remapMembers rewrites a pasted group config's members onto the pasted keys,
 * dropping members that were not pasted. Non-group configs pass through.
 */
export const remapMembers = (
  config: record.Unknown | undefined,
  remap: Record<string, string>,
): record.Unknown | undefined => {
  if (!isConfig(config)) return config;
  const members = config.members.map((m) => remap[m]).filter((m) => m != null);
  return { ...config, members };
};

/**
 * withMembers returns the keys plus every selected group's members, recursively.
 */
export const withMembers = (
  keys: readonly string[],
  configs: Record<string, record.Unknown>,
): string[] => {
  const out = new Set(keys);
  for (const key of keys) collectMembers(key, configs, out);
  return [...out];
};

/**
 * fanOutMoves applies a moved group's delta to every symbol inside it. Keys
 * already moved in the batch are skipped. Zero deltas still fan out, so every
 * drag frame targets the same keys and coalesces into one undo step.
 */
export const fanOutMoves = (
  current: schematic.Schematic,
  actions: schematic.Action[],
): schematic.Action[] => {
  const { nodes, configs } = current;
  const moved = new Set<string>();
  for (const a of actions)
    if (a.type === "set_node_position") moved.add(a.setNodePosition.key);
  if (moved.size === 0) return actions;
  const nodeByKey = new Map(nodes.map((n) => [n.key, n]));
  const out = [...actions];
  for (const action of actions) {
    if (action.type !== "set_node_position") continue;
    const { key, position } = action.setNodePosition;
    if (!isConfig(configs[key])) continue;
    const prev = nodeByKey.get(key);
    if (prev == null) continue;
    const dx = position.x - prev.position.x;
    const dy = position.y - prev.position.y;
    const members = new Set<string>();
    collectMembers(key, configs, members);
    for (const m of members) {
      if (moved.has(m)) continue;
      const node = nodeByKey.get(m);
      if (node == null) continue;
      moved.add(m);
      out.push(
        schematic.setNodePosition({
          key: m,
          position: { x: node.position.x + dx, y: node.position.y + dy },
        }),
      );
    }
  }
  return out;
};

/**
 * lockMembers marks every symbol inside a group as non-draggable; a group moves
 * its members via fanOutMoves. Returns the input array unchanged when there
 * are no groups.
 */
export const lockMembers = (
  nodes: schematic.Node[],
  parentOf: Map<string, string>,
): Diagram.Node[] => {
  if (parentOf.size === 0) return nodes;
  return nodes.map((n) => (parentOf.has(n.key) ? { ...n, draggable: false } : n));
};

/** lockedKeys returns the grouped symbols in keys whose group is absent. */
export const lockedKeys = (
  keys: readonly string[],
  parentOf: Map<string, string>,
): string[] => {
  const keySet = new Set(keys);
  return keys.filter((k) => {
    const parent = parentOf.get(k);
    return parent != null && !keySet.has(parent);
  });
};

/** canUngroup returns whether the selection includes a group. */
export const canUngroup = (
  selected: readonly string[],
  configs: Record<string, record.Unknown>,
): boolean => selected.some((key) => isConfig(configs[key]));

export interface UngroupResult {
  actions: schematic.Action[];
  /** freed lists the removed groups' members, nested contents included. */
  freed: string[];
}

// Resolves an ungrouped group to its members, recursively.
const resolveUngrouped = (
  key: string,
  targets: Set<string>,
  configs: Record<string, record.Unknown>,
  visited: Set<string>,
): string[] => {
  if (!targets.has(key)) return [key];
  if (visited.has(key)) return [];
  visited.add(key);
  const config = configs[key];
  if (!isConfig(config)) return [];
  return config.members.flatMap((m) => resolveUngrouped(m, targets, configs, visited));
};

/**
 * ungroupActions builds the one-batch ungroup action: removes the selected
 * groups and the immediate parents of selected members, promoting a removed
 * group's members into the closest enclosing group that remains. Returns null
 * when the selection touches no group.
 */
export const ungroupActions = (
  selected: readonly string[],
  configs: Record<string, record.Unknown>,
): UngroupResult | null => {
  const parentOf = buildParentOf(configs);
  const targets = new Set<string>();
  for (const key of selected)
    if (isConfig(configs[key])) targets.add(key);
    else {
      const parent = parentOf.get(key);
      if (parent != null) targets.add(parent);
    }
  if (targets.size === 0) return null;
  // A selected group's nested groups are selected too; ungroup only the outermost.
  const nested = [...targets].filter((key) => {
    const visited = new Set<string>();
    let parent = parentOf.get(key);
    while (parent != null && !visited.has(parent)) {
      if (targets.has(parent)) return true;
      visited.add(parent);
      parent = parentOf.get(parent);
    }
    return false;
  });
  for (const key of nested) targets.delete(key);
  const actions: schematic.Action[] = [];
  for (const [key, config] of Object.entries(configs)) {
    if (!isConfig(config) || targets.has(key)) continue;
    const next = config.members.flatMap((m) =>
      resolveUngrouped(m, targets, configs, new Set()),
    );
    if (!compare.arraysEqual(next, config.members))
      actions.push(schematic.setConfig({ key, config: { members: next } }));
  }
  const freed: string[] = [];
  for (const key of targets) {
    actions.push(schematic.removeNode({ key }));
    const config = configs[key];
    if (!isConfig(config)) continue;
    for (const m of config.members) if (!targets.has(m)) freed.push(m);
  }
  return { actions, freed: withMembers(freed, configs) };
};
