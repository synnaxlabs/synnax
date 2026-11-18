// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { patch } from "@synnaxlabs/x";
import { type Draft } from "immer";

import { actions } from "@/actions";
import {
  type Action,
  addCol,
  addRow,
  createReduceAll,
  type HandlerResult,
  type Handlers,
  moveCol,
  moveRow,
  removeCol,
  removeRow,
  resizeCol,
  resizeRow,
  setCell,
} from "@/table/actions.gen";
import {
  type Cell,
  type CellConfig,
  cellConfigZ,
  columnZ,
  rowZ,
  type Table,
  tableZ,
} from "@/table/types.gen";

const MIN_CELL_DIM = 32;
// baseRow and baseCol are the entries an action gives the opposing axis when it
// bootstraps that axis on an empty table. Both take their size from the schema.
const baseRow = (key: string) => rowZ.parse({ key });
const baseCol = (key: string) => columnZ.parse({ key });

// deriveKey returns the index-th key derived from base. Both reducers run the same
// scheme so optimistic client state agrees with the server.
const deriveKey = (base: string, index: number): string => {
  const suffix = index.toString(16).padStart(4, "0");
  if (base.length < 36) return `${base}-${suffix}`;
  return base.slice(0, 32) + suffix;
};

const findIndex = (items: Array<{ key: string }>, key: string): number =>
  items.findIndex((item) => item.key === key);

// move relocates the element at from to index to, shifting the elements between them by
// one.
const move = <T>(items: T[], from: number, to: number): void => {
  items.splice(to, 0, ...items.splice(from, 1));
};

// expandTemplate returns count replicas of template with keys derived via deriveKey.
// Each replica holds its own copy of the config, as an edit to one changes it in place.
const expandTemplate = (template: Cell, count: number): Cell[] =>
  Array.from({ length: count }, (_, i) => ({
    key: deriveKey(template.key, i),
    config: structuredClone(template.config),
  }));

const ROWS_FIELD = "rows";
const CELLS_FIELD = "cells";
const CONFIG_FIELD = "config";

// A path into the rows goes into the config of one cell. The result targets the cell.
const applyPath = (state: Draft<Table>, action: patch.Action): HandlerResult => {
  const path = patch.pathOf(action);
  if (path[0] !== ROWS_FIELD) return actions.applyPath(state, action, tableZ);
  if (path.length < 5 || path[2] !== CELLS_FIELD || path[4] !== CONFIG_FIELD)
    throw new Error("[Table] - a path into rows must go into the config of one cell");
  const result = actions.applyPath(state, action, tableZ);
  if (result.inverse.length === 0) return result;
  return { ...result, targets: [path[3]] };
};

const keysOf = (cells: Cell[]): string[] => cells.map((c) => c.key);

// setConfig replaces the config of the cell with the key, and returns the replaced
// config. Returns undefined when no row holds the cell.
const setConfig = (
  state: Draft<Table>,
  key: string,
  config: CellConfig,
): CellConfig | undefined => {
  for (const row of state.rows) {
    const i = row.cells.findIndex((c) => c.key === key);
    if (i === -1) continue;
    const prev = actions.snapshotDraft(row.cells[i].config);
    row.cells[i].config = config;
    return prev;
  }
  return undefined;
};

// removeColAt removes the cell at index from every row that holds one, and returns the
// removed cells. A row with no cell at index is skipped, not padded.
const removeColAt = (state: Draft<Table>, index: number): Cell[] => {
  const removed: Cell[] = [];
  for (const row of state.rows) {
    if (index >= row.cells.length) continue;
    removed.push(actions.snapshotDraft(row.cells[index]));
    row.cells.splice(index, 1);
  }
  return removed;
};

const handlers: Handlers = {
  create: (state, payload) => {
    Object.assign(state, payload.table);
    return { inverse: [], targets: [payload.table.key] };
  },

  addRow: (state, payload) => {
    const cells = payload.cellTemplate
      ? expandTemplate(
          payload.cellTemplate,
          state.columns.length === 0 && state.rows.length === 0
            ? 1
            : state.columns.length,
        )
      : payload.cells;
    if (state.columns.length === 0 && cells.length > 0)
      for (let i = 0; i < cells.length; i++)
        state.columns.push(baseCol(deriveKey(payload.key, i)));
    const index = Math.min(payload.index, state.rows.length);
    state.rows.splice(index, 0, {
      key: payload.key,
      size: Math.max(payload.size, MIN_CELL_DIM),
      cells: [...cells],
    });
    return {
      inverse: [removeRow({ key: payload.key })],
      targets: [payload.key, ...keysOf(cells)],
    };
  },

  removeRow: (state, { key }) => {
    const index = findIndex(state.rows, key);
    if (index === -1) return actions.NO_OP_RESULT;
    const { size, cells } = actions.snapshotDraft(state.rows[index]);
    state.rows.splice(index, 1);
    return {
      inverse: [addRow({ index, key, size, cells })],
      targets: [key, ...keysOf(cells)],
    };
  },

  addCol: (state, payload) => {
    const cells = payload.cellTemplate
      ? expandTemplate(
          payload.cellTemplate,
          state.rows.length === 0 && state.columns.length === 0 ? 1 : state.rows.length,
        )
      : payload.cells;
    if (state.rows.length === 0 && cells.length > 0)
      for (let i = 0; i < cells.length; i++)
        state.rows.push(baseRow(deriveKey(payload.key, i)));
    const idx = Math.min(payload.index, state.columns.length);
    state.columns.splice(idx, 0, {
      key: payload.key,
      size: Math.max(payload.size, MIN_CELL_DIM),
    });
    const added = cells.slice(0, state.rows.length);
    added.forEach((cell, i) => {
      const row = state.rows[i];
      row.cells.splice(Math.min(idx, row.cells.length), 0, cell);
    });
    return {
      inverse: [removeCol({ key: payload.key })],
      targets: [payload.key, ...keysOf(added)],
    };
  },

  removeCol: (state, { key }) => {
    const index = findIndex(state.columns, key);
    if (index === -1) return actions.NO_OP_RESULT;
    const { size } = state.columns[index];
    state.columns.splice(index, 1);
    // The inverse repopulates only the rows that held a cell at this column.
    const cells = removeColAt(state, index);
    return {
      inverse: [addCol({ index, key, size, cells })],
      targets: [key, ...keysOf(cells)],
    };
  },

  resizeRow: (state, { key, size }) => {
    const index = findIndex(state.rows, key);
    if (index === -1) return actions.NO_OP_RESULT;
    const oldSize = state.rows[index].size;
    state.rows[index].size = Math.max(size, MIN_CELL_DIM);
    return { inverse: [resizeRow({ key, size: oldSize })], targets: [key] };
  },

  resizeCol: (state, { key, size }) => {
    const index = findIndex(state.columns, key);
    if (index === -1) return actions.NO_OP_RESULT;
    const oldSize = state.columns[index].size;
    state.columns[index].size = Math.max(size, MIN_CELL_DIM);
    return { inverse: [resizeCol({ key, size: oldSize })], targets: [key] };
  },

  moveRow: (state, { key, to }) => {
    const from = findIndex(state.rows, key);
    if (from === -1) return actions.NO_OP_RESULT;
    move(state.rows, from, Math.min(to, state.rows.length - 1));
    return { inverse: [moveRow({ key, to: from })], targets: [key] };
  },

  moveCol: (state, { key, to }) => {
    const from = findIndex(state.columns, key);
    if (from === -1) return actions.NO_OP_RESULT;
    const clamped = Math.min(to, state.columns.length - 1);
    move(state.columns, from, clamped);
    for (const row of state.rows)
      if (Math.max(from, clamped) < row.cells.length) move(row.cells, from, clamped);
    return { inverse: [moveCol({ key, to: from })], targets: [key] };
  },

  setCell: (state, { cell: { key, config } }) => {
    const oldConfig = setConfig(state, key, config);
    if (oldConfig == null) return actions.NO_OP_RESULT;
    return {
      inverse: [setCell({ cell: { key, config: oldConfig } })],
      targets: [key],
    };
  },

  eraseCells: (state, payload) => {
    if (payload.cells.length === 0) return actions.NO_OP_RESULT;
    const selected = new Set(payload.cells);
    const rowPosOf = new Map<string, { row: number; col: number }>();
    for (let r = 0; r < state.rows.length; r++) {
      const cells = state.rows[r].cells;
      for (let c = 0; c < cells.length; c++)
        rowPosOf.set(cells[c].key, { row: r, col: c });
    }
    const rowTally = new Map<number, number>();
    const colTally = new Map<number, number>();
    for (const k of selected) {
      const pos = rowPosOf.get(k);
      if (pos == null) continue;
      rowTally.set(pos.row, (rowTally.get(pos.row) ?? 0) + 1);
      colTally.set(pos.col, (colTally.get(pos.col) ?? 0) + 1);
    }
    const fullRowIdx: number[] = [];
    for (const [rowIdx, tally] of rowTally)
      if (tally > 0 && tally === state.rows[rowIdx].cells.length)
        fullRowIdx.push(rowIdx);
    fullRowIdx.sort((a, b) => a - b);
    const fullColIdx: number[] = [];
    for (const [colIdx, tally] of colTally)
      if (tally === state.rows.length) fullColIdx.push(colIdx);
    fullColIdx.sort((a, b) => a - b);
    const inverse: Action[] = [];
    const targets: string[] = [];
    // Iterate highest-first so earlier removals don't shift later indices;
    // unshift the inverse so undo replays it in ascending order.
    for (let i = fullRowIdx.length - 1; i >= 0; i--) {
      const index = fullRowIdx[i];
      const { key, size, cells } = actions.snapshotDraft(state.rows[index]);
      state.rows.splice(index, 1);
      inverse.unshift(addRow({ index, key, size, cells }));
      targets.push(key, ...keysOf(cells));
    }
    for (let i = fullColIdx.length - 1; i >= 0; i--) {
      const index = fullColIdx[i];
      const { key, size } = state.columns[index];
      state.columns.splice(index, 1);
      const cells = removeColAt(state, index);
      inverse.unshift(addCol({ index, key, size, cells }));
      targets.push(key, ...keysOf(cells));
    }
    for (const key of selected) {
      const oldConfig = setConfig(state, key, structuredClone(payload.template));
      if (oldConfig == null) continue;
      inverse.push(setCell({ cell: { key, config: oldConfig } }));
      targets.push(key);
    }
    return { inverse, targets };
  },
  ...actions.createPathHandlers(applyPath),
};

const reduce = createReduceAll(handlers);

// The handlers put the objects of an action into the state, where a later action in
// the batch edits them in place. A copy keeps the caller's actions unchanged.
export const reduceAll = (state: Table, actions: Action[]) =>
  reduce(state, structuredClone(actions));

export interface SetChangedArgs {
  /** The key of the row that holds the cell. */
  row: string;
  /** The key of the cell. */
  key: string;
  base: CellConfig;
  next: CellConfig;
}

/**
 * Writes only the fields of a cell's config that differ between base and next, so a
 * concurrent edit to another field of the cell survives.
 * @returns the actions to dispatch, with none when nothing differs.
 */
export const setChanged = ({ row, key, base, next }: SetChangedArgs): Action[] =>
  patch
    .diff(cellConfigZ, base, next)
    .map((a) => patch.prefix(a, [ROWS_FIELD, row, CELLS_FIELD, key, CONFIG_FIELD]));

/**
 * Converts a path action from the camelCase form of the cached table into the
 * snake_case form that the Core stores. Other actions keep their form.
 */
export const encode = (state: Table, action: Action): Action => {
  switch (action.type) {
    case "set":
    case "remove":
    case "insert":
    case "move":
      return patch.camelToSnake(action, { schema: tableZ, data: state });
    default:
      return action;
  }
};

// createOf hands the dispatch controller the document carried by a create
// action so frames for never-cached documents ingest instead of drop.
export const createOf = (action: Action) =>
  action.type === "create" ? action.create.table : undefined;

const cellOf = (action: Action): string | undefined => {
  if (action.type === "set_cell") return action.setCell.cell.key;
  if (action.type !== "set" && action.type !== "remove") return undefined;
  const [root, , , cell] = patch.pathOf(action);
  return root === ROWS_FIELD ? cell : undefined;
};

// kindOf classifies an action batch for the undoable store's per-kind coalesce window.
// Continuous resize gestures (a stream of resize_row or resize_col actions) collapse
// into a single undo step; successive edits to the same cell coalesce per-cell so
// typing inside one cell collapses to one undo step but switching cells starts a fresh
// entry; everything else is treated as a discrete user action.
export const kindOf = (actions: Action[]): string => {
  if (actions.length === 0) return "default";
  if (actions.every((a) => a.type === "resize_row" || a.type === "resize_col"))
    return "resize";
  const cell = cellOf(actions[0]);
  if (cell != null && actions.every((a) => cellOf(a) === cell))
    return `set_cell:${cell}`;
  if (actions.length === 1) return actions[0].type;
  return "transaction";
};
