// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type record } from "@synnaxlabs/x";
import { type Draft, produce } from "immer";

import {
  type Action,
  addEdge,
  type AddEdgePayload,
  removeEdge,
  type RemoveEdgePayload,
  removeNode,
  type RemoveNodePayload,
  setConfig,
  type SetConfigPayload,
  setNode,
  type SetNodeMeasuredPayload,
  type SetNodePayload,
  setNodePosition,
  type SetNodePositionPayload,
} from "@/schematic/actions.gen";
import { type Schematic } from "@/schematic/types.gen";

type Handler<P> = (state: Draft<Schematic>, payload: P) => Action[];

// Captured pre-state values must be detached from the Immer draft before being
// stored in an inverse action. The draft proxies are revoked after produce()
// returns; if an inverse action holds a draft reference, applying it later
// throws "cannot perform 'get' on a revoked proxy". JSON round-trip is the
// simplest correct snapshot for the JSON-shaped data the schema permits.
const snapshot = <T>(v: T): T => JSON.parse(JSON.stringify(v)) as T;

const setNodePositionH: Handler<SetNodePositionPayload> = (state, payload) => {
  const node = state.nodes.find((n) => n.key === payload.key);
  if (node == null) return [];
  const oldPosition = { x: node.position.x, y: node.position.y };
  node.position = payload.position;
  return [setNodePosition({ key: payload.key, position: oldPosition })];
};

const setNodeMeasuredH: Handler<SetNodeMeasuredPayload> = (state, payload) => {
  const node = state.nodes.find((n) => n.key === payload.key);
  if (node != null) node.measured = payload.measured;
  return [];
};

const setNodeH: Handler<SetNodePayload> = (state, payload) => {
  const idx = state.nodes.findIndex((n) => n.key === payload.node.key);
  if (idx === -1) {
    state.nodes.push(payload.node);
    if (payload.config != null) state.configs[payload.node.key] = payload.config;
    return [removeNode({ key: payload.node.key })];
  }
  const oldNode = snapshot(state.nodes[idx]);
  const oldConfigRaw = state.configs[payload.node.key];
  const oldConfig =
    oldConfigRaw != null ? snapshot(oldConfigRaw as record.Unknown) : undefined;
  state.nodes[idx] = payload.node;
  if (payload.config != null) state.configs[payload.node.key] = payload.config;
  return [
    setNode(
      oldConfig != null
        ? { node: oldNode, config: oldConfig }
        : { node: oldNode, config: undefined },
    ),
  ];
};

const removeNodeH: Handler<RemoveNodePayload> = (state, payload) => {
  const idx = state.nodes.findIndex((n) => n.key === payload.key);
  if (idx === -1) return [];
  const oldNode = snapshot(state.nodes[idx]);
  const oldConfigRaw = state.configs[payload.key];
  const oldConfig =
    oldConfigRaw != null ? snapshot(oldConfigRaw as record.Unknown) : undefined;
  state.nodes.splice(idx, 1);
  delete state.configs[payload.key];
  return [
    setNode(
      oldConfig != null
        ? { node: oldNode, config: oldConfig }
        : { node: oldNode, config: undefined },
    ),
  ];
};

const addEdgeH: Handler<AddEdgePayload> = (state, payload) => {
  if (state.edges.some((e) => e.key === payload.edge.key)) return [];
  state.edges.push(payload.edge);
  return [removeEdge({ key: payload.edge.key })];
};

const removeEdgeH: Handler<RemoveEdgePayload> = (state, payload) => {
  const idx = state.edges.findIndex((e) => e.key === payload.key);
  if (idx === -1) return [];
  const oldEdge = snapshot(state.edges[idx]);
  state.edges.splice(idx, 1);
  return [addEdge({ edge: oldEdge })];
};

// The inverse of SetConfig is imperfect for keys the action newly introduces:
// SetConfig only merges, so it cannot remove keys that did not previously
// exist. The inverse here restores values for keys that DID exist before the
// merge; keys added by the action remain on undo as phantom fields. A future
// ReplaceConfig action can close the gap by enabling wholesale replacement.
const setConfigH: Handler<SetConfigPayload> = (state, payload) => {
  const existingRaw = state.configs[payload.key];
  if (existingRaw != null) {
    const existing = snapshot(existingRaw as record.Unknown);
    const restoreFields: record.Unknown = {};
    for (const k of Object.keys(payload.config))
      if (existing[k] !== undefined) restoreFields[k] = existing[k];
    state.configs[payload.key] = { ...existing, ...payload.config };
    if (Object.keys(restoreFields).length === 0) return [];
    return [setConfig({ key: payload.key, config: restoreFields })];
  }
  let cfg = payload.config;
  const edge = state.edges.find((e) => e.key === payload.key);
  if (edge != null) {
    const srcCfg = state.configs[edge.source.node] as
      | { color?: color.Crude }
      | undefined;
    if (srcCfg?.color != null && !color.isZero(srcCfg.color))
      cfg = { ...cfg, color: srcCfg.color };
  }
  state.configs[payload.key] = cfg;
  return [];
};

export const reduce = (state: Draft<Schematic>, action: Action): Action[] => {
  switch (action.type) {
    case "set_node_position":
      return setNodePositionH(state, action.setNodePosition);
    case "set_node_measured":
      return setNodeMeasuredH(state, action.setNodeMeasured);
    case "set_node":
      return setNodeH(state, action.setNode);
    case "remove_node":
      return removeNodeH(state, action.removeNode);
    case "add_edge":
      return addEdgeH(state, action.addEdge);
    case "remove_edge":
      return removeEdgeH(state, action.removeEdge);
    case "set_config":
      return setConfigH(state, action.setConfig);
  }
};

export interface ReduceAllResult {
  next: Schematic;
  inverse: Action[];
}

export const reduceAll = (state: Schematic, actions: Action[]): ReduceAllResult => {
  const inverse: Action[] = [];
  const next = produce(state, (draft) => {
    for (const action of actions) {
      const inv = reduce(draft, action);
      if (inv.length > 0) inverse.unshift(...inv);
    }
  });
  return { next, inverse };
};

export const isUndoable = (action: Action): boolean =>
  action.type !== "set_node_measured";
