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

export type NodeProps = object & {
  type: string;
};

export interface State {
  editable: boolean;
  viewport: PID.Viewport;
  nodes: PID.Node[];
  edges: PID.Edge[];
  props: Record<string, object>;
  control: Control.Status;
  controlAcquireTrigger: number;
}

// ||||| TOOLBAR |||||

const TOOLBAR_TABS = ["elements", "properties"] as const;
export type ToolbarTab = (typeof TOOLBAR_TABS)[number];

export interface ToolbarState {
  activeTab: ToolbarTab;
}

export interface SliceState {
  toolbar: ToolbarState;
  pids: Record<string, State>;
}

export const SLICE_NAME = "pid";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const ZERO_STATE: State = {
  nodes: [],
  edges: [],
  props: {},
  viewport: { position: XY.ZERO.crude, zoom: 1 },
  editable: true,
  control: "released",
  controlAcquireTrigger: 0,
};

export const ZERO_PID_SLICE_STATE: SliceState = {
  toolbar: { activeTab: "elements" },
  pids: {},
};

export interface SetViewportPayload {
  layoutKey: string;
  viewport: PID.Viewport;
}

export interface AddElementPayload {
  layoutKey: string;
  key: string;
  props: NodeProps;
  node?: Partial<PID.Node>;
}

export interface SetElementPropsPayload {
  layoutKey: string;
  key: string;
  props: NodeProps;
}

export interface SetNodesPayload {
  layoutKey: string;
  nodes: PID.Node[];
}

export interface SetEdgesPayload {
  layoutKey: string;
  edges: PID.Edge[];
}

export interface CreatePayload {
  key: string;
}

export interface DeletePayload {
  layoutKey: string;
}

export interface SetEditablePayload {
  layoutKey: string;
  editable: boolean;
}

export interface SetControlStatusPayload {
  layoutKey: string;
  control: Control.Status;
}

export interface TogggleControlPayload {
  layoutKey: string;
  status: Control.Status;
}

export interface SetPIDActiveToolbarTabPayload {
  tab: ToolbarTab;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_PID_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key: layoutKey } = payload;
      state.pids[layoutKey] = { ...ZERO_STATE };
    },
    delete: (state, { payload }: PayloadAction<DeletePayload>) => {
      const { layoutKey } = payload;
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete state.pids[layoutKey];
    },
    addElement: (state, { payload }: PayloadAction<AddElementPayload>) => {
      const { layoutKey, key, props, node } = payload;
      const pid = state.pids[layoutKey];
      if (!pid.editable) return;
      pid.nodes.push({
        key,
        selected: false,
        position: XY.ZERO.crude,
        ...node,
      });
      pid.props[key] = props;
    },
    setElementProps: (state, { payload }: PayloadAction<SetElementPropsPayload>) => {
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
    setNodes: (state, { payload }: PayloadAction<SetNodesPayload>) => {
      const { layoutKey, nodes } = payload;
      const pid = state.pids[layoutKey];
      pid.nodes = nodes;
      const anySelected = nodes.some((node) => node.selected);
      if (anySelected) state.toolbar.activeTab = "properties";
      else state.toolbar.activeTab = "elements";
    },
    setEdges: (state, { payload }: PayloadAction<SetEdgesPayload>) => {
      const { layoutKey, edges } = payload;
      const pid = state.pids[layoutKey];
      pid.edges = edges;
      const anySelected = edges.some((edge) => edge.selected);
      if (anySelected) state.toolbar.activeTab = "properties";
      else state.toolbar.activeTab = "elements";
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetPIDActiveToolbarTabPayload>
    ) => {
      const { tab } = payload;
      state.toolbar.activeTab = tab;
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const { layoutKey, viewport } = payload;
      const pid = state.pids[layoutKey];
      pid.viewport = viewport;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { layoutKey, editable } = payload;
      const pid = state.pids[layoutKey];
      pid.editable = editable;
    },
    toggleControl: (state, { payload }: PayloadAction<TogggleControlPayload>) => {
      let { layoutKey, status } = payload;
      const pid = state.pids[layoutKey];
      if (status == null) status = pid.control === "released" ? "acquired" : "released";
      pid.controlAcquireTrigger += -2 * Number(status === "released") + 1;
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { layoutKey, control } = payload;
      const pid = state.pids[layoutKey];
      pid.control = control;
    },
  },
});

export const {
  toggleControl,
  setControlStatus,
  addElement,
  setEdges,
  setNodes,
  setElementProps,
  setActiveToolbarTab,
  setViewport,
  setEditable,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const create =
  (initial: Partial<State> & Omit<Partial<LayoutState>, "type">): LayoutCreator =>
  ({ dispatch }) => {
    const { name = "PID", location = "mosaic", window, tab, ...rest } = initial;
    const key = initial.key ?? nanoid();
    dispatch(actions.create({ ...Deep.copy(ZERO_STATE), key, ...rest }));
    return {
      key,
      location,
      name,
      type: "pid",
      window,
      tab,
    };
  };
