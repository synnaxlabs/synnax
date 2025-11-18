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

import { edgeZ, nodeZ, type Schematic } from "@/schematic/payload";

export const setNode = relapse.createAction({
  type: "set_node" as const,
  payload: z.object({
    node: nodeZ,
    props: record.unknownZ.optional(),
  }),
  handler: (state: Schematic, { node, props }) => {
    const index = state.nodes.findIndex((n) => n.key === node.key);
    if (index !== -1) state.nodes[index] = node;
    else state.nodes.push(node);
    if (props != null) state.props[node.key] = props;
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

export const setEdge = relapse.createAction({
  type: "set_edge" as const,
  payload: edgeZ,
  handler: (state: Schematic, edge) => {
    const index = state.edges.findIndex((e) => e.key === edge.key);
    if (index !== -1) state.edges[index] = edge;
    else state.edges.push(edge);
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
    typeof setNode,
    typeof setNodeProps,
    typeof setNodePosition,
    typeof removeNode,
    typeof removeEdge,
    typeof setEdge,
  ]
>([setNode, setNodeProps, setNodePosition, removeNode, removeEdge, setEdge] as const);

export type Action = z.infer<typeof actionZ>;

export const scopedActionZ = actionZ.and(
  z.object({
    key: z.string(),
  }),
);

export type ScopedAction = z.infer<typeof scopedActionZ>;
