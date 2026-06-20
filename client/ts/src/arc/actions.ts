// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { actions } from "@/actions";
import {
  type Action,
  addEdge,
  createReduceAll,
  type Handlers,
  removeEdge,
  removeNode,
  rename,
  setMode,
  setNode,
  setNodeConfig,
  setNodePosition,
} from "@/arc/actions.gen";
import { ir } from "@/arc/ir";

const sameEndpoints = (a: ir.Edge, source: ir.Handle, target: ir.Handle): boolean =>
  a.source.node === source.node &&
  a.source.param === source.param &&
  a.target.node === target.node &&
  a.target.param === target.param;

const handlers: Handlers = {
  rename: (state, payload) => {
    const oldName = state.name;
    state.name = payload.name;
    return { inverse: [rename({ name: oldName })], targets: [state.key] };
  },
  setMode: (state, payload) => {
    const oldMode = state.mode;
    if (oldMode === payload.mode) return actions.NO_OP_RESULT;
    state.mode = payload.mode;
    return { inverse: [setMode({ mode: oldMode })], targets: [state.key] };
  },
  setNode: (state, payload) => {
    const idx = state.graph.nodes.findIndex((n) => n.key === payload.node.key);
    if (idx === -1) {
      state.graph.nodes.push(payload.node);
      return {
        inverse: [removeNode({ key: payload.node.key })],
        targets: [payload.node.key],
      };
    }
    const oldNode = actions.snapshotDraft(state.graph.nodes[idx]);
    state.graph.nodes[idx] = payload.node;
    return { inverse: [setNode({ node: oldNode })], targets: [payload.node.key] };
  },
  setNodePosition: (state, payload) => {
    const node = state.graph.nodes.find((n) => n.key === payload.key);
    if (node == null) return actions.NO_OP_RESULT;
    const oldPosition = { x: node.position.x, y: node.position.y };
    node.position = payload.position;
    return {
      inverse: [setNodePosition({ key: payload.key, position: oldPosition })],
      targets: [payload.key],
    };
  },
  // The inverse of SetNodeConfig is imperfect for keys the action newly
  // introduces: SetNodeConfig only merges, so it cannot remove keys that did
  // not previously exist. The inverse here restores values for keys that DID
  // exist before the merge; keys added by the action remain on undo as phantom
  // fields. A future ReplaceNodeConfig action can close the gap by enabling
  // wholesale replacement.
  setNodeConfig: (state, payload) => {
    const existingRaw = state.graph.configs[payload.key];
    if (existingRaw == null) {
      state.graph.configs[payload.key] = payload.config;
      return { inverse: [], targets: [payload.key] };
    }
    const existing = actions.snapshotDraft(existingRaw);
    const restoreFields: record.Unknown = {};
    for (const k of Object.keys(payload.config))
      if (existing[k] !== undefined) restoreFields[k] = existing[k];
    state.graph.configs[payload.key] = { ...existing, ...payload.config };
    if (Object.keys(restoreFields).length === 0)
      return { inverse: [], targets: [payload.key] };
    return {
      inverse: [setNodeConfig({ key: payload.key, config: restoreFields })],
      targets: [payload.key],
    };
  },
  removeNode: (state, payload) => {
    const idx = state.graph.nodes.findIndex((n) => n.key === payload.key);
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldNode = actions.snapshotDraft(state.graph.nodes[idx]);
    const oldConfig = actions.snapshotDraft(state.graph.configs[payload.key]);
    const removedEdges = state.graph.edges
      .filter((e) => e.source.node === payload.key || e.target.node === payload.key)
      .map((e) => actions.snapshotDraft(e));
    state.graph.nodes.splice(idx, 1);
    delete state.graph.configs[payload.key];
    state.graph.edges = state.graph.edges.filter(
      (e) => e.source.node !== payload.key && e.target.node !== payload.key,
    );
    const inverse: Action[] = [setNode({ node: oldNode })];
    if (oldConfig != null)
      inverse.push(setNodeConfig({ key: payload.key, config: oldConfig }));
    inverse.push(...removedEdges.map((e) => addEdge({ edge: e })));
    return { inverse, targets: [payload.key] };
  },
  addEdge: (state, payload) => {
    const { source, target } = payload.edge;
    if (state.graph.edges.some((e) => sameEndpoints(e, source, target)))
      return actions.NO_OP_RESULT;
    state.graph.edges.push(payload.edge);
    return {
      inverse: [removeEdge({ source, target })],
      targets: [ir.edgeKey(source, target)],
    };
  },
  removeEdge: (state, payload) => {
    const { source, target } = payload;
    const idx = state.graph.edges.findIndex((e) => sameEndpoints(e, source, target));
    if (idx === -1) return actions.NO_OP_RESULT;
    const oldEdge = actions.snapshotDraft(state.graph.edges[idx]);
    state.graph.edges.splice(idx, 1);
    return {
      inverse: [addEdge({ edge: oldEdge })],
      targets: [ir.edgeKey(source, target)],
    };
  },
  // The text document is a replicated op-log: applying an op is appending it, and
  // materialization (crdt.Text) deduplicates and orders. Text edits are not pushed onto
  // the undo stack (collaborative text undo is a separate concern), so inverse is empty.
  insertChar: (state, payload) => {
    state.text.doc.inserts.push(payload);
    return { inverse: [], targets: [] };
  },
  deleteChar: (state, payload) => {
    state.text.doc.deletes.push(payload);
    return { inverse: [], targets: [] };
  },
};

export const reduceAll = createReduceAll(handlers);

// isUndoable reports whether an action should push onto the undo stack. Graph mutations
// are user-driven and undoable; collaborative text edits are not (they carry no inverse
// and belong to the CRDT, not the graph undo stack).
export const isUndoable = (action: Action): boolean =>
  action.type !== "insert_char" && action.type !== "delete_char";
