// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Control, type Diagram, type Viewport } from "@synnaxlabs/pluto";
import { type color, type control, type xy } from "@synnaxlabs/x";

import * as latest from "@/schematic/types";
import { type RootState } from "@/store";

export type SliceState = latest.SliceState;
export type State = latest.State;
export interface Viewport extends latest.Viewport {}
export type LegendState = latest.LegendState;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const anyStateZ = latest.anyStateZ;

export const SLICE_NAME = "schematic";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload {
  key: string;
  editable?: boolean;
}

export interface SetSelectedPayload {
  key: string;
  selected: string[];
}

export interface SetControlStatusPayload {
  key: string;
  control: Control.Status;
}

export interface SetAuthorityPayload {
  key: string;
  authority: control.Authority;
}

export interface MoveLegendPayload {
  key: string;
  position: xy.XY;
}

export interface SetLegendColorsPayload {
  key: string;
  colors: Record<string, color.Color>;
}

export interface SetLegendVisiblePayload {
  key: string;
  visible: boolean;
}

export interface SetActiveToolbarTabPayload {
  key: string;
  tab: ToolbarTab;
}

export interface SetSelectedSymbolGroupPayload {
  key: string;
  group: string;
}

export interface SetEditablePayload {
  key: string;
  editable: boolean;
}

export interface SetFitViewOnResizePayload {
  key: string;
  fitViewOnResize: boolean;
}

export interface SetViewportPayload {
  key: string;
  viewport: Diagram.Viewport;
}

export interface SetViewportModePayload {
  key: string;
  mode: Viewport.Mode;
}

export interface RemovePayload {
  keys: string[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (state.schematics[payload.key] != null) return;
      state.schematics[payload.key] = {
        ...ZERO_STATE,
        legend: { ...ZERO_STATE.legend },
        selected: [],
        editable: payload.editable ?? ZERO_STATE.editable,
      };
    },
    setSelected: (state, { payload }: PayloadAction<SetSelectedPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.selected = payload.selected;
      s.toolbar.activeTab = payload.selected.length > 0 ? "properties" : "symbols";
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { key: layoutKey, control } = payload;
      const schematic = state.schematics[layoutKey];
      if (schematic == null) return;
      schematic.controlStatus = control;
      if (control === "acquired") {
        schematic.selected = [];
        schematic.editable = false;
      }
    },
    setAuthority: (state, { payload }: PayloadAction<SetAuthorityPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.authority = payload.authority;
    },
    moveLegend: (state, { payload }: PayloadAction<MoveLegendPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.legend.position = payload.position;
    },
    setLegendColors: (state, { payload }: PayloadAction<SetLegendColorsPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.legend.colors = payload.colors;
    },
    setLegendVisible: (state, { payload }: PayloadAction<SetLegendVisiblePayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.legend.visible = payload.visible;
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.toolbar.activeTab = payload.tab;
    },
    setSelectedSymbolGroup: (
      state,
      { payload }: PayloadAction<SetSelectedSymbolGroupPayload>,
    ) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.toolbar.selectedSymbolGroup = payload.group;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.editable = payload.editable;
      if (!payload.editable) s.selected = [];
    },
    setFitViewOnResize: (
      state,
      { payload }: PayloadAction<SetFitViewOnResizePayload>,
    ) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.fitViewOnResize = payload.fitViewOnResize;
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.viewport = { ...s.viewport, ...payload.viewport };
    },
    setViewportMode: (state, { payload }: PayloadAction<SetViewportModePayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.viewport.mode = payload.mode;
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.schematics[key]);
    },
  },
});

export const {
  create: internalCreate,
  setSelected,
  setControlStatus,
  setAuthority,
  setLegendColors,
  moveLegend,
  setLegendVisible,
  setActiveToolbarTab,
  setSelectedSymbolGroup,
  setEditable,
  setFitViewOnResize,
  setViewport,
  setViewportMode,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const purgeState = (state: State): State => {
  state.controlStatus = "released";
  state.toolbar = { ...state.toolbar, activeTab: "symbols" };
  state.selected = [];
  return state;
};

export const purgeSliceState = (state: RootState): RootState => {
  Object.values(state[SLICE_NAME].schematics).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];
