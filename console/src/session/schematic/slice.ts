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
import { type Control, control, type Diagram, Viewport } from "@synnaxlabs/pluto";
import { color, control as xcontrol, sticky, xy } from "@synnaxlabs/x";
import z from "zod";

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

export const sliceStateZ = z.object({
  schematics: z.record(z.string(), stateZ).default({}),
});

export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "schematic";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload {
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

const withSelectedState =
  <Payload extends KeyedPayload, Type extends string = string>(
    handler?: (state: State, action: PayloadAction<Payload, Type>) => void,
  ) =>
  (state: SliceState, action: PayloadAction<Payload, Type>) => {
    const {
      payload: { key },
    } = action;
    let s = state.schematics[key];
    if (s == null) {
      s = stateZ.parse({});
      state.schematics[key] = s;
    }
    handler?.(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (payload.key in state.schematics) return;
      state.schematics[payload.key] = stateZ.parse(payload);
    },
    setSelected: withSelectedState(
      (state, { payload }: PayloadAction<SetSelectedPayload>) => {
        state.selected = payload.selected;
        state.toolbar.selectedTab =
          payload.selected.length > 0 ? "properties" : "symbols";
      },
    ),
    setControlStatus: withSelectedState(
      (
        state,
        { payload: { status: control } }: PayloadAction<SetControlStatusPayload>,
      ) => {
        state.control.status = control;
        if (control !== "acquired") return;
        state.selected = [];
        state.editable = false;
      },
    ),
    setControlAuthority: withSelectedState(
      (state, { payload: { authority } }: PayloadAction<SetAuthorityPayload>) => {
        state.control.authority = authority;
      },
    ),
    moveLegend: withSelectedState(
      (state, { payload: { position } }: PayloadAction<MoveLegendPayload>) => {
        state.legend.position = position;
      },
    ),
    setLegendColors: withSelectedState(
      (state, { payload: { colors } }: PayloadAction<SetLegendColorsPayload>) => {
        state.legend.colors = colors;
      },
    ),
    setLegendVisible: withSelectedState(
      (state, { payload: { visible } }: PayloadAction<SetLegendVisiblePayload>) => {
        state.legend.visible = visible;
      },
    ),
    selectToolbarTab: withSelectedState(
      (state, { payload: { tab } }: PayloadAction<SelectToolbarTabPayload>) => {
        state.toolbar.selectedTab = tab;
      },
    ),
    setSelectedSymbolGroup: withSelectedState(
      (state, { payload: { group } }: PayloadAction<SetSelectedSymbolGroupPayload>) => {
        state.toolbar.selectedSymbolGroup = group;
      },
    ),
    setEditable: withSelectedState(
      (state, { payload: { editable } }: PayloadAction<SetEditablePayload>) => {
        state.editable = editable;
        if (!editable) state.selected = [];
      },
    ),
    setFitViewOnResize: withSelectedState(
      (
        state,
        { payload: { fitViewOnResize } }: PayloadAction<SetFitViewOnResizePayload>,
      ) => {
        state.fitViewOnResize = fitViewOnResize;
      },
    ),
    setViewport: withSelectedState(
      (state, { payload: { viewport } }: PayloadAction<SetViewportPayload>) => {
        state.viewport = { ...state.viewport, ...viewport };
      },
    ),
    setViewportMode: withSelectedState(
      (state, { payload: { mode } }: PayloadAction<SetViewportModePayload>) => {
        state.viewport.mode = mode;
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.schematics[key]);
    },
  },
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
  Object.values(state[SLICE_NAME].schematics).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];
