// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

import * as latest from "@/table/types";

export type State = latest.State;
export const stateZ = latest.stateZ;
export const ZERO_STATE: State = latest.ZERO_STATE;
export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE: SliceState = latest.ZERO_SLICE_STATE;
export const anyStateZ = latest.anyStateZ;
export const migrateSlice = latest.migrateSlice;

export const SLICE_NAME = "table";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export interface CreatePayload {
  key: string;
  editable?: boolean;
}

export type SelectionMode = "replace" | "add" | "region";

export interface RemovePayload {
  keys: string[];
}

export interface SelectCellsPayload {
  key: string;
  mode: SelectionMode;
  // The set of cell keys to apply the mode against. The caller resolves
  // shift-click ranges into the full set before dispatching; the slice only
  // tracks UI selection, so the rectangular-region geometry lives in the
  // consumer (which has access to the Pluto-owned rows/columns) rather than
  // in the slice.
  cells: string[];
  // anchor records the most-recently-clicked cell so subsequent shift-clicks
  // (mode === "region") can build the range relative to it. Defaults to the
  // last entry in cells when omitted.
  anchor?: string;
}

export interface SetEditablePayload {
  key: string;
  editable?: boolean;
}

export interface SetSelectedCellsPayload {
  key: string;
  cells: string[];
  anchor?: string | null;
}

export interface SetHideIndicatorsPayload {
  key: string;
  hideIndicators?: boolean;
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      if (state.tables[payload.key] != null) return;
      state.tables[payload.key] = {
        ...ZERO_STATE,
        key: payload.key,
        editable: payload.editable ?? ZERO_STATE.editable,
      };
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      for (const k of keys) delete state.tables[k];
    },
    selectCells: (state, { payload }: PayloadAction<SelectCellsPayload>) => {
      const { key, mode, cells, anchor } = payload;
      const table = state.tables[key];
      if (table == null || !table.editable) return;
      if (cells.length === 0) {
        if (mode === "replace") {
          table.selectedCells = [];
          table.lastSelected = null;
        }
        return;
      }
      if (mode === "add") {
        const next = new Set(table.selectedCells);
        for (const c of cells) next.add(c);
        table.selectedCells = Array.from(next);
      } else table.selectedCells = cells;
      table.lastSelected = anchor ?? cells[cells.length - 1];
    },
    setSelectedCells: (state, { payload }: PayloadAction<SetSelectedCellsPayload>) => {
      const table = state.tables[payload.key];
      if (table == null) return;
      table.selectedCells = payload.cells;
      if (payload.anchor !== undefined) table.lastSelected = payload.anchor;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { key, editable } = payload;
      const table = state.tables[key];
      if (table == null) return;
      table.editable = editable ?? !table.editable;
      if (!table.editable) {
        table.selectedCells = [];
        table.lastSelected = null;
      }
    },
    setHideIndicators: (
      state,
      { payload }: PayloadAction<SetHideIndicatorsPayload>,
    ) => {
      const t = state.tables[payload.key];
      if (t == null) return;
      t.hideIndicators = payload.hideIndicators ?? !t.hideIndicators;
    },
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
export type Payload = Action["payload"];
