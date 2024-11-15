// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { mapValues, xy } from "@synnaxlabs/x";

import * as latest from "@/table/migrations";

export type State = latest.State;
export const ZERO_STATE: State = latest.ZERO_STATE;
export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE: SliceState = latest.ZERO_SLICE_STATE;
export type CellState = latest.CellState;

export const SLICE_NAME = "table";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type CreatePayload = latest.State & {
  key: string;
};

export interface SelectCellsPayload {
  key: string;
  mode: "replace" | "add" | "region";
  cells: string[];
}

export const { actions, reducer } = createSlice({
  name: SLICE_NAME,
  initialState: ZERO_SLICE_STATE,
  reducers: {
    create: (state, { payload }: PayloadAction<CreatePayload>) => {
      const table: State = {
        ...ZERO_STATE,
        ...payload,
      };
      state.tables[payload.key] = table;
    },
    selectCells: (state, { payload }: PayloadAction<SelectCellsPayload>) => {
      const { key, mode, cells } = payload;
      const table = state.tables[key];

      if (cells.length === 0) {
        if (mode === "replace") {
          table.cells = mapValues(table.cells, (cell) => ({
            ...cell,
            selected: false,
          }));
        }
        return;
      }

      table.lastSelected = cells[cells.length - 1];

      if (mode === "replace") {
        table.cells = mapValues(table.cells, (cell) => ({
          ...cell,
          selected: cells.includes(cell.key),
        }));
        return;
      }

      if (mode === "add") {
        table.cells = mapValues(table.cells, (cell) => ({
          ...cell,
          selected: cell.selected || cells.includes(cell.key),
        }));
        return;
      }

      const startPos = findPosition(table, table.lastSelected);
      const endPos = findPosition(table, cells[0]);
      if (startPos == null || endPos == null) return;   
      const selected = allCellsInRegion(table, startPos, endPos);
        table.cells = mapValues(table.cells, (cell) => ({
            ...cell,
            selected: selected.includes(cell.key),
        }));
    },
  },
});

export const { create: internalCreate } = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export type Payload = Action["payload"];

export const findPosition = (state: State, key: string): xy.XY | null => {
  const pos: xy.XY = { x: -1, y: -1 };
  state.layout.rows.find((row, i) => {
    const column = row.cells.find((cell, j) => {
      if (cell.key !== key) return false;
      pos.y = i;
      return true;
    });
    if (column == null) return false;
    pos.x = i;
    return true;
  });
  if (pos.x === -1) return null;
  return pos;
};

export const allCellsInRegion = (state: State, start: xy.XY, end: xy.XY): string[] => {
  const cells: string[] = [];
  for (let i = start.x; i <= end.x; i++)
    for (let j = start.y; j <= end.y; j++)
      cells.push(state.layout.rows[i].cells[j].key);
  return cells;
};
