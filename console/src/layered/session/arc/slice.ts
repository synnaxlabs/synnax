// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type Diagram, Viewport } from "@synnaxlabs/pluto";
import { xy } from "@synnaxlabs/x";
import { z } from "zod";

export const toolbarTabZ = z.enum(["stages", "properties"]);
export type ToolbarTab = z.infer<typeof toolbarTabZ>;

export const toolbarStateZ = z.object({ activeTab: toolbarTabZ.default("stages") });
export interface ToolbarState extends z.infer<typeof toolbarStateZ> {}

export const viewportZ = z.object({
  position: xy.xyZ.default({ x: 0, y: 0 }),
  zoom: z.number().default(1),
});

// graphStateZ holds only the session UI state for the graph view. The graph document
// itself (nodes, edges, configs) lives in the flux store, server-synced via Arc actions.
export const graphStateZ = z.object({
  editable: z.boolean().default(true),
  fitViewOnResize: z.boolean().default(false),
  viewport: viewportZ.prefault({}),
  selected: z.array(z.string()).default([]),
});
export interface GraphState extends z.infer<typeof graphStateZ> {}

export const stateZ = z.object({ graph: graphStateZ.prefault({}) });
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  mode: Viewport.modeZ.default("select"),
  toolbar: toolbarStateZ.prefault({}),
  arcs: z.record(z.string(), stateZ).default({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "arc";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export const PERSIST_EXCLUDE = [];

export interface CreatePayload extends NewState {
  key: string;
}

export interface RemovePayload {
  keys: string[];
}

export interface SetViewportPayload {
  key: string;
  viewport: Diagram.Viewport;
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

interface KeyedPayload {
  key: string;
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
      state.toolbar.activeTab = "stages";
    },
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.arcs[key]);
    },
    setSelected: (state, { payload }: PayloadAction<SetSelectedPayload>) => {
      const { key, selected } = payload;
      let s = state.arcs[key];
      if (s == null) {
        s = stateZ.parse({});
        state.arcs[key] = s;
      }
      s.graph.selected = selected;
      if (selected.length > 0) {
        if (state.toolbar.activeTab !== "properties")
          Object.keys(state.arcs).forEach((other) => {
            if (other === key) return;
            state.arcs[other].graph.selected = [];
          });
        state.toolbar.activeTab = "properties";
      } else state.toolbar.activeTab = "stages";
    },
    setActiveToolbarTab: (
      state,
      { payload: { tab } }: PayloadAction<SetActiveToolbarTabPayload>,
    ) => {
      state.toolbar.activeTab = tab;
    },
    setViewport: withSelectedState(
      (state, { payload: { viewport } }: PayloadAction<SetViewportPayload>) => {
        state.graph.viewport = viewport;
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
    setViewportMode: (
      state,
      { payload: { mode } }: PayloadAction<SetViewportModePayload>,
    ) => {
      state.mode = mode;
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
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
