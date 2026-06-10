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
import { elementConfigZ } from "@/schematic/types.gen";

const handlers: Handlers = {
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

  // set_node_measured carries layout-derived dimensions, not user intent.
  // Both the inverse and the target list are empty so it neither contributes
  // to the undo stack nor invalidates undoables targeting the same node when
  // a remote session emits it.
  setNodeMeasured: (state, payload) => {
    const node = state.nodes.find((n) => n.key === payload.key);
    if (node == null) return actions.NO_OP_RESULT;
    node.measured = payload.measured;
    return { inverse: [], targets: [] };
  },
  setNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.node.key);
    if (idx === -1) {
      state.nodes.push(payload.node);
      if (payload.config != null)
        state.configs[payload.node.key] = elementConfigZ.parse(payload.config);
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
    if (payload.config != null)
      state.configs[payload.node.key] = elementConfigZ.parse(payload.config);
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
      const existing: record.Unknown = actions.snapshotDraft(existingRaw);
      const restoreFields: record.Unknown = {};
      for (const k of Object.keys(payload.config))
        if (existing[k] !== undefined) restoreFields[k] = existing[k];
      state.configs[payload.key] = elementConfigZ.parse({
        ...existing,
        ...payload.config,
      });
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
        | { color?: color.Crude }
        | undefined;
      if (srcCfg?.color != null && !color.isZero(srcCfg.color))
        cfg = { ...cfg, color: srcCfg.color };
    }
    state.configs[payload.key] = elementConfigZ.parse(cfg);
    return { inverse: [], targets: [payload.key] };
  },
};

export const reduceAll = createReduceAll(handlers);

export const isUndoable = (action: Action): boolean =>
  action.type !== "set_node_measured";
