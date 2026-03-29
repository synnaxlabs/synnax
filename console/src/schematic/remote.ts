// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";

import { type CreatePayload, type State, ZERO_STATE } from "@/schematic/slice";

export const stateFromRemote = (s: schematic.Schematic): CreatePayload => ({
  ...ZERO_STATE,
  key: s.key,
  snapshot: s.snapshot,
  editable: s.editable,
  fitViewOnResize: s.fitViewOnResize,
  authority: s.authority,
  viewport: s.viewport,
  legend: s.legend,
  nodes: s.nodes,
  edges: s.edges,
  props: s.props as State["props"],
});

export const stateToRemote = (
  state: State,
  key: string,
  name: string,
): schematic.New => ({
  key,
  name,
  snapshot: state.snapshot,
  editable: state.editable,
  fitViewOnResize: state.fitViewOnResize,
  authority: state.authority,
  viewport: state.viewport,
  legend: state.legend,
  nodes: state.nodes,
  edges: state.edges as schematic.New["edges"],
  props: state.props,
});

export const ZERO_REMOTE: schematic.New = {
  name: "New Schematic",
  snapshot: false,
  editable: ZERO_STATE.editable,
  fitViewOnResize: ZERO_STATE.fitViewOnResize,
  authority: ZERO_STATE.authority,
  viewport: ZERO_STATE.viewport,
  legend: ZERO_STATE.legend,
  nodes: ZERO_STATE.nodes,
  edges: ZERO_STATE.edges as schematic.New["edges"],
  props: ZERO_STATE.props,
};
