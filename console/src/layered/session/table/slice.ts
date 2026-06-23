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
import z from "zod";

export const stateZ = z.object({
  editable: z.boolean().default(true),
  selectedCells: z.array(z.string()).default([]),
  lastSelected: z.string().nullable().default(null),
  // hideIndicators hides the row/column indicator strips when editable is
  // false. The setting only takes effect outside edit mode; entering edit mode
  // always shows them.
  hideIndicators: z.boolean().default(false),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const ZERO_STATE = stateZ.parse({});

export const sliceStateZ = z.object({
  version: z.literal(0).default(0),
  tables: z.record(z.string(), stateZ).default({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const ZERO_SLICE_STATE = sliceStateZ.parse({});

export const SLICE_NAME = "table";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface KeyedPayload {
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

export interface RemovePayload {
  keys: string[];
}

const withSelectedState =
  <Payload extends KeyedPayload, Type extends string = string>(
    handler?: (state: State, action: PayloadAction<Payload, Type>) => void,
  ) =>
  (state: SliceState, action: PayloadAction<Payload, Type>) => {
    const {
      payload: { key },
    } = action;
    let s = state.tables[key];
    if (s == null) {
      s = stateZ.parse({});
      state.tables[key] = s;
    }
    handler?.(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (payload.key in state.tables) return;
      state.tables[payload.key] = stateZ.parse(payload);
    },
    setSelectedCells: withSelectedState(
      (state, { payload }: PayloadAction<SetSelectedCellsPayload>) => {
        state.selectedCells = payload.cells;
        if (payload.anchor !== undefined) state.lastSelected = payload.anchor;
      },
    ),
    setEditable: withSelectedState(
      (state, { payload: { editable } }: PayloadAction<SetEditablePayload>) => {
        state.editable = editable ?? !state.editable;
        if (!state.editable) {
          state.selectedCells = [];
          state.lastSelected = null;
        }
      },
    ),
    setHideIndicators: withSelectedState(
      (
        state,
        { payload: { hideIndicators } }: PayloadAction<SetHideIndicatorsPayload>,
      ) => {
        state.hideIndicators = hideIndicators ?? !state.hideIndicators;
      },
    ),
    remove: (state, { payload }: PayloadAction<RemovePayload>) => {
      payload.keys.forEach((key) => delete state.tables[key]);
    },
  },
});

export const {
  create: internalCreate,
  setSelectedCells,
  setEditable,
  setHideIndicators,
  remove,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;
export type Payload = Action["payload"];

// purgeState resets transient cell selection so a restored table opens without
// stale highlights pointing at cells that may no longer exist.
export const purgeState = (state: State): State => {
  state.selectedCells = [];
  state.lastSelected = null;
  return state;
};

export const purgeSliceState = <S extends StoreState>(state: S): S => {
  Object.values(state[SLICE_NAME].tables).forEach(purgeState);
  return state;
};

export const PERSIST_EXCLUDE = [purgeSliceState];
