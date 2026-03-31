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
import { type sticky } from "@synnaxlabs/x";
import { z } from "zod";

export const SLICE_NAME = "schematic";

export const toolbarTabZ = z.enum(["symbols", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export interface LegendState {
  visible: boolean;
  position: sticky.XY;
}

export interface State {
  selected: string[];
  control: Control.Status;
  legend: LegendState;
  activeToolbarTab: ToolbarTab;
  selectedSymbolGroup: string;
  editable: boolean;
  fitViewOnResize: boolean;
}

export const ZERO_STATE: State = {
  selected: [],
  control: "released",
  legend: { visible: false, position: { x: 0, y: 0 } },
  activeToolbarTab: "symbols",
  selectedSymbolGroup: "general",
  editable: true,
  fitViewOnResize: false,
};

export interface SliceState {
  schematics: Record<string, State>;
}

export const ZERO_SLICE_STATE: SliceState = {
  schematics: {},
};

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

const ensure = (state: SliceState, key: string): State => {
  state.schematics[key] ??= {
    ...ZERO_STATE,
    legend: { ...ZERO_STATE.legend },
    selected: [],
  };
  return state.schematics[key];
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    setSelected: (
      state,
      {
        payload: { key, selected },
      }: PayloadAction<{ key: string; selected: string[] }>,
    ) => {
      const s = ensure(state, key);
      s.selected = selected;
      s.activeToolbarTab = selected.length > 0 ? "properties" : "symbols";
    },
    setControlStatus: (
      state,
      {
        payload: { key, control },
      }: PayloadAction<{ key: string; control: Control.Status }>,
    ) => {
      ensure(state, key).control = control;
    },
    setLegend: (
      state,
      {
        payload: { key, legend },
      }: PayloadAction<{ key: string; legend: Partial<LegendState> }>,
    ) => {
      const s = ensure(state, key);
      s.legend = { ...s.legend, ...legend };
    },
    setLegendVisible: (
      state,
      { payload: { key, visible } }: PayloadAction<{ key: string; visible: boolean }>,
    ) => {
      ensure(state, key).legend.visible = visible;
    },
    setActiveToolbarTab: (
      state,
      { payload: { key, tab } }: PayloadAction<{ key: string; tab: ToolbarTab }>,
    ) => {
      ensure(state, key).activeToolbarTab = tab;
    },
    setSelectedSymbolGroup: (
      state,
      { payload: { key, group } }: PayloadAction<{ key: string; group: string }>,
    ) => {
      ensure(state, key).selectedSymbolGroup = group;
    },
    setEditable: (
      state,
      { payload: { key, editable } }: PayloadAction<{ key: string; editable: boolean }>,
    ) => {
      ensure(state, key).editable = editable;
    },
    setFitViewOnResize: (
      state,
      {
        payload: { key, fitViewOnResize },
      }: PayloadAction<{ key: string; fitViewOnResize: boolean }>,
    ) => {
      ensure(state, key).fitViewOnResize = fitViewOnResize;
    },
    remove: (state, { payload: { keys } }: PayloadAction<{ keys: string[] }>) => {
      keys.forEach((key) => delete state.schematics[key]);
    },
  },
});

export const {
  setSelected,
  setControlStatus,
  setLegend,
  setLegendVisible,
  setActiveToolbarTab,
  setSelectedSymbolGroup,
  setEditable,
  setFitViewOnResize,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const PERSIST_EXCLUDE = ["schematic.**.selected" as const];
