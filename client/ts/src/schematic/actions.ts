// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type AddNodePayload,
  type RemoveEdgePayload,
  type RemoveNodePayload,
  type SetAuthorityPayload,
  type SetEdgePayload,
  type SetLegendPayload,
  type SetNodePositionPayload,
  type SetPropsPayload,
} from "@/schematic/actions.gen";
import { type Schematic } from "@/schematic/types.gen";

export const handleSetNodePosition = (
  state: Schematic,
  payload: SetNodePositionPayload,
): void => {
  const node = state.nodes.find((n) => n.key === payload.key);
  if (node != null) node.position = payload.position;
};

export const handleAddNode = (state: Schematic, payload: AddNodePayload): void => {
  state.nodes.push(payload.node);
  if (payload.props != null) state.props[payload.node.key] = payload.props;
};

export const handleRemoveNode = (
  state: Schematic,
  payload: RemoveNodePayload,
): void => {
  const idx = state.nodes.findIndex((n) => n.key === payload.key);
  if (idx !== -1) state.nodes.splice(idx, 1);
  delete state.props[payload.key];
};

export const handleSetEdge = (state: Schematic, payload: SetEdgePayload): void => {
  const idx = state.edges.findIndex((e) => e.key === payload.edge.key);
  if (idx !== -1) state.edges[idx] = payload.edge;
  else state.edges.push(payload.edge);
};

export const handleRemoveEdge = (
  state: Schematic,
  payload: RemoveEdgePayload,
): void => {
  const idx = state.edges.findIndex((e) => e.key === payload.key);
  if (idx !== -1) state.edges.splice(idx, 1);
};

export const handleSetProps = (state: Schematic, payload: SetPropsPayload): void => {
  state.props[payload.key] = payload.props;
};

export const handleSetAuthority = (
  state: Schematic,
  payload: SetAuthorityPayload,
): void => {
  state.authority = payload.value;
};

export const handleSetLegend = (
  state: Schematic,
  payload: SetLegendPayload,
): void => {
  state.legend = payload.legend;
};
