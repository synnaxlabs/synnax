// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type table } from "@synnaxlabs/client";
import { type Drift } from "@synnaxlabs/drift";
import z from "zod";

import { Window } from "@/session/window";

export const stateZ = z.object({
  editable: z.boolean().default(true),
  selectedCells: z.array(z.string()).default([]),
  lastSelected: z.string().nullable().default(null),
  hideIndicators: z.boolean().default(false),
  centered: z.boolean().default(false),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export const windowStateZ = z.record(z.string(), stateZ).default({});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  /** A window is a viewport, so each holds its own view of every table it shows. */
  windows: z.record(z.string(), windowStateZ).default({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "table";

export interface StoreState extends Drift.StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload extends Window.OptionalKeyParams {
  key: table.Key;
}

export interface CreatePayload extends KeyedPayload, NewState {}

export interface SetSelectedCellsPayload extends KeyedPayload {
  cells: string[];
  anchor?: string | null;
}

export interface SetEditablePayload extends KeyedPayload {
  editable?: boolean;
}

export interface SetHideIndicatorsPayload extends KeyedPayload {
  hideIndicators?: boolean;
}

export interface SetCenteredPayload extends KeyedPayload {
  centered?: boolean;
}

export interface RemovePayload {
  keys: string[];
}

const withSelectedState = Window.createWithDocumentHandler(stateZ);
const initializeDocument = Window.createDocumentInitializer(stateZ);

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: initializeDocument<CreatePayload, SliceState>,
    setSelectedCells: withSelectedState<SetSelectedCellsPayload, SliceState>(
      (state, { payload }) => {
        state.selectedCells = payload.cells;
        if (payload.anchor !== undefined) state.lastSelected = payload.anchor;
      },
    ),
    setEditable: withSelectedState<SetEditablePayload, SliceState>(
      (state, { payload: { editable } }) => {
        state.editable = editable ?? !state.editable;
        if (!state.editable) {
          state.selectedCells = [];
          state.lastSelected = null;
        }
      },
    ),
    setHideIndicators: withSelectedState<SetHideIndicatorsPayload, SliceState>(
      (state, { payload: { hideIndicators } }) => {
        state.hideIndicators = hideIndicators ?? !state.hideIndicators;
      },
    ),
    setCentered: withSelectedState<SetCenteredPayload, SliceState>(
      (state, { payload: { centered } }) => {
        state.centered = centered ?? !state.centered;
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      Window.removeDocuments(state, payload.keys);
    },
  },
  extraReducers: Window.handleRemoved,
});

export const {
  create,
  setSelectedCells,
  setEditable,
  setHideIndicators,
  setCentered,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

export const MIDDLEWARE = [
  Window.createInjectKeyMiddleware([
    create,
    setSelectedCells,
    setEditable,
    setHideIndicators,
    setCentered,
  ]),
];
