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
import { type Diagram, Viewport } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { z } from "zod";

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

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),

  arcs: z.record(z.string(), stateZ).default({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "arc";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const PERSIST_EXCLUDE = [];

interface KeyedPayload {
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

const withSelectedState =
  <Payload extends KeyedPayload, Type extends string = string>(
    handler?: (state: State, action: PayloadAction<Payload, Type>) => void,
  ) =>
  (state: SliceState, action: PayloadAction<Payload, Type>) => {
    const {
      payload: { key },
    } = action;
    let s = state.arcs[key];
    if (s == null) {
      s = stateZ.parse({});
      state.arcs[key] = s;
    }
    handler?.(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (payload.key in state.arcs) return;
      state.arcs[payload.key] = stateZ.parse(payload);
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.arcs[key]);
    },
    setSelected: withSelectedState(
      (state, { payload }: PayloadAction<SetSelectedPayload>) => {
        const { selected } = payload;
        state.graph.selected = selected;
        if (selected.length > 0) state.toolbar.selectedTab = "properties";
        else state.toolbar.selectedTab = "stages";
      },
    ),
    selectToolbarTab: withSelectedState(
      (state, { payload: { tab } }: PayloadAction<SelectToolbarTabPayload>) => {
        state.toolbar.selectedTab = tab;
      },
    ),
    setViewport: withSelectedState(
      (state, { payload: { viewport } }: PayloadAction<SetViewportPayload>) => {
        state.graph.viewport = { ...state.graph.viewport, ...viewport };
      },
    ),
    setEditable: withSelectedState(
      (state, { payload: { editable } }: PayloadAction<SetEditablePayload>) => {
        state.graph.selected = [];
        state.graph.editable = editable;
      },
    ),
    setFitViewOnResize: withSelectedState(
      (
        state,
        { payload: { fitViewOnResize } }: PayloadAction<SetFitViewOnResizePayload>,
      ) => {
        state.graph.fitViewOnResize = fitViewOnResize;
      },
    ),
    setViewportMode: withSelectedState(
      (state, { payload: { mode } }: PayloadAction<SetViewportModePayload>) => {
        state.graph.viewport.mode = mode;
      },
    ),
  },
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
