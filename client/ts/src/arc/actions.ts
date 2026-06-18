// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
import { type ir } from "@/arc/ir";

// edgeID derives a stable identifier for an edge from its endpoints. Arc edges
// carry no key of their own, so the source and target handles serve as identity
// for undo invalidation.
const edgeID = (source: ir.Handle, target: ir.Handle): string =>
  `${source.node}:${source.param}->${target.node}:${target.param}`;

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
  setNodeConfig: (state, payload) => {
    const oldConfig = actions.snapshotDraft(state.graph.configs[payload.key]);
    state.graph.configs[payload.key] = payload.config;
    return {
      inverse: [setNodeConfig({ key: payload.key, config: oldConfig ?? {} })],
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
      targets: [edgeID(source, target)],
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
      targets: [edgeID(source, target)],
    };
  },
};

export const reduceAll = createReduceAll(handlers);

// isUndoable reports whether an action should push onto the undo stack. Every
// Arc graph mutation is user-driven and therefore undoable.
export const isUndoable = (_action: Action): boolean => true;
