// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record, relapse, xy } from "@synnaxlabs/x";
import z from "zod";

import { nodeZ, type Schematic } from "@/schematic/payload";

export const addNode = relapse.createAction({
  type: "add_node" as const,
  payload: z.object({
    key: z.string(),
    node: nodeZ,
    props: record.unknownZ.optional(),
  }),
  handler: (state: Schematic, payload) => {
    state.nodes.push(payload.node);
    if (payload.props != null) state.props[payload.key] = payload.props;
  },
});

export const setNodeProps = relapse.createAction({
  type: "set_node_props" as const,
  payload: z.object({ key: z.string(), props: record.unknownZ }),
  handler: (state: Schematic, { key, props }) => {
    state.props[key] = props;
  },
});

export const setNodePosition = relapse.createAction({
  type: "set_node_position" as const,
  payload: z.object({ key: z.string(), position: xy.xy }),
  handler: (state: Schematic, { key, position }) => {
    const node = state.nodes.find((n) => n.key === key);
    if (node != null) node.position = position;
  },
});

export const removeNode = relapse.createAction({
  type: "remove_node" as const,
  payload: z.object({ key: z.string() }),
  handler: (state: Schematic, { key }) => {
    const index = state.nodes.findIndex((n) => n.key === key);
    if (index !== -1) state.nodes.splice(index, 1);
  },
});

export const removeEdge = relapse.createAction({
  type: "remove_edge" as const,
  payload: z.object({ key: z.string() }),
  handler: (state: Schematic, { key }) => {
    const index = state.edges.findIndex((e) => e.key === key);
    if (index !== -1) state.edges.splice(index, 1);
  },
});

export const { actionZ, reducer } = relapse.createReducer<
  Schematic,
  [
    typeof addNode,
    typeof setNodeProps,
    typeof setNodePosition,
    typeof removeNode,
    typeof removeEdge,
  ]
>([addNode, setNodeProps, setNodePosition, removeNode, removeEdge] as const);

export type Action = z.infer<typeof actionZ>;

export const scopedActionZ = actionZ.and(
  z.object({
    key: z.string(),
  }),
);

export type ScopedAction = z.infer<typeof scopedActionZ>;
