// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type record } from "@synnaxlabs/x";

import { actions } from "@/actions";
import {
  type Action,
  addEdge,
  createReduceAll,
  type Handlers,
  removeEdge,
  removeNode,
  rename,
  setConfig,
  setNode,
  setNodePosition,
} from "@/schematic/actions.gen";

const handlers: Handlers = {
  create: (state, payload) => {
    Object.assign(state, payload.schematic);
    return { inverse: [], targets: [payload.schematic.key] };
  },

  rename: (state, payload) => {
    const oldName = state.name;
    state.name = payload.name;
    return {
      inverse: [rename({ name: oldName })],
      targets: [state.key],
    };
  },
  setNodePosition: (state, payload) => {
    const node = state.nodes.find((n) => n.key === payload.key);
    if (node == null) return actions.NO_OP_RESULT;
    const oldPosition = { x: node.position.x, y: node.position.y };
    node.position = payload.position;
    return {
      inverse: [setNodePosition({ key: payload.key, position: oldPosition })],
      targets: [payload.key],
    };
  },
  setNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.node.key);
    if (idx === -1) {
      state.nodes.push(payload.node);
      if (payload.config != null) state.configs[payload.node.key] = payload.config;
      return {
        inverse: [removeNode({ key: payload.node.key })],
        targets: [payload.node.key],
      };
    }
    const oldNode = actions.snapshotDraft(state.nodes[idx]);
    const oldConfigRaw = state.configs[payload.node.key];
    const oldConfig =
      oldConfigRaw != null ? actions.snapshotDraft(oldConfigRaw) : undefined;
    state.nodes[idx] = payload.node;
    if (payload.config != null) state.configs[payload.node.key] = payload.config;
    return {
      inverse: [
        setNode(
          oldConfig != null
            ? { node: oldNode, config: oldConfig }
            : { node: oldNode, config: undefined },
        ),
      ],
      targets: [payload.node.key],
    };
  },
  removeNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldNode = actions.snapshotDraft(state.nodes[idx]);
    const oldConfigRaw = state.configs[payload.key];
    const oldConfig =
      oldConfigRaw != null ? actions.snapshotDraft(oldConfigRaw) : undefined;
    state.nodes.splice(idx, 1);
    delete state.configs[payload.key];
    return {
      inverse: [
        setNode(
          oldConfig != null
            ? { node: oldNode, config: oldConfig }
            : { node: oldNode, config: undefined },
        ),
      ],
      targets: [payload.key],
    };
  },
  addEdge: (state, payload) => {
    if (state.edges.some((e) => e.key === payload.edge.key))
      return actions.NO_OP_RESULT;
    state.edges.push(payload.edge);
    return {
      inverse: [removeEdge({ key: payload.edge.key })],
      targets: [payload.edge.key],
    };
  },

  removeEdge: (state, payload) => {
    const idx = state.edges.findIndex((e) => e.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldEdge = actions.snapshotDraft(state.edges[idx]);
    state.edges.splice(idx, 1);
    return {
      inverse: [addEdge({ edge: oldEdge })],
      targets: [payload.key],
    };
  },
  // The inverse of SetConfig is imperfect for keys the action newly
  // introduces: SetConfig only merges, so it cannot remove keys that did not
  // previously exist. The inverse here restores values for keys that DID
  // exist before the merge; keys added by the action remain on undo as
  // phantom fields. A future ReplaceConfig action can close the gap by
  // enabling wholesale replacement.
  setConfig: (state, payload) => {
    const existingRaw = state.configs[payload.key];
    if (existingRaw != null) {
      const existing = actions.snapshotDraft(existingRaw);
      const restoreFields: record.Unknown = {};
      for (const k of Object.keys(payload.config))
        if (existing[k] !== undefined) restoreFields[k] = existing[k];
      state.configs[payload.key] = { ...existing, ...payload.config };
      if (Object.keys(restoreFields).length === 0)
        return { inverse: [], targets: [payload.key] };
      return {
        inverse: [setConfig({ key: payload.key, config: restoreFields })],
        targets: [payload.key],
      };
    }
    let cfg = payload.config;
    const edge = state.edges.find((e) => e.key === payload.key);
    if (edge != null) {
      const srcCfg = state.configs[edge.source.node] as
        { color?: color.Crude } | undefined;
      if (srcCfg?.color != null && !color.isZero(srcCfg.color))
        cfg = { ...cfg, color: srcCfg.color };
    }
    state.configs[payload.key] = cfg;
    return { inverse: [], targets: [payload.key] };
  },
};

export const reduceAll = createReduceAll(handlers);

// createOf hands the dispatch controller the document carried by a create
// action so frames for never-cached documents ingest instead of drop.
export const createOf = (action: Action) =>
  action.type === "create" ? action.create.schematic : undefined;

export const kindOf = (actions: Action[]): string => {
  if (actions.length === 0) return "default";
  // A drag dispatches a stream of `set_node_position` per frame, plus
  // `set_config` companions synthesized by augmentWithEdgeSegments for any
  // affected edges. Both shapes are part of one user gesture and must coalesce
  // together; classify them all as "move" so the per-kind coalesce window
  // collapses them into a single undoable.
  const hasMove = actions.some((a) => a.type === "set_node_position");
  const onlyMoveOrSegment = actions.every(
    (a) => a.type === "set_node_position" || a.type === "set_config",
  );
  if (hasMove && onlyMoveOrSegment) return "move";
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};
