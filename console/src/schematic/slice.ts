// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Control } from "@synnaxlabs/pluto";

import * as latest from "@/schematic/types";

export type State = latest.State;
export type SliceState = latest.SliceState;
export type ToolbarTab = latest.ToolbarTab;
export type LegendState = latest.LegendState;
export type Viewport = latest.Viewport;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;

export const SLICE_NAME = "schematic";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload {
  key: string;
}

export interface SetSelectedPayload {
  key: string;
  selected: string[];
}

export interface SetControlStatusPayload {
  key: string;
  control: Control.Status;
}

export interface SetLegendPayload {
  key: string;
  legend: Partial<LegendState>;
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
  viewport: Viewport;
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
      };
    },
    setSelected: (state, { payload }: PayloadAction<SetSelectedPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.selected = payload.selected;
      s.activeToolbarTab = payload.selected.length > 0 ? "properties" : "symbols";
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.control = payload.control;
    },
    setLegend: (state, { payload }: PayloadAction<SetLegendPayload>) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.legend = { ...s.legend, ...payload.legend };
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
      s.activeToolbarTab = payload.tab;
    },
    setSelectedSymbolGroup: (
      state,
      { payload }: PayloadAction<SetSelectedSymbolGroupPayload>,
    ) => {
      const s = state.schematics[payload.key];
      if (s == null) return;
      s.selectedSymbolGroup = payload.group;
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
      s.viewport = payload.viewport;
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
  setLegend,
  setLegendVisible,
  setActiveToolbarTab,
  setSelectedSymbolGroup,
  setEditable,
  setFitViewOnResize,
  setViewport,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const PERSIST_EXCLUDE = [];
