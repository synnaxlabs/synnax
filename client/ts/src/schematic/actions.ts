// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createReduceAll, createReducer, type Handlers } from "@/schematic/actions.gen";

const handlers: Handlers = {
  setNodePosition: (state, payload) => {
    const node = state.nodes.find((n) => n.key === payload.key);
    if (node != null) node.position = payload.position;
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
  setEdge: (state, payload) => {
    const idx = state.edges.findIndex((e) => e.key === payload.edge.key);
    if (idx !== -1) state.edges[idx] = payload.edge;
    else state.edges.push(payload.edge);
  },
  removeEdge: (state, payload) => {
    const idx = state.edges.findIndex((e) => e.key === payload.key);
    if (idx !== -1) state.edges.splice(idx, 1);
  },
  setConfig: (state, payload) => {
    state.configs[payload.key] = payload.config;
  },
  setAuthority: (state, payload) => {
    state.authority = payload.value;
  },
  setLegend: (state, payload) => {
    state.legend = payload.legend;
  },
};

export const reduce = createReducer(handlers);
export const reduceAll = createReduceAll(handlers);
