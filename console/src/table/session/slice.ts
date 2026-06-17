// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { Table } from "@synnaxlabs/pluto";
import z from "zod";

import { Keyed } from "@/keyed";

export const stateZ = z.object({
  editable: z.boolean().default(true),
  // hideIndicators controls whether the row/column indicator strips are hidden when
  // editable is false. The setting only takes effect outside edit mode; entering edit
  // mode always shows them.
  hideIndicators: z.boolean().default(false),
  selectedCells: z.array(z.string()).default([]),
  lastSelected: z.string().nullable().default(null),
});
export interface State extends z.infer<typeof stateZ> {}
export interface NewState extends z.input<typeof stateZ> {}

export const sliceStateZ = z.object({
  tables: z.record(z.string(), stateZ).prefault({}),
});
export interface SliceState extends z.infer<typeof sliceStateZ> {}

export const SLICE_NAME = "table";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type SelectionMode = "replace" | "add" | "region";

export interface CreatePayload extends Keyed.Payload, NewState {}

export interface RemovePayload {
  keys: string[];
}

export interface SelectCellsPayload extends Keyed.Payload {
  mode: SelectionMode;
  // cells is the set of cell keys to apply the mode against. The caller resolves
  // shift-click ranges into the full set before dispatching; the slice only tracks UI
  // selection, so the rectangular-region geometry lives in the consumer.
  cells: string[];
  // anchor records the most-recently-clicked cell so subsequent shift-clicks
  // (mode === "region") can build the range relative to it. Defaults to the last entry
  // in cells when omitted.
  anchor?: string;
}

export interface SetSelectedCellsPayload extends Keyed.Payload {
  cells: string[];
  anchor?: string | null;
}

export interface SetEditablePayload extends Keyed.Payload {
  editable?: boolean;
}

export interface SetHideIndicatorsPayload extends Keyed.Payload {
  hideIndicators?: boolean;
}

const withSelectedState =
  <Payload extends Keyed.Payload, Type extends string = string>(
    handler: (state: State, action: PayloadAction<Payload, Type>) => void,
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
    handler(s, action);
  };

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: sliceStateZ.parse({}),
  reducers: {
    create: (state, { payload: { key, ...rest } }: PayloadAction<CreatePayload>) => {
      if (!(key in state.tables)) state.tables[key] = stateZ.parse(rest);
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      for (const k of keys) delete state.tables[k];
    },
    selectCells: withSelectedState(
      (
        state,
        { payload: { mode, cells, anchor } }: PayloadAction<SelectCellsPayload>,
      ) => {
        if (!state.editable) return;
        if (cells.length === 0) {
          if (mode === "replace") {
            state.selectedCells = [];
            state.lastSelected = null;
          }
          return;
        }
        if (mode === "add") {
          const next = new Set(state.selectedCells);
          for (const c of cells) next.add(c);
          state.selectedCells = Array.from(next);
        } else state.selectedCells = cells;
        state.lastSelected = anchor ?? cells[cells.length - 1];
      },
    ),
    setSelectedCells: withSelectedState(
      (
        state,
        { payload: { cells, anchor } }: PayloadAction<SetSelectedCellsPayload>,
      ) => {
        state.selectedCells = cells;
        if (anchor !== undefined) state.lastSelected = anchor;
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
  },
});

export const {
  create: internalCreate,
  remove,
  selectCells,
  setSelectedCells,
  setEditable,
  setHideIndicators,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export const selectSliceState = (state: StoreState): SliceState => state[SLICE_NAME];

export const select = (state: StoreState, key: string): State =>
  selectSliceState(state).tables[key] ?? stateZ.parse({});

export const selectEditable = (state: StoreState, key: string): boolean =>
  select(state, key).editable;

export const selectHideIndicators = (state: StoreState, key: string): boolean =>
  select(state, key).hideIndicators;

export const selectSelectedCellKeys = (state: StoreState, key: string): string[] =>
  select(state, key).selectedCells;

export const selectLastSelected = (state: StoreState, key: string): string | null =>
  select(state, key).lastSelected;

export const {
  useSelectState,
  useSelectEditable,
  useSelectHideIndicators,
  useSelectSelectedCellKeys,
  useSelectLastSelected,
} = Keyed.createSelectors(Table.useKey, {
  useSelectState: select,
  useSelectEditable: selectEditable,
  useSelectHideIndicators: selectHideIndicators,
  useSelectSelectedCellKeys: selectSelectedCellKeys,
  useSelectLastSelected: selectLastSelected,
});
