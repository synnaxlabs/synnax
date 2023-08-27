// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PayloadAction, createSlice } from "@reduxjs/toolkit";
import { PID, Control } from "@synnaxlabs/pluto";
import { Deep, XY } from "@synnaxlabs/x";
import { nanoid } from "nanoid";

import { LayoutState, LayoutCreator } from "@/layout";

export type PIDNodeProps = object & {
  type: string;
};

export interface PIDState {
  editable: boolean;
  viewport: PID.Viewport;
  nodes: PID.Node[];
  edges: PID.Edge[];
  props: Record<string, object>;
  control: Control.Status;
  controlAcquireTrigger: number;
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
  viewport: { position: XY.ZERO.crude, zoom: 1 },
  editable: true,
  control: "released",
  controlAcquireTrigger: 0,
};

export const ZERO_PID_SLICE_STATE: PIDSliceState = {
  toolbar: { activeTab: "elements" },
  pids: {},
};

export interface SetPIDViewportPayload {
  layoutKey: string;
  viewport: PID.Viewport;
}

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
  nodes: PID.Node[];
}

export interface SetPIDEdgesPayload {
  layoutKey: string;
  edges: PID.Edge[];
}

export interface CreatePIDPayload {
  key: string;
}

export interface DeletePIDPayload {
  layoutKey: string;
}

export interface SetPIDEditablePayload {
  layoutKey: string;
  editable: boolean;
}

export interface SetPIDControlStatusPayload {
  layoutKey: string;
  control: Control.Status;
}

export interface TogglePIDControlPayload {
  layoutKey: string;
  status: Control.Status;
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
      state.pids[layoutKey] = { ...ZERO_PID_STATE };
    },
    deletePID: (state, { payload }: PayloadAction<DeletePIDPayload>) => {
      const { layoutKey } = payload;
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.pids[layoutKey];
    },
    addPIDelement: (state, { payload }: PayloadAction<AddPIDelementPayload>) => {
      const { layoutKey, key, props } = payload;
      const pid = state.pids[layoutKey];
      if (!pid.editable) return;
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
      if (!pid.editable) return;
      if (key in pid.props) {
        pid.props[key] = { ...pid.props[key], ...props };
      } else {
        const edge = pid.edges.findIndex((edge) => edge.key === key);
        if (edge !== -1) {
          pid.edges[edge] = { ...pid.edges[edge], ...props };
        }
      }
    },
    setPIDNodes: (state, { payload }: PayloadAction<SetPIDNodesPayload>) => {
      const { layoutKey, nodes } = payload;
      const pid = state.pids[layoutKey];
      pid.nodes = nodes;
      const anySelected = nodes.some((node) => node.selected);
      if (anySelected) state.toolbar.activeTab = "properties";
      else state.toolbar.activeTab = "elements";
    },
    setPIDEdges: (state, { payload }: PayloadAction<SetPIDEdgesPayload>) => {
      const { layoutKey, edges } = payload;
      const pid = state.pids[layoutKey];
      pid.edges = edges;
      const anySelected = edges.some((edge) => edge.selected);
      if (anySelected) state.toolbar.activeTab = "properties";
      else state.toolbar.activeTab = "elements";
    },
    setPIDActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetPIDActiveToolbarTabPayload>
    ) => {
      const { tab } = payload;
      state.toolbar.activeTab = tab;
    },
    setPIDViewport: (state, { payload }: PayloadAction<SetPIDViewportPayload>) => {
      const { layoutKey, viewport } = payload;
      const pid = state.pids[layoutKey];
      pid.viewport = viewport;
    },
    setPIDEditable: (state, { payload }: PayloadAction<SetPIDEditablePayload>) => {
      const { layoutKey, editable } = payload;
      const pid = state.pids[layoutKey];
      pid.editable = editable;
    },
    togglePIDControl: (state, { payload }: PayloadAction<TogglePIDControlPayload>) => {
      let { layoutKey, status } = payload;
      const pid = state.pids[layoutKey];
      if (status == null) status = pid.control === "released" ? "acquired" : "released";
      pid.controlAcquireTrigger += -2 * Number(status === "released") + 1;
    },
    setPIDControlState: (
      state,
      { payload }: PayloadAction<SetPIDControlStatusPayload>
    ) => {
      const { layoutKey, control } = payload;
      const pid = state.pids[layoutKey];
      pid.control = control;
    },
  },
});

export const {
  togglePIDControl,
  setPIDControlState,
  addPIDelement,
  setPIDEdges,
  setPIDNodes,
  setPIDElementProps,
  setPIDActiveToolbarTab,
  setPIDViewport,
  setPIDEditable,
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
