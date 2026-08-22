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
import { type Drift } from "@synnaxlabs/drift";
import { type Diagram, Viewport } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { z } from "zod";

import { Window } from "@/session/window";

export const toolbarTabZ = z.enum(["stages", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({ selectedTab: toolbarTabZ.default("stages") });
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}

export const viewportStateZ = z.object({
  position: xy.xyZ.default({ x: 0, y: 0 }),
  zoom: z.number().default(1),
  mode: Viewport.modeZ.default("select"),
});

export const graphStateZ = z.object({
  editable: z.boolean().default(true),
  fitViewOnResize: z.boolean().default(false),
  viewport: viewportStateZ.prefault({}),
  selected: z.array(z.string()).default([]),
});
export interface GraphState extends z.infer<typeof graphStateZ> {}

export const stateZ = z.object({
  graph: graphStateZ.prefault({}),
  toolbar: toolbarStateZ.prefault({}),
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

export const SLICE_NAME = "arc";

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

interface KeyedPayload extends Window.OptionalKeyParams {
  key: arc.Key;
}

export interface CreatePayload extends KeyedPayload, NewState {}

export interface RemovePayload {
  keys: string[];
}

export interface SetViewportPayload extends KeyedPayload {
  viewport: Diagram.Viewport;
}

export interface SetEditablePayload extends KeyedPayload {
  editable: boolean;
}

export interface SetFitViewOnResizePayload extends KeyedPayload {
  fitViewOnResize: boolean;
}

export interface SelectToolbarTabPayload extends KeyedPayload {
  tab: ToolbarTab;
}

export interface SetSelectedPayload extends KeyedPayload {
  selected: string[];
}

export interface SetViewportModePayload extends KeyedPayload {
  mode: Viewport.Mode;
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);
const initializeDocument = Window.createDocumentInitializer(stateZ);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: initializeDocument<CreatePayload, SliceState>,
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      Window.removeDocuments(state, payload.keys);
    },
    setSelected: withSelectedState<SetSelectedPayload, SliceState>(
      (state, { payload }) => {
        const { selected } = payload;
        state.graph.selected = selected;
        if (selected.length > 0) state.toolbar.selectedTab = "properties";
        else state.toolbar.selectedTab = "stages";
      },
    ),
    selectToolbarTab: withSelectedState<SelectToolbarTabPayload, SliceState>(
      (state, { payload: { tab } }) => {
        state.toolbar.selectedTab = tab;
      },
    ),
    setViewport: withSelectedState<SetViewportPayload, SliceState>(
      (state, { payload: { viewport } }) => {
        state.graph.viewport = { ...state.graph.viewport, ...viewport };
      },
    ),
    setEditable: withSelectedState<SetEditablePayload, SliceState>(
      (state, { payload: { editable } }) => {
        state.graph.selected = [];
        state.graph.editable = editable;
      },
    ),
    setFitViewOnResize: withSelectedState<SetFitViewOnResizePayload, SliceState>(
      (state, { payload: { fitViewOnResize } }) => {
        state.graph.fitViewOnResize = fitViewOnResize;
      },
    ),
    setViewportMode: withSelectedState<SetViewportModePayload, SliceState>(
      (state, { payload: { mode } }) => {
        state.graph.viewport.mode = mode;
      },
    ),
  },
  extraReducers: Window.handleRemoved,
});

export const {
  remove,
  setSelected,
  setFitViewOnResize,
  create,
  selectToolbarTab,
  setViewport,
  setEditable,
  setViewportMode,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    setSelected,
    setFitViewOnResize,
    create,
    selectToolbarTab,
    setViewport,
    setEditable,
    setViewportMode,
  ]),
];
