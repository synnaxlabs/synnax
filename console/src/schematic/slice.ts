// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Control, type Diagram, type Viewport } from "@synnaxlabs/pluto";

import * as latest from "@/schematic/types";
import { type RootState } from "@/store";

export type SliceState = latest.SliceState;
export type NodeProps = latest.NodeProps;
export type EdgeProps = latest.EdgeProps;
export type State = latest.State;
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

/** Purges fields in schematic state that should not be persisted. */
export const purgeState = (state: State): State => {
  // Reset control states.
  state.control = "released";
  state.controlAcquireTrigger = 0;
  state.toolbar = { ...state.toolbar, activeTab: "symbols" };
  return state;
};

export const purgeSliceState = (state: RootState): RootState => {
  Object.values(state[SLICE_NAME].schematics).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];

export interface SetViewportPayload {
  key: string;
  viewport: Diagram.Viewport;
}

export type CreatePayload = latest.AnyState & {
  key: string;
};

export interface RemovePayload {
  keys: string[];
}

export interface SetEditablePayload {
  key: string;
  editable: boolean;
}

export interface SetFitViewOnResizePayload {
  key: string;
  fitViewOnResize: boolean;
}

export interface SetControlStatusPayload {
  key: string;
  control: Control.Status;
}

export interface ToggleControlPayload {
  key: string;
  status: Control.Status;
}

export interface SetActiveToolbarTabPayload {
  key: string;
  tab: ToolbarTab;
}

export interface SetViewportModePayload {
  key: string;
  mode: Viewport.Mode;
}

export interface SetSelectedSymbolGroupPayload {
  key: string;
  group: string;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const { key: layoutKey, viewport } = payload;
      const schematic = state.schematics[layoutKey];
      schematic.viewport = viewport;
    },
    setControlStatus: (state, { payload }: PayloadAction<SetControlStatusPayload>) => {
      const { key: layoutKey, control } = payload;
      const schematic = state.schematics[layoutKey];
      if (schematic == null) return;
      schematic.control = control;
      if (control === "acquired") {
        schematic.selected = [];
        schematic.editable = false;
      }
    },
    setViewportMode: (
      state,
      { payload: { key, mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.schematics[key].mode = mode;
    },
    setSelectedSymbolGroup: (
      state,
      { payload }: PayloadAction<SetSelectedSymbolGroupPayload>,
    ) => {
      const { key, group } = payload;
      state.schematics[key].toolbar.selectedSymbolGroup = group;
    },
  },
});

export const { setControlStatus, setViewportMode, setSelectedSymbolGroup } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
