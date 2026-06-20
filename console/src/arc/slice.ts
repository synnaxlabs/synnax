// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type arc } from "@synnaxlabs/client";
import { type Diagram, type Viewport } from "@synnaxlabs/pluto";

import * as latest from "@/arc/types";

export type SliceState = latest.SliceState;
export type State = latest.State;
export type ToolbarTab = latest.ToolbarTab;
export type ToolbarState = latest.ToolbarState;
export const ZERO_STATE = latest.ZERO_STATE;
export const ZERO_SLICE_STATE = latest.ZERO_SLICE_STATE;
export const migrateSlice = latest.migrateSlice;
export const migrateState = latest.migrateState;
export const anyStateZ = latest.anyStateZ;
export const TYPE = latest.TYPE;
export const ZERO_GRAPH = latest.ZERO_GRAPH;

export const SLICE_NAME = "arc";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const PERSIST_EXCLUDE = [];

export interface SetViewportPayload {
  key: string;
  viewport: Diagram.Viewport;
}

export type CreatePayload = {
  key: string;
  mode?: arc.Mode;
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

export interface SetActiveToolbarTabPayload {
  tab: ToolbarTab;
}

export interface SetSelectedPayload {
  key: string;
  selected: string[];
}

export interface SetViewportModePayload {
  mode: Viewport.Mode;
}

export interface SetRemoteCreatedPayload {
  key: string;
}

export interface SetRawTextPayload {
  key: string;
  raw: string;
}

export interface SetModePayload {
  key: string;
  mode: arc.Mode;
}

const clearOtherSelections = (state: SliceState, layoutKey: string): void => {
  Object.keys(state.arcs).forEach((key) => {
    if (key === layoutKey) return;
    state.arcs[key].graph.selected = [];
  });
};

const setActiveTabFromSelection = (
  state: latest.SliceState,
  layoutKey: string,
  hasSelection: boolean,
): void => {
  if (hasSelection) {
    if (state.toolbar.activeTab !== "properties")
      clearOtherSelections(state, layoutKey);
    state.toolbar.activeTab = "properties";
  } else state.toolbar.activeTab = "stages";
};

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: latest.ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const { key } = payload;
      if (state.arcs[key] != null) return;
      state.arcs[key] = { ...ZERO_STATE, ...payload };
      state.toolbar.activeTab = "stages";
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      const { keys: layoutKeys } = payload;
      layoutKeys.forEach((layoutKey) => {
        const arc = state.arcs[layoutKey];
        if (arc == null) return;
        delete state.arcs[layoutKey];
      });
    },
    setSelected: (state, { payload }: PayloadAction<SetSelectedPayload>) => {
      const { key: layoutKey, selected } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.selected = selected;
      setActiveTabFromSelection(state, layoutKey, selected.length > 0);
    },
    setActiveToolbarTab: (
      state,
      { payload }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      const { tab } = payload;
      state.toolbar.activeTab = tab;
    },
    setViewport: (state, { payload }: PayloadAction<SetViewportPayload>) => {
      const { key: layoutKey, viewport } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.viewport = viewport;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { key: layoutKey, editable } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.selected = [];
      arc.graph.editable = editable;
    },
    setFitViewOnResize: (
      state,
      { payload }: PayloadAction<SetFitViewOnResizePayload>,
    ) => {
      const { key: layoutKey, fitViewOnResize } = payload;
      const arc = state.arcs[layoutKey];
      arc.graph.fitViewOnResize = fitViewOnResize;
    },
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.mode = mode;
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      const { key: layoutKey } = payload;
      const arc = state.arcs[layoutKey];
      arc.remoteCreated = true;
    },
    setRawText: (state, { payload }: PayloadAction<SetRawTextPayload>) => {
      const { key: layoutKey, raw } = payload;
      const arc = state.arcs[layoutKey];
      arc.text.raw = raw;
    },
    setMode: (state, { payload }: PayloadAction<SetModePayload>) => {
      const { key, mode } = payload;
      const arc = state.arcs[key];
      if (arc != null) arc.mode = mode;
    },
  },
});

export const {
  remove,
  setSelected,
  setFitViewOnResize,
  create: internalCreate,
  setActiveToolbarTab,
  setViewport,
  setEditable,
  setViewportMode,
  setRemoteCreated,
  setRawText,
  setMode,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
