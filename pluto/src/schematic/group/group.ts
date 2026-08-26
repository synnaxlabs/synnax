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

const PADDING = 30;

/** Measure returns a symbol's rendered size, or null when it is not mounted. */
export type Measure = (key: string) => dimensions.Dimensions | null;

const isConfig = (c: record.Unknown | undefined): c is Node.GroupBox.Config =>
  c?.variant === Node.GroupBox.VARIANT;

// Membership forms a forest: a container's config lists its member symbols' keys,
// and a container is itself a symbol, so it can be a member of another container.
// parentOf is that relation inverted.
const buildParentOf = (
  configs: Record<string, record.Unknown>,
): Map<string, string> => {
  const parentOf = new Map<string, string>();
  for (const [key, config] of Object.entries(configs)) {
    if (!isConfig(config)) continue;
    // First wins: corrupt data claiming a key for two containers keeps the first.
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

export interface CreateParams {
  selected: readonly string[];
  nodes: readonly schematic.Node[];
  configs: Record<string, record.Unknown>;
  measure: Measure;
}

export interface CreateResult {
  actions: schematic.Action[];
  /** key is the new container's key. */
  key: string;
}

/**
 * createActions builds the single action batch that groups the selection: one
 * setNode inserting a container whose members are the outermost containers (or
 * loose symbols) the selection resolves to, positioned on their bounding box.
 * Returns null when fewer than two resolve or one is not measurable.
 */
export const createActions = ({
  selected,
  nodes,
  configs,
  measure,
}: CreateParams): CreateResult | null => {
  const nodeByKey = new Map(nodes.map((n) => [n.key, n]));
  const sel = selected.filter((k) => nodeByKey.has(k));
  const parentOf = buildParentOf(configs);
  // Selected keys resolve to their outermost container (or themselves when
  // parentless): grouping nests whole groups, and no symbol ever gains two parents.
  const memberNodes = [...new Set(sel.map((k) => rootOf(parentOf, k)))]
    .map((k) => nodeByKey.get(k))
    .filter((n) => n != null);
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
  const config: Node.GroupBox.Config = {
    ...Node.GroupBox.defaultConfig(),
    members: memberNodes.map((n) => n.key),
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
  return { actions: [schematic.setNode({ node, config })], key };
};

export interface UngroupResult {
  actions: schematic.Action[];
  /** freed lists the direct members of the dissolved containers. */
  freed: string[];
}

// Expands a member to its surviving representation: dissolved containers are
// replaced by their own members, recursively.
const expandDissolved = (
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
  return config.members.flatMap((m) => expandDissolved(m, targets, configs, visited));
};

/**
 * ungroupActions builds the single action batch that dissolves the containers the
 * selection resolves to: selected containers and the immediate parents of selected
 * members. A dissolved container's members are promoted into its nearest surviving
 * ancestor. Returns null when the selection touches no group.
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
  const actions: schematic.Action[] = [];
  for (const [key, config] of Object.entries(configs)) {
    if (!isConfig(config) || targets.has(key)) continue;
    const next = config.members.flatMap((m) =>
      expandDissolved(m, targets, configs, new Set()),
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
  return { actions, freed };
};
