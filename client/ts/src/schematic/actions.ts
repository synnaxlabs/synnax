// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";

import { createReduceAll, createReducer, type Handlers } from "@/schematic/actions.gen";

const handlers: Handlers = {
  setNodePosition: (state, payload) => {
    const node = state.nodes.find((n) => n.key === payload.key);
    if (node != null) node.position = payload.position;
  },
  setNodeMeasured: (state, payload) => {
    const node = state.nodes.find((n) => n.key === payload.key);
    if (node != null) node.measured = payload.measured;
  },
  setNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.node.key);
    if (idx !== -1) state.nodes[idx] = payload.node;
    else state.nodes.push(payload.node);
    if (payload.config != null) state.configs[payload.node.key] = payload.config;
  },
  removeNode: (state, payload) => {
    const idx = state.nodes.findIndex((n) => n.key === payload.key);
    if (idx !== -1) state.nodes.splice(idx, 1);
    delete state.configs[payload.key];
  },
  addEdge: (state, payload) => {
    if (state.edges.some((e) => e.key === payload.edge.key)) return;
    state.edges.push(payload.edge);
  },
  removeEdge: (state, payload) => {
    const idx = state.edges.findIndex((e) => e.key === payload.key);
    if (idx !== -1) state.edges.splice(idx, 1);
  },
  setConfig: (state, payload) => {
    const existing = state.configs[payload.key];
    if (existing != null) {
      state.configs[payload.key] = { ...existing, ...payload.config };
      return;
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
  },
};

export const reduce = createReducer(handlers);
export const reduceAll = createReduceAll(handlers);
