// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";
import { type TableCells } from "@synnaxlabs/pluto";
import { id, type location, record, xy } from "@synnaxlabs/x";

import * as latest from "@/table/types";
import { BASE_COL_SIZE, BASE_ROW_SIZE } from "@/table/types";

export type State = latest.State;
export const stateZ = latest.stateZ;
export const ZERO_STATE: State = latest.ZERO_STATE;
export type SliceState = latest.SliceState;
export const ZERO_SLICE_STATE: SliceState = latest.ZERO_SLICE_STATE;
export type CellState<
  T extends TableCells.Variant = TableCells.Variant,
  P extends object = record.Unknown,
> = latest.CellState<T, P>;
export const ZERO_CELL_STATE: CellState = latest.ZERO_CELL_STATE;
export type RowLayout = latest.RowLayout;
export type CellLayout = latest.CellLayout;
export const ZERO_CELL_PROPS = latest.ZERO_CELL_PROPS;

export const SLICE_NAME = "table";

export interface StoreState {
  [SLICE_NAME]: SliceState;
}

export type CreatePayload = latest.State & {
  key: string;
};

export type SelectionMode = "replace" | "add" | "region";

export interface RemovePayload {
  keys: string[];
}

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

export interface SetCellPropsPayload {
  key: string;
  cellKey: string;
  props: CellState["props"];
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

export interface SetCellVariantPayload {
  key: string;
  cellKey: string;
  variant: TableCells.Variant;
  nextProps?: CellState["props"];
}

export interface SetEditablePayload {
  key: string;
  editable?: boolean;
}

export interface CopySelectedPayload {
  key: string;
}

export interface PasteCelebrationPayload {
  key: string;
}

export interface ClearSelectedPayload {
  key: string;
}

export interface SetRemoteCreatedPayload {
  key: string;
}

const addRowInternal = (
  state: SliceState,
  { payload }: PayloadAction<AddRowPayload>,
) => {
  const { key, index, loc, cellKey } = payload;
  const table = state.tables[key];
  if (table == null) return;

  let newRow: RowLayout;
  // If we have no rows, initialize the table with a single cell and add a column
  // as well.
  if (table.layout.rows.length === 0)
    if (table.layout.columns.length == 0) {
      const cellKey = id.create();
      table.cells[cellKey] = { ...ZERO_CELL_STATE, key: cellKey };
      newRow = { cells: [{ key: cellKey }], size: BASE_ROW_SIZE };
      table.layout.columns = [{ size: BASE_COL_SIZE }];
    } else
      newRow = {
        cells: table.layout.columns.map(() => {
          const key = id.create();
          table.cells[key] = { ...ZERO_CELL_STATE, key };
          return { key };
        }),
        size: BASE_ROW_SIZE,
      };
  // If we have an existing row, use it as a template for the new row.
  else
    newRow = {
      cells: table.layout.rows[0].cells.map(() => {
        const key = id.create();
        table.cells[key] = { ...ZERO_CELL_STATE, key };
        return { key };
      }),
      size: BASE_ROW_SIZE,
    };

  // This means that the user wants to add a row above or below a specific cell.
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
  // This means they clicked on a row, and want to add a row above or below it.
  else if (loc !== "top") table.layout.rows.splice(index + 1, 0, newRow);
  else table.layout.rows.splice(index, 0, newRow);
};

export const addColInternal = (
  state: SliceState,
  { payload }: PayloadAction<AddColPayload>,
) => {
  const { index, loc, cellKey } = payload;
  const table = state.tables[payload.key];
  if (table == null) return;

  // Means the user clicked
  if (cellKey != null && loc != null) {
    const pos = findCellPosition(table, cellKey);
    if (pos == null) return;
    table.layout.rows.forEach((row) => {
      const cellKey = id.create();
      if (loc === "left") row.cells.splice(pos.x, 0, { key: cellKey });
      else if (loc === "right") row.cells.splice(pos.x + 1, 0, { key: cellKey });
      table.cells[cellKey] = { ...ZERO_CELL_STATE, key: cellKey };
    });
    table.layout.columns.splice(pos.x, 0, { size: BASE_COL_SIZE });
    return;
  }

  table.layout.rows.forEach((row) => {
    const cellKey = id.create();
    if (index == null) row.cells.push({ key: cellKey });
    else if (loc !== "left") row.cells.splice(index + 1, 0, { key: cellKey });
    else row.cells.splice(index, 0, { key: cellKey });
    table.cells[cellKey] = { ...ZERO_CELL_STATE, key: cellKey };
  });
  if (index == null) table.layout.columns.push({ size: BASE_COL_SIZE });
  else if (loc !== "left")
    table.layout.columns.splice(index + 1, 0, { size: BASE_COL_SIZE });
  else table.layout.columns.splice(index, 0, { size: BASE_COL_SIZE });
};

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
      if (table == null || !table.editable) return;

      if (cells.length === 0) {
        if (mode === "replace")
          table.cells = record.map(table.cells, (cell) => ({
            ...cell,
            selected: false,
          }));
        return;
      }

      if (mode === "region") {
        if (table.lastSelected == null) return;
        const startPos = findCellPosition(table, table.lastSelected);
        const endPos = findCellPosition(table, cells[0]);
        if (startPos == null || endPos == null) return;
        const selected = allCellsInRegion(table, startPos, endPos);
        table.cells = record.map(table.cells, (cell) => ({
          ...cell,
          selected: selected.includes(cell.key),
        }));
      } else if (mode === "add")
        table.cells = record.map(table.cells, (cell) => ({
          ...cell,
          selected: cell.selected || cells.includes(cell.key),
        }));
      else
        Object.values(table.cells).forEach((cell) => {
          if (cells.includes(cell.key)) cell.selected = true;
          else cell.selected &&= false;
        });

      table.lastSelected = cells[cells.length - 1];
    },
    remove: (state, { payload: { keys } }: PayloadAction<RemovePayload>) => {
      keys.forEach((k) => {
        delete state.tables[k];
      });
    },
    addRow: addRowInternal,
    addCol: addColInternal,
    selectRow: (state, { payload }: PayloadAction<SelectRowPayload>) => {
      const { key, index } = payload;
      const table = state.tables[key];
      if (table == null || !table.editable) return;
      table.layout.rows.forEach((row, i) => {
        if (i === index)
          row.cells.forEach((cell) => (table.cells[cell.key].selected = true));
        else row.cells.forEach((cell) => (table.cells[cell.key].selected = false));
      });
    },
    selectCol: (state, { payload }: PayloadAction<SelectColPayload>) => {
      const { key, index } = payload;
      const table = state.tables[key];
      if (table == null || !table.editable) return;
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
      let pos = index;
      if (cellKey != null) {
        const cellPos = findCellPosition(table, cellKey);
        if (cellPos == null) return;
        pos = cellPos.y;
      }
      if (pos == null) return;
      table.layout.rows[pos].cells.forEach((cell) => delete table.cells[cell.key]);
      table.layout.rows.splice(pos, 1);
    },
    deleteCol: (
      state,
      { payload: { key, index, cellKey } }: PayloadAction<DeleteColPayload>,
    ) => {
      const table = state.tables[key];
      if (table == null) return;
      let pos = index;
      if (pos == null) {
        if (cellKey == null) return;
        const cellPos = findCellPosition(table, cellKey);
        if (cellPos == null) return;
        pos = cellPos.x;
      }
      table.layout.rows.forEach((row) => {
        delete table.cells[row.cells[pos].key];
        row.cells.splice(pos, 1);
      });
      table.layout.columns.splice(pos, 1);
    },
    setCellProps: (state, { payload }: PayloadAction<SetCellPropsPayload>) => {
      const { key, cellKey, props } = payload;
      const table = state.tables[key];
      table.cells[cellKey].props = props;
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
    setCellType: (state, { payload }: PayloadAction<SetCellVariantPayload>) => {
      const { key, cellKey, variant, nextProps } = payload;
      const table = state.tables[key];
      const cell = table.cells[cellKey];
      if (cell == null) return;
      cell.variant = variant;
      if (nextProps != null) cell.props = nextProps;
    },
    setEditable: (state, { payload }: PayloadAction<SetEditablePayload>) => {
      const { key, editable } = payload;
      const table = state.tables[key];
      if (table == null) return;
      if (editable == null) table.editable = !table.editable;
      else table.editable = editable;
      if (!table.editable) {
        Object.values(table.cells).forEach((cell) => (cell.selected = false));
        table.lastSelected = "";
      }
    },
    copySelected: (state, { payload }: PayloadAction<CopySelectedPayload>) => {
      const table = state.tables[payload.key];
      if (table == null) return;

      // Grab all currently selected cells, as we assume that is what we want to copy.
      const candidateCells = Object.values(table.cells).filter((cell) => cell.selected);
      // We need to store the positions so we can paste them in the correct location
      // later.
      const candidatePositions = candidateCells
        .map((cell) => findCellPosition(table, cell.key))
        .filter((pos) => pos != null);
      const positionsObj = Object.fromEntries(
        candidatePositions.map((pos, i) => [candidateCells[i].key, pos]),
      );

      if (table.lastSelected == null) return;
      // We only want to copy the contiguous region of cells that the user has selected,
      // using the last selected cell as the reference point for contiguity.
      const contiguousRegion = contiguousRegionFrom(
        positionsObj[table.lastSelected],
        candidatePositions,
      );
      const contiguousCells = contiguousRegion.map(
        (cell) => candidateCells[cell.index],
      );
      // Choose the top left cell as the reference point for the copy buffer. We use
      // this cell as the location to paste.
      const topLeftContiguous = contiguousCells.reduce(
        (min, cell) => {
          const pos = positionsObj[cell.key];
          return {
            x: Math.min(min.x, pos.x),
            y: Math.min(min.y, pos.y),
            key: cell.key,
          };
        },
        { x: Infinity, y: Infinity, key: "" },
      );

      state.copyBuffer = {
        epicenter: topLeftContiguous.key,
        cells: Object.fromEntries(contiguousCells.map((cell) => [cell.key, cell])),
        positions: Object.fromEntries(
          contiguousCells.map((cell) => [cell.key, positionsObj[cell.key]]),
        ),
      };
    },
    pasteSelected: (state, { payload }: PayloadAction<PasteCelebrationPayload>) => {
      const table = state.tables[payload.key];
      if (table == null) return;
      const { copyBuffer } = state;
      if (copyBuffer == null) return;

      // Use the last selected cell as the reference point for pasting.
      const pasteCenter = table.lastSelected;
      if (pasteCenter == null) return;

      const copyCenter = copyBuffer.epicenter;
      const pasteCenterPosition = findCellPosition(table, pasteCenter);
      const copyCenterPosition = copyBuffer.positions[copyCenter];

      if (pasteCenterPosition == null || copyCenterPosition == null) return;

      // Find the translation between the copy center and the paste center.
      const translation = xy.translation(copyCenterPosition, pasteCenterPosition);

      const copiedCells = Object.values(copyBuffer.cells);

      const tableDims = {
        width: table.layout.columns.length - 1,
        height: table.layout.rows.length - 1,
      };

      copiedCells.forEach((cell) => {
        // Grab the position of the cell in the original table.
        const copiedPos = copyBuffer.positions[cell.key];
        const pastePos = xy.translate(copiedPos, translation);
        if (pastePos.y > tableDims.height) {
          const delta = pastePos.y - tableDims.height;
          for (let i = 0; i < delta; i++)
            addRowInternal(state, {
              type: addRow.type,
              payload: { key: payload.key, loc: "bottom" },
            });
          tableDims.height += delta;
        }
        if (pastePos.x > tableDims.width) {
          const delta = pastePos.x - tableDims.width;
          for (let i = 0; i < delta; i++)
            addColInternal(state, {
              type: addCol.type,
              payload: { key: payload.key, loc: "right" },
            });
          tableDims.width += delta;
        }

        const existing = getCellAt(table, pastePos);
        if (existing == null) return;
        table.cells[existing.key] = {
          ...existing,
          ...(cell as CellState),
          // Keep the existing cell key so we don't need to update the new layout.
          key: existing.key,
        };
      });
    },
    clearSelected: (state, { payload }: PayloadAction<ClearSelectedPayload>) => {
      const table = state.tables[payload.key];
      if (table == null) return;

      // We want to remove any rows that are completely selected.
      table.layout.rows = table.layout.rows.filter((row) => {
        if (row.cells.every((cell) => table.cells[cell.key].selected)) {
          row.cells.forEach((cell) => delete table.cells[cell.key]);
          return false;
        }
        return true;
      });

      // Identify any columns that are entirely selected.
      const colIdxToDelete = table.layout.columns
        .map((_, i) => {
          if (table.layout.rows.every((row) => table.cells[row.cells[i].key].selected))
            return i;
          return null;
        })
        .filter((col) => col != null);

      // Filter out and delete the columns that are entirely selected.
      table.layout.columns = table.layout.columns.filter((_, col) => {
        if (!colIdxToDelete.includes(col)) return true;
        table.layout.rows.forEach((row) => {
          delete table.cells[row.cells[col].key];
          row.cells.splice(col, 1);
        });
        return false;
      });

      // Clear all selected cells.
      Object.values(table.cells).forEach((cell) => {
        if (!cell.selected) return;
        cell.variant = "text";
        cell.props = { ...ZERO_CELL_PROPS };
      });
    },
    setRemoteCreated: (state, { payload }: PayloadAction<SetRemoteCreatedPayload>) => {
      state.tables[payload.key].remoteCreated = true;
    },
  },
});

export const {
  create: internalCreate,
  selectCells,
  addCol,
  clearSelected,
  addRow,
  setCellProps,
  remove,
  resizeCol,
  resizeRow,
  deleteCol,
  deleteRow,
  setCellType,
  selectCol,
  selectRow,
  setEditable,
  copySelected,
  pasteSelected,
  setRemoteCreated,
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

export const getCellAt = (state: State, pos: xy.XY): CellState | null => {
  const row = state.layout.rows[pos.y];
  if (row == null) return null;
  return state.cells[row.cells[pos.x].key] as CellState;
};

export const allCellsInRegion = (state: State, start: xy.XY, end: xy.XY): string[] => {
  const cells: string[] = [];
  const minX = Math.min(start.x, end.x);
  const maxX = Math.max(start.x, end.x);
  const minY = Math.min(start.y, end.y);
  const maxY = Math.max(start.y, end.y);

  for (let y = minY; y <= maxY; y++)
    for (let x = minX; x <= maxX; x++) {
      const row = state.layout.rows[y];
      if (row?.cells[x]) cells.push(row.cells[x].key);
    }

  return cells;
};

type XYWithIndex = { x: number; y: number; index: number };

// The function finds all contiguous points starting from 'pos' in the 'points' array,
// and includes their indexes from the 'points' array.
const contiguousRegionFrom = (pos: xy.XY, points: xy.XY[]): XYWithIndex[] => {
  // Create a map for quick lookup of point indexes by their coordinates
  const positionMap = new Map<string, number>();
  for (let i = 0; i < points.length; i++) {
    const point = points[i];
    const key = `${point.x},${point.y}`;
    positionMap.set(key, i);
  }

  const visited = new Set<string>(); // To keep track of visited positions
  const queue: xy.XY[] = [pos]; // Queue for BFS traversal
  const region: XYWithIndex[] = []; // To store the contiguous region points with indexes

  while (queue.length > 0) {
    const current = queue.shift()!;
    const key = `${current.x},${current.y}`;

    if (visited.has(key)) continue;
    visited.add(key);

    // Check if the current point exists in the positionMap
    if (!positionMap.has(key)) continue;

    const index = positionMap.get(key)!;
    const point = points[index];
    region.push({ x: point.x, y: point.y, index });

    // Define the four adjacent positions (up, down, left, right)
    const neighbors = [
      { x: current.x + 1, y: current.y },
      { x: current.x - 1, y: current.y },
      { x: current.x, y: current.y + 1 },
      { x: current.x, y: current.y - 1 },
    ];

    for (const neighbor of neighbors) {
      const neighborKey = `${neighbor.x},${neighbor.y}`;
      if (!visited.has(neighborKey)) queue.push(neighbor);
    }
  }

  return region;
};
