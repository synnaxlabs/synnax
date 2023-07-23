// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PayloadAction, createSlice } from "@reduxjs/toolkit";
import { PIDEdge, PIDNode } from "@synnaxlabs/pluto";
import { Deep, XY } from "@synnaxlabs/x";
import { nanoid } from "nanoid";

import { LayoutState, LayoutCreator } from "@/layout";

export type PIDNodeProps = object & {
  type: string;
};

export interface PIDState {
  editable: boolean;
  nodes: PIDNode[];
  edges: PIDEdge[];
  props: Record<string, object>;
}

// ||||| TOOLBAR |||||

const PID_TOOLBAR_TABS = ["elements", "properties"] as const;
export type PIDToolbarTab = (typeof PID_TOOLBAR_TABS)[number];

export interface PIDToolbarState {
  activeTab: PIDToolbarTab;
}

export interface PIDSliceState {
  toolbar: PIDToolbarState;
  pids: Record<string, PIDState>;
}

export const PID_SLICE_NAME = "pid";

export interface PIDStoreState {
  [PID_SLICE_NAME]: PIDSliceState;
}

export const ZERO_PID_STATE: PIDState = {
  nodes: [],
  edges: [],
  props: {},
  editable: true,
};

export const ZERO_PID_SLICE_STATE: PIDSliceState = {
  toolbar: { activeTab: "elements" },
  pids: {},
};

export interface AddPIDelementPayload {
  layoutKey: string;
  key: string;
  props: PIDNodeProps;
}

export interface SetPIDElementPropsPayload {
  layoutKey: string;
  key: string;
  props: PIDNodeProps;
}

export interface SetPIDNodesPayload {
  layoutKey: string;
  nodes: PIDNode[];
}

export interface SetPIDEdgesPayload {
  layoutKey: string;
  edges: PIDEdge[];
}

export interface CreatePIDPayload {
  key: string;
}

export interface DeletePIDPayload {
  layoutKey: string;
}

export interface SetPIDActiveToolbarTabPayload {
  tab: PIDToolbarTab;
}

export const { actions, reducer: pidReducer } = createSlice({
  name: PID_SLICE_NAME,
  initialState: ZERO_PID_SLICE_STATE,
  reducers: {
    createPID: (state, { payload }: PayloadAction<CreatePIDPayload>) => {
      const { key: layoutKey } = payload;
      state.pids[layoutKey] = ZERO_PID_STATE;
    },
    deletePID: (state, { payload }: PayloadAction<DeletePIDPayload>) => {
      const { layoutKey } = payload;
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.pids[layoutKey];
    },
    addPIDelement: (state, { payload }: PayloadAction<AddPIDelementPayload>) => {
      const { layoutKey, key, props } = payload;
      const pid = state.pids[layoutKey];
      pid.nodes.push({
        key,
        selected: false,
        position: XY.ZERO.crude,
      });
      pid.props[key] = props;
    },
    setPIDElementProps: (
      state,
      { payload }: PayloadAction<SetPIDElementPropsPayload>
    ) => {
      const { layoutKey, key, props } = payload;
      const pid = state.pids[layoutKey];
      pid.props[key] = props;
    },
    setPIDNodes: (state, { payload }: PayloadAction<SetPIDNodesPayload>) => {
      const { layoutKey, nodes } = payload;
      const pid = state.pids[layoutKey];
      pid.nodes = nodes;
      const anySelected = nodes.some((node) => node.selected);
      if (anySelected) state.toolbar.activeTab = "properties";
    },
    setPIDEdges: (state, { payload }: PayloadAction<SetPIDEdgesPayload>) => {
      const { layoutKey, edges } = payload;
      const pid = state.pids[layoutKey];
      pid.edges = edges;
    },
    setPIDActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetPIDActiveToolbarTabPayload>
    ) => {
      const { tab } = payload;
      state.toolbar.activeTab = tab;
    },
  },
});

export const {
  addPIDelement,
  setPIDEdges,
  setPIDNodes,
  setPIDElementProps,
  setPIDActiveToolbarTab,
} = actions;

export type PIDAction = ReturnType<(typeof actions)[keyof typeof actions]>;
export type PIDPayload = PIDAction["payload"];

export const createPID =
  (initial: Partial<PIDState> & Omit<Partial<LayoutState>, "type">): LayoutCreator =>
  ({ dispatch }) => {
    const { name = "PID", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? nanoid();
    dispatch(actions.createPID({ ...Deep.copy(ZERO_PID_STATE), key, ...rest }));
    return {
      key,
      location,
      name,
      type: "pid",
      window,
      tab,
    };
  };
