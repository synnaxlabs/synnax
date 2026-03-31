// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import {
  type Action,
  actionZ,
  type AddNodePayload,
  type RemoveEdgePayload,
  type RemoveNodePayload,
  type SetEdgePayload,
  type SetNodeDimensionsPayload,
  type SetNodePositionPayload,
  type SetPropsPayload,
} from "@/schematic/actions.gen";
import { keyZ, type Schematic } from "@/schematic/types.gen";

export type { Action } from "@/schematic/actions.gen";
export {
  ACTION_TYPES,
  actionZ,
  addNode,
  removeEdge,
  removeNode,
  setEdge,
  setNodeDimensions,
  setNodePosition,
  setProps,
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
    state.props ??= {};
    state.props[payload.node.key] = payload.props;
  }
};

const handleRemoveNode = (state: Schematic, payload: RemoveNodePayload): void => {
  const i = state.nodes.findIndex((n) => n.key === payload.key);
  if (i !== -1) state.nodes.splice(i, 1);
  delete state.props[payload.key];
};

const handleSetEdge = (state: Schematic, payload: SetEdgePayload): void => {
  const i = state.edges.findIndex((e) => e.key === payload.edge.key);
  if (i !== -1) state.edges[i] = payload.edge;
  else state.edges.push(payload.edge);
};

const handleRemoveEdge = (state: Schematic, payload: RemoveEdgePayload): void => {
  const i = state.edges.findIndex((e) => e.key === payload.key);
  if (i !== -1) state.edges.splice(i, 1);
  delete state.props[payload.key];
};

const handleSetNodeDimensions = (
  state: Schematic,
  payload: SetNodeDimensionsPayload,
): void => {
  const node = state.nodes.find((n) => n.key === payload.key);
  if (node != null) node.measured = payload.dimensions;
};

const handleSetProps = (state: Schematic, payload: SetPropsPayload): void => {
  state.props ??= {};
  state.props[payload.key] = payload.props;
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
    case "set_props":
      handleSetProps(state, action.setProps);
      break;
  }
  return state;
};

export const reduceAll = (state: Schematic, actions: Action[]): Schematic => {
  for (const action of actions) state = reduce(state, action);
  return state;
};
