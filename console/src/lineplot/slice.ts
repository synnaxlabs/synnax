// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Viewport } from "@synnaxlabs/pluto";
import { type lineplot } from "@synnaxlabs/pluto/ether";
import { array, deep } from "@synnaxlabs/x";

import * as latest from "@/lineplot/types";
import { type RootState } from "@/store";

export type ViewportState = latest.ViewportState;
export type SelectionState = latest.SelectionState;
export type AnnotationsState = latest.AnnotationsState;
export type State = latest.State;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export type ClickMode = latest.ClickMode;
export type ControlState = latest.ControlState;
export type SliceState = latest.SliceState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_ANNOTATIONS_STATE = latest.ZERO_ANNOTATIONS_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const anyStateZ = latest.anyStateZ;

export const SLICE_NAME = "line";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type CreatePayload = latest.AnyState;

export interface RemovePayload {
  keys: string[];
}

export interface SetViewportPayload extends Partial<
  Omit<ViewportState, "renderTrigger">
> {
  key: string;
}

export interface StoreViewportPayload extends Omit<ViewportState, "renderTrigger"> {
  key: string;
}

export interface SetSelectionPayload extends SelectionState {
  key: string;
}

export interface SetActiveToolbarTabPayload {
  key: string;
  tab: ToolbarTab;
}

export interface SetControlStatePayload {
  key: string;
  state: Partial<ControlState>;
}

export interface SetViewportModePayload {
  key: string;
  mode: Viewport.Mode;
}

export interface SelectRulePayload {
  key: string;
  ruleKey: string;
}

export interface SetRemoteCreatedPayload {
  key: string;
}

export interface SetMeasureModePayload {
  key: string;
  mode: lineplot.measure.Mode;
}

export interface SetRangeAnnotationsVisiblePayload {
  key: string;
  visible: boolean;
}

export interface SetLineVisiblePayload {
  key: string;
  lineKey: string;
  visible: boolean;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const parsed = anyStateZ.safeParse(payload);
      const migrated = parsed.success ? parsed.data : migrateState(payload);
      const { key: layoutKey } = migrated;
      const existing = state.plots[layoutKey];
      if (existing != null && existing.version === migrated.version) return;
      state.plots[layoutKey] = migrated;
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((k) => {
        delete state.plots[k];
      });
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const p = state.plots[payload.key];
      p.viewport = {
        ...deep.copy(latest.ZERO_VIEWPORT_STATE),
        ...payload,
        renderTrigger: p.viewport.renderTrigger + 1,
      };
      p.selection = { ...latest.ZERO_SELECTION_STATE };
    },
    storeViewport: (state, { payload }: PayloadAction<StoreViewportPayload>) => {
      const p = state.plots[payload.key];
      p.viewport = { ...state.plots[payload.key].viewport, ...payload };
      p.selection = { ...latest.ZERO_SELECTION_STATE };
    },
    setSelection: (state, { payload }: PayloadAction<SetSelectionPayload>) => {
      state.plots[payload.key].selection = {
        ...state.plots[payload.key].selection,
        ...payload,
      };
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      state.plots[payload.key].toolbar.activeTab = payload.tab;
    },
    setControlState: (state, { payload }: PayloadAction<SetControlStatePayload>) => {
      state.plots[payload.key].control = {
        ...state.plots[payload.key].control,
        ...payload.state,
      };
    },
    setViewportMode: (
      state,
      { payload: { key, mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.plots[key].mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      state.plots[payload.key].remoteCreated = true;
    },
    setSelectedRule: (
      state,
      { payload }: PayloadAction<{ key: string; ruleKey: string | string[] }>,
    ) => {
      const plot = state.plots[payload.key];
      plot.selectedRules = array.toArray(payload.ruleKey);
      plot.toolbar.activeTab = "annotations";
    },
    setMeasureMode: (state, { payload }: PayloadAction<SetMeasureModePayload>) => {
      state.plots[payload.key].measure.mode = payload.mode;
    },
    setRangeAnnotationsVisible: (
      state,
      { payload }: PayloadAction<SetRangeAnnotationsVisiblePayload>,
    ) => {
      state.plots[payload.key].annotations.visible = payload.visible;
    },
    setLineVisible: (state, { payload }: PayloadAction<SetLineVisiblePayload>) => {
      const plot = state.plots[payload.key];
      if (plot == null) return;
      const hidden = new Set(plot.hiddenLines);
      if (payload.visible) hidden.delete(payload.lineKey);
      else hidden.add(payload.lineKey);
      plot.hiddenLines = Array.from(hidden);
    },
  },
});

export const {
  remove,
  setViewport,
  setActiveToolbarTab,
  setControlState,
  storeViewport,
  setViewportMode,
  setSelectedRule,
  setRemoteCreated,
  setSelection,
  setMeasureMode,
  setRangeAnnotationsVisible,
  setLineVisible,
  create: internalCreate,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

const purgeState = (state: latest.State): latest.State => {
  state.hiddenLines = [];
  return state;
};

export const purgeSliceState = (state: RootState): RootState => {
  Object.values(state[SLICE_NAME].plots).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];
