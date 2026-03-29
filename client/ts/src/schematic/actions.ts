// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Schematic, keyZ } from "@/schematic/types.gen";
import {
  type Action,
  actionZ,
  type AddNodePayload,
  type RemoveEdgePayload,
  type RemoveNodePayload,
  type SetEdgeDataPayload,
  type SetEdgePayload,
  type SetNodeDimensionsPayload,
  type SetNodePositionPayload,
  type SetNodePropsPayload,
} from "@/schematic/actions.gen";

export type { Action } from "@/schematic/actions.gen";
export {
  actionZ,
  ACTION_TYPES,
  setNodePosition,
  addNode,
  removeNode,
  setEdge,
  removeEdge,
  setNodeDimensions,
  setNodeProps,
  setEdgeData,
} from "@/schematic/actions.gen";

export const scopedActionZ = z.object({
  key: keyZ,
  sessionKey: z.string(),
  actions: actionZ.array(),
});

export type ScopedAction = z.infer<typeof scopedActionZ>;

const handleSetNodePosition = (
  state: Schematic,
  payload: SetNodePositionPayload,
): void => {
  const node = state.nodes.find((n) => n.key === payload.key);
  if (node != null) node.position = payload.position;
};

const handleAddNode = (state: Schematic, payload: AddNodePayload): void => {
  state.nodes.push(payload.node);
  if (payload.props != null) {
    if (state.props == null) state.props = {};
    state.props[payload.node.key] = payload.props;
  }
};

const handleRemoveNode = (
  state: Schematic,
  payload: RemoveNodePayload,
): void => {
  const i = state.nodes.findIndex((n) => n.key === payload.key);
  if (i !== -1) state.nodes.splice(i, 1);
  // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
  delete state.props[payload.key];
};

const handleSetEdge = (state: Schematic, payload: SetEdgePayload): void => {
  const i = state.edges.findIndex((e) => e.key === payload.edge.key);
  if (i !== -1) state.edges[i] = payload.edge;
  else state.edges.push(payload.edge);
};

const handleRemoveEdge = (
  state: Schematic,
  payload: RemoveEdgePayload,
): void => {
  const i = state.edges.findIndex((e) => e.key === payload.key);
  if (i !== -1) state.edges.splice(i, 1);
};

const handleSetNodeDimensions = (
  state: Schematic,
  payload: SetNodeDimensionsPayload,
): void => {
  const node = state.nodes.find((n) => n.key === payload.key);
  if (node != null) node.measured = payload.dimensions;
};

const handleSetNodeProps = (
  state: Schematic,
  payload: SetNodePropsPayload,
): void => {
  if (state.props == null) state.props = {};
  state.props[payload.key] = payload.props;
};

const handleSetEdgeData = (
  state: Schematic,
  payload: SetEdgeDataPayload,
): void => {
  const edge = state.edges.find((e) => e.key === payload.key);
  if (edge != null) edge.data = payload.data;
};

export const reduce = (state: Schematic, action: Action): Schematic => {
  switch (action.type) {
    case "set_node_position":
      handleSetNodePosition(state, action.setNodePosition);
      break;
    case "add_node":
      handleAddNode(state, action.addNode);
      break;
    case "remove_node":
      handleRemoveNode(state, action.removeNode);
      break;
    case "set_edge":
      handleSetEdge(state, action.setEdge);
      break;
    case "remove_edge":
      handleRemoveEdge(state, action.removeEdge);
      break;
    case "set_node_dimensions":
      handleSetNodeDimensions(state, action.setNodeDimensions);
      break;
    case "set_node_props":
      handleSetNodeProps(state, action.setNodeProps);
      break;
    case "set_edge_data":
      handleSetEdgeData(state, action.setEdgeData);
      break;
  }
  return state;
};

export const reduceAll = (state: Schematic, actions: Action[]): Schematic => {
  for (const action of actions) state = reduce(state, action);
  return state;
};
