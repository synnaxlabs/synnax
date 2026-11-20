// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { unique, uuid } from "@synnaxlabs/x";

import { Node } from "@/schematic/node";
import type { Diagram } from "@/vis/diagram";

const { PADDING, TOP_PADDING } = Node.GroupBox;

const isConfig = (c: schematic.ElementConfig | undefined): c is Node.GroupBox.Config =>
  c?.variant === Node.GroupBox.VARIANT;

/** buildParentOf maps the key of each grouped node to the key of its group. */
export const buildParentOf = (
  nodes: readonly schematic.Node[],
): Map<string, string> => {
  const parentOf = new Map<string, string>();
  for (const n of nodes) if (n.parent != null) parentOf.set(n.key, n.parent);
  return parentOf;
};

/** childrenOf maps the key of each group to the keys of the nodes it holds. */
export const childrenOf = (parentOf: Map<string, string>): Map<string, string[]> => {
  const children = new Map<string, string[]>();
  for (const [key, parent] of parentOf) {
    const list = children.get(parent);
    if (list == null) children.set(parent, [key]);
    else list.push(key);
  }
  return children;
};

const rootOf = (parentOf: Map<string, string>, key: string): string => {
  let current = key;
  for (let p = parentOf.get(key); p != null; p = parentOf.get(p)) current = p;
  return current;
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
}

export interface CreateResult {
  actions: schematic.Action[];
  /** selection lists the keys to select: the new group first, then its members. */
  selection: string[];
}

/**
 * createActions builds the one-batch group action: a setNode inserting a group, and a
 * parent set on each outermost group (or ungrouped symbol) the selection resolves to.
 * The box sizes itself from its members at render time. Returns null when fewer than
 * two resolve.
 */
export const createActions = ({
  selected,
  nodes,
}: CreateParams): CreateResult | null => {
  const parentOf = buildParentOf(nodes);
  const memberNodes = resolveOutermost(selected, nodes, parentOf);
  if (memberNodes.length < 2) return null;
  let minX = Infinity;
  let minY = Infinity;
  for (const node of memberNodes) {
    minX = Math.min(minX, node.position.x);
    minY = Math.min(minY, node.position.y);
  }
  const key = uuid.create();
  const members = memberNodes.map((n) => n.key);
  // Not Node.createConfig: it reads every symbol, and this file loads at startup.
  const config = { variant: Node.GroupBox.VARIANT };
  const node = {
    key,
    position: { x: minX - PADDING, y: minY - TOP_PADDING },
    zIndex: -1,
    config,
  };
  return {
    actions: [
      schematic.setNode({ node }),
      ...members.map((m) =>
        schematic.set({ path: ["nodes", m, "parent"], value: key }),
      ),
    ],
    selection: [key, ...withMembers(members, parentOf)],
  };
};

const collectMembers = (
  key: string,
  children: Map<string, string[]>,
  out: Set<string>,
): void => {
  for (const c of children.get(key) ?? []) {
    out.add(c);
    collectMembers(c, children, out);
  }
};

/**
 * withMembers returns the keys plus every selected group's members, recursively.
 */
export const withMembers = (
  keys: readonly string[],
  parentOf: Map<string, string>,
): string[] => {
  const children = childrenOf(parentOf);
  const out = new Set(keys);
  for (const key of keys) collectMembers(key, children, out);
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
  const moved = new Set<string>();
  for (const a of actions)
    if (a.type === "set_node_position") moved.add(a.setNodePosition.key);
  if (moved.size === 0) return actions;
  const { nodes } = current;
  const nodeByKey = new Map(nodes.map((n) => [n.key, n]));
  const children = childrenOf(buildParentOf(nodes));
  const out = [...actions];
  for (const action of actions) {
    if (action.type !== "set_node_position") continue;
    const { key, position } = action.setNodePosition;
    const prev = nodeByKey.get(key);
    if (prev == null || !isConfig(prev.config)) continue;
    const dx = position.x - prev.position.x;
    const dy = position.y - prev.position.y;
    const members = new Set<string>();
    collectMembers(key, children, members);
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
 * lockMembers marks grouped symbols and locked group boxes as non-draggable.
 * Returns the input unchanged when there are no groups.
 */
export const lockMembers = (
  nodes: schematic.Node[],
  parentOf: Map<string, string>,
  configs: Record<string, schematic.ElementConfig>,
): Diagram.Node[] => {
  if (parentOf.size === 0) return nodes;
  const undraggable = (key: string): boolean => {
    if (parentOf.has(key)) return true;
    const config = configs[key];
    return isConfig(config) && config.locked === true;
  };
  return nodes.map((n) => (undraggable(n.key) ? { ...n, draggable: false } : n));
};

// Whether any group containing the key is locked.
const underLocked = (
  key: string,
  parentOf: Map<string, string>,
  configs: Record<string, schematic.ElementConfig>,
): boolean => {
  for (let p = parentOf.get(key); p != null; p = parentOf.get(p)) {
    const config = configs[p];
    if (isConfig(config) && config.locked === true) return true;
  }
  return false;
};

/**
 * drillIn returns the member and, for a nested group, its members. Null when
 * the key is in no group or a containing group is locked.
 */
export const drillIn = (
  key: string,
  parentOf: Map<string, string>,
  configs: Record<string, schematic.ElementConfig>,
): string[] | null => {
  if (!parentOf.has(key) || underLocked(key, parentOf, configs)) return null;
  return withMembers([key], parentOf);
};

/** shielded returns the keys that sit inside a locked group. */
export const shielded = (
  keys: readonly string[],
  parentOf: Map<string, string>,
  configs: Record<string, schematic.ElementConfig>,
): Set<string> => new Set(keys.filter((k) => underLocked(k, parentOf, configs)));

/** closure resolves each key to its outermost group and includes its members. */
export const closure = (
  keys: readonly string[],
  parentOf: Map<string, string>,
): string[] =>
  withMembers(unique.unique(keys.map((k) => rootOf(parentOf, k))), parentOf);

/** canUngroup returns whether the selection includes a group. */
export const canUngroup = (
  selected: readonly string[],
  configs: Record<string, schematic.ElementConfig>,
): boolean => selected.some((key) => isConfig(configs[key]));

export interface UngroupResult {
  actions: schematic.Action[];
  /** freed lists the removed groups' members, nested contents included. */
  freed: string[];
}

/**
 * ungroupActions builds the one-batch ungroup action: removes the selected groups and
 * the immediate parents of selected members. Removing a group moves its members into
 * its own parent. Returns null when the selection touches no group.
 */
export const ungroupActions = (
  selected: readonly string[],
  nodes: readonly schematic.Node[],
): UngroupResult | null => {
  const parentOf = buildParentOf(nodes);
  const groups = new Set(nodes.filter((n) => isConfig(n.config)).map((n) => n.key));
  const targets = new Set<string>();
  for (const key of selected)
    if (groups.has(key)) targets.add(key);
    else {
      const parent = parentOf.get(key);
      if (parent != null) targets.add(parent);
    }
  if (targets.size === 0) return null;
  // A selected group's nested groups are selected too; ungroup only the outermost.
  for (const key of [...targets])
    for (let p = parentOf.get(key); p != null; p = parentOf.get(p))
      if (targets.has(p)) {
        targets.delete(key);
        break;
      }
  const children = childrenOf(parentOf);
  const freed = [...targets].flatMap((key) => children.get(key) ?? []);
  return {
    actions: [...targets].map((key) => schematic.removeNode({ key })),
    freed: withMembers(freed, parentOf),
  };
};
