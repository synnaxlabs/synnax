// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type schematic } from "@synnaxlabs/client";
import { type Drift } from "@synnaxlabs/drift";
import { type Control, control, type Diagram, Viewport } from "@synnaxlabs/pluto";
import { color, control as xcontrol, sticky, xy } from "@synnaxlabs/x";
import z from "zod";

import { Window } from "@/session/window";

export const viewportZ = z.object({
  position: xy.xyZ.default({ x: 0, y: 0 }),
  zoom: z.number().default(1),
  mode: Viewport.modeZ.default("select"),
});
export interface Viewport extends z.infer<typeof viewportZ> {}

export const legendStateZ = z.object({
  visible: z.boolean().default(true),
  position: sticky.xyZ.default({
    x: 24,
    y: 24,
    root: { x: "left", y: "top" },
    units: { x: "px", y: "px" },
  }),
  colors: z.record(z.string(), color.colorZ).default({}),
});
export interface LegendState extends z.infer<typeof legendStateZ> {}

export const toolbarTabZ = z.enum(["symbols", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;
export const toolbarStateZ = z.object({
  selectedTab: toolbarTabZ.default("symbols"),
  selectedSymbolGroup: z.string().default("general"),
});
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}

export const controlStateZ = z.object({
  authority: xcontrol.authorityZ.default(1),
  status: control.statusZ.default("released"),
});

export const stateZ = z.object({
  control: controlStateZ.prefault({}),
  selected: z.array(z.string()).default([]),
  legend: legendStateZ.prefault({}),
  toolbar: toolbarStateZ.prefault({}),
  editable: z.boolean().default(false),
  fitViewOnResize: z.boolean().default(false),
  viewport: viewportZ.prefault({}),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export const windowStateZ = z.record(z.string(), stateZ).default({});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** A window is a viewport, so each holds its own view of every document. */
  windows: z.record(z.string(), windowStateZ).default({}),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "schematic";

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload extends Window.OptionalKeyParams {
  key: schematic.Key;
}

export interface CreatePayload extends KeyedPayload, NewState {}

export interface SetSelectedPayload extends KeyedPayload {
  selected: string[];
}

export interface SetControlStatusPayload extends KeyedPayload {
  status: Control.Status;
}

export interface SetAuthorityPayload extends KeyedPayload {
  authority: xcontrol.Authority;
}

export interface MoveLegendPayload extends KeyedPayload {
  position: sticky.XY;
}

export interface SetLegendColorsPayload extends KeyedPayload {
  colors: Record<string, color.Color>;
}

export interface SetLegendVisiblePayload extends KeyedPayload {
  visible: boolean;
}

export interface SelectToolbarTabPayload extends KeyedPayload {
  tab: ToolbarTab;
}

export interface SetSelectedSymbolGroupPayload extends KeyedPayload {
  group: string;
}

export interface SetEditablePayload extends KeyedPayload {
  editable: boolean;
}

export interface SetFitViewOnResizePayload extends KeyedPayload {
  fitViewOnResize: boolean;
}

export interface SetViewportPayload extends KeyedPayload {
  viewport: Diagram.Viewport;
}

export interface SetViewportModePayload extends KeyedPayload {
  mode: Viewport.Mode;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);
const initializeDocument = Window.createDocumentInitializer(stateZ);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: initializeDocument<CreatePayload, SliceState>,
    setSelected: withSelectedState<SetSelectedPayload, SliceState>(
      (state, { payload }) => {
        state.selected = payload.selected;
        state.toolbar.selectedTab =
          payload.selected.length > 0 ? "properties" : "symbols";
      },
    ),
    setControlStatus: withSelectedState<SetControlStatusPayload, SliceState>(
      (state, { payload: { status: control } }) => {
        state.control.status = control;
        if (control !== "acquired") return;
        state.selected = [];
        state.editable = false;
      },
    ),
    setControlAuthority: withSelectedState<SetAuthorityPayload, SliceState>(
      (state, { payload: { authority } }) => {
        state.control.authority = authority;
      },
    ),
    moveLegend: withSelectedState<MoveLegendPayload, SliceState>(
      (state, { payload: { position } }) => {
        state.legend.position = position;
      },
    ),
    setLegendColors: withSelectedState<SetLegendColorsPayload, SliceState>(
      (state, { payload: { colors } }) => {
        state.legend.colors = colors;
      },
    ),
    setLegendVisible: withSelectedState<SetLegendVisiblePayload, SliceState>(
      (state, { payload: { visible } }) => {
        state.legend.visible = visible;
      },
    ),
    selectToolbarTab: withSelectedState<SelectToolbarTabPayload, SliceState>(
      (state, { payload: { tab } }) => {
        state.toolbar.selectedTab = tab;
      },
    ),
    setSelectedSymbolGroup: withSelectedState<
      SetSelectedSymbolGroupPayload,
      SliceState
    >((state, { payload: { group } }) => {
      state.toolbar.selectedSymbolGroup = group;
    }),
    setEditable: withSelectedState<SetEditablePayload, SliceState>(
      (state, { payload: { editable } }) => {
        state.editable = editable;
        if (!editable) state.selected = [];
      },
    ),
    setFitViewOnResize: withSelectedState<SetFitViewOnResizePayload, SliceState>(
      (state, { payload: { fitViewOnResize } }) => {
        state.fitViewOnResize = fitViewOnResize;
      },
    ),
    setViewport: withSelectedState<SetViewportPayload, SliceState>(
      (state, { payload: { viewport } }) => {
        state.viewport = { ...state.viewport, ...viewport };
      },
    ),
    setViewportMode: withSelectedState<SetViewportModePayload, SliceState>(
      (state, { payload: { mode } }) => {
        state.viewport.mode = mode;
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      Window.removeDocuments(state, payload.keys);
    },
  },
  extraReducers: Window.handleRemoved,
});

export const {
  create,
  setSelected,
  setControlStatus,
  setControlAuthority,
  setLegendColors,
  moveLegend,
  setLegendVisible,
  selectToolbarTab,
  setSelectedSymbolGroup,
  setEditable,
  setFitViewOnResize,
  setViewport,
  setViewportMode,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const purgeState = (state: State): State => {
  state.control.status = "released";
  return state;
};

export const purgeSliceState = <S extends StoreState>(state: S): S => {
  Window.purgeDocuments(state[SLICE_NAME], purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    create,
    setSelected,
    setControlStatus,
    setControlAuthority,
    setLegendColors,
    moveLegend,
    setLegendVisible,
    selectToolbarTab,
    setSelectedSymbolGroup,
    setEditable,
    setFitViewOnResize,
    setViewport,
    setViewportMode,
  ]),
];
