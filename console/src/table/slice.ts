// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { id, type location, mapValues, type xy } from "@synnaxlabs/x";

import * as latest from "@/table/migrations";

export type State = latest.State;
export const ZERO_STATE: State = latest.ZERO_STATE;
export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE: SliceState = latest.ZERO_SLICE_STATE;
export type CellState<T extends string = string, P = unknown> = latest.CellState<T, P>;
export const ZERO_CELL_STATE: CellState = latest.ZERO_CELL_STATE;
export type RowLayout = latest.RowLayout;
export type CellLayout = latest.CellLayout;

export const SLICE_NAME = "table";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type CreatePayload = latest.State & {
  key: string;
};

export type SelectionMode = "replace" | "add" | "region";

export interface SelectCellsPayload {
  key: string;
  mode: SelectionMode;
  cells: string[];
}

export interface AddRowPayload {
  key: string;
  index?: number;
  cellKey?: string;
  loc?: location.Y;
}

export interface AddColPayload {
  key: string;
  index?: number;
  cellKey?: string;
  loc?: location.X;
}

export interface DeleteRowPayload {
  key: string;
  index?: number;
  cellKey?: string;
}

export interface DeleteColPayload {
  key: string;
  index?: number;
  cellKey?: string;
}

export interface SetCellStatePayload {
  key: string;
  state: Partial<CellState> & { key: string };
}

export interface ResizeRowPayload {
  key: string;
  index: number;
  size: number;
}

export interface ResizeColPayload {
  key: string;
  index: number;
  size: number;
}

export interface SelectColPayload {
  key: string;
  index: number;
}

export interface SelectRowPayload {
  key: string;
  index: number;
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
        if (mode === "replace")
          table.cells = mapValues(table.cells, (cell) => ({
            ...cell,
            selected: false,
          }));

        return;
      }

      table.lastSelected = cells[cells.length - 1];

      if (mode === "replace") {
        Object.values(table.cells).forEach((cell) => {
          if (cells.includes(cell.key)) cell.selected = true;
          else cell.selected &&= false;
        });
        return;
      }

      if (mode === "add") {
        table.cells = mapValues(table.cells, (cell) => ({
          ...cell,
          selected: cell.selected || cells.includes(cell.key),
        }));
        return;
      }

      const startPos = findCellPosition(table, table.lastSelected);
      const endPos = findCellPosition(table, cells[0]);
      if (startPos == null || endPos == null) return;
      const selected = allCellsInRegion(table, startPos, endPos);
      table.cells = mapValues(table.cells, (cell) => ({
        ...cell,
        selected: selected.includes(cell.key),
      }));
    },
    addRow: (state, { payload }: PayloadAction<AddRowPayload>) => {
      const { key, index, loc, cellKey } = payload;
      const table = state.tables[key];
      if (table == null) return;

      let newRow: RowLayout;
      if (table.layout.rows.length === 0) {
        const cellKey = id.id();
        table.cells[cellKey] = { ...ZERO_CELL_STATE, key: cellKey };
        newRow = { cells: [{ key: cellKey }], size: 24 };
      } else
        newRow = {
          cells: table.layout.rows[0].cells.map(() => {
            const key = id.id();
            table.cells[key] = { ...ZERO_CELL_STATE, key };
            return { key };
          }),
          size: 24,
        };

      if (cellKey != null && loc != null) {
        const pos = findCellPosition(table, cellKey);
        if (pos == null) return;
        if (loc === "top")
          if (pos.y === 0) table.layout.rows.unshift(newRow);
          else table.layout.rows.splice(pos.y, 0, newRow);
        else if (loc === "bottom")
          if (pos.y === table.layout.rows.length - 1) table.layout.rows.push(newRow);
          else table.layout.rows.splice(pos.y + 1, 0, newRow);
      } else if (index == null) table.layout.rows.push(newRow);
      else table.layout.rows.splice(index, 0, newRow);
    },
    addCol: (state, { payload }: PayloadAction<AddColPayload>) => {
      const { index, loc, cellKey } = payload;
      const table = state.tables[payload.key];
      if (table == null) return;
      if (cellKey != null && loc != null) {
        const pos = findCellPosition(table, cellKey);
        if (pos == null) return;
        table.layout.rows.forEach((row) => {
          const cellKey = id.id();
          if (loc === "left") row.cells.splice(pos.x, 0, { key: cellKey });
          else if (loc === "right") row.cells.splice(pos.x + 1, 0, { key: cellKey });
          table.cells[cellKey] = { ...ZERO_CELL_STATE, key: cellKey };
        });
        table.layout.columns.splice(pos.x, 0, { size: 24 });
        return;
      }

      table.layout.rows.forEach((row) => {
        const cellKey = id.id();
        if (index == null) row.cells.push({ key: cellKey });
        else row.cells.splice(index, 0, { key: cellKey });
        table.cells[cellKey] = { ...ZERO_CELL_STATE, key: cellKey };
      });
      if (index == null) table.layout.columns.push({ size: 24 });
      else table.layout.columns.splice(index, 0, { size: 24 });
    },
    selectRow: (state, { payload }: PayloadAction<SelectRowPayload>) => {
      const { key, index } = payload;
      const table = state.tables[key];
      if (table == null) return;
      table.layout.rows.forEach((row, i) => {
        if (i === index)
          row.cells.forEach((cell) => (table.cells[cell.key].selected = true));
        else row.cells.forEach((cell) => (table.cells[cell.key].selected = false));
      });
    },
    selectCol: (state, { payload }: PayloadAction<SelectColPayload>) => {
      const { key, index } = payload;
      const table = state.tables[key];
      if (table == null) return;
      table.layout.rows.forEach((row) => {
        row.cells.forEach((cell, i) => {
          if (i === index) table.cells[cell.key].selected = true;
          else table.cells[cell.key].selected = false;
        });
      });
    },
    deleteRow: (state, { payload }: PayloadAction<DeleteRowPayload>) => {
      const { key, index, cellKey } = payload;
      const table = state.tables[key];
      if (table == null) return;
      if (cellKey != null) {
        const pos = findCellPosition(table, cellKey);
        if (pos == null) return;
        table.layout.rows.splice(pos.y, 1);
        return;
      }
      if (index == null) return;
      table.layout.rows.splice(index, 1);
    },
    deleteCol: (state, { payload }: PayloadAction<DeleteColPayload>) => {
      const { key, index, cellKey } = payload;
      const table = state.tables[key];
      if (table == null) return;
      if (cellKey != null) {
        const pos = findCellPosition(table, cellKey);
        if (pos == null) return;
        table.layout.rows.forEach((row) => row.cells.splice(pos.x, 1));
        table.layout.columns.splice(pos.x, 1);
        return;
      }
      if (index == null) return;
      table.layout.rows.forEach((row) => row.cells.splice(index, 1));
      table.layout.columns.splice(index, 1);
    },
    setCellState: (state, { payload }: PayloadAction<SetCellStatePayload>) => {
      const { key, state: cState } = payload;
      const table = state.tables[key];
      table.cells[cState.key] = { ...table.cells[cState.key], ...cState };
    },
    resizeRow: (state, { payload }: PayloadAction<ResizeRowPayload>) => {
      const { key, index, size } = payload;
      const table = state.tables[key];
      if (table == null) return;
      table.layout.rows[index].size = size;
    },
    resizeCol: (state, { payload }: PayloadAction<ResizeColPayload>) => {
      const { key, index, size } = payload;
      const table = state.tables[key];
      if (table == null) return;
      table.layout.columns[index].size = size;
    },
  },
});

export const {
  create: internalCreate,
  selectCells,
  addCol,
  addRow,
  setCellState,
  resizeCol,
  resizeRow,
  deleteCol,
  deleteRow,
  selectCol,
  selectRow,
} = actions;

export type Action = ReturnType<(typeof actions)[keyof typeof actions]>;

export type Payload = Action["payload"];

export const findCellPosition = (state: State, key: string): xy.XY | null => {
  const pos: xy.XY = { x: -1, y: -1 };
  state.layout.rows.find((row, j) => {
    const column = row.cells.find((cell, i) => {
      if (cell.key !== key) return false;
      pos.x = i;
      return true;
    });
    if (column == null) return false;
    pos.y = j;
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
