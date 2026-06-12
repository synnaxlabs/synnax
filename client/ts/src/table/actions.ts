// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { actions } from "@/actions";
import {
  type Action,
  addCol,
  addRow,
  createReduceAll,
  type Handlers,
  removeCol,
  removeRow,
  rename,
  resizeCol,
  resizeRow,
  setCell,
} from "@/table/actions.gen";
import { type Cell } from "@/table/types.gen";

// MIN_CELL_DIM is the floor enforced on row and column sizes.
const MIN_CELL_DIM = 32;
// BASE_ROW_DIM and BASE_COL_DIM are the defaults used when an action
// bootstraps the opposing axis on an empty table.
const BASE_ROW_DIM = 36;
const BASE_COL_DIM = 72;

// deriveCellKey returns the key for the index-th replica of template. Both
// reducers run the same scheme so optimistic client state agrees with the
// server.
const deriveCellKey = (templateKey: string, index: number): string => {
  const suffix = index.toString(16).padStart(4, "0");
  if (templateKey.length < 36) return `${templateKey}-${suffix}`;
  return templateKey.slice(0, 32) + suffix;
};

// expandTemplate returns count replicas of template with keys derived via
// deriveCellKey.
const expandTemplate = (template: Cell, count: number): Cell[] =>
  Array.from({ length: count }, (_, i) => ({
    key: deriveCellKey(template.key, i),
    config: template.config,
  }));

const handlers: Handlers = {
  rename: (state, payload) => {
    const oldName = state.name;
    state.name = payload.name;
    return {
      inverse: [rename({ name: oldName })],
      targets: [state.key],
    };
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
      for (let i = 0; i < cells.length; i++) state.columns.push({ size: BASE_COL_DIM });
    const idx = Math.min(payload.index, state.rows.length);
    const keys = cells.map((c) => c.key);
    state.rows.splice(idx, 0, {
      size: Math.max(payload.size, MIN_CELL_DIM),
      cells: keys,
    });
    for (const c of cells) state.cells[c.key] = c.config;
    return {
      inverse: [removeRow({ index: idx })],
      targets: keys,
    };
  },

  removeRow: (state, payload) => {
    if (payload.index >= state.rows.length) return actions.NO_OP_RESULT;
    const removed = actions.snapshotDraft(state.rows[payload.index]);
    const cells: Cell[] = [];
    for (const k of removed.cells) {
      const c = state.cells[k];
      if (c != null) cells.push({ key: k, config: actions.snapshotDraft(c) });
    }
    state.rows.splice(payload.index, 1);
    for (const k of removed.cells) delete state.cells[k];
    return {
      inverse: [addRow({ index: payload.index, size: removed.size, cells })],
      targets: removed.cells,
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
        state.rows.push({ size: BASE_ROW_DIM, cells: [] });
    const idx = Math.min(payload.index, state.columns.length);
    state.columns.splice(idx, 0, { size: Math.max(payload.size, MIN_CELL_DIM) });
    for (let i = 0; i < state.rows.length; i++) {
      if (i >= cells.length) break;
      const rowIdx = Math.min(idx, state.rows[i].cells.length);
      state.rows[i].cells.splice(rowIdx, 0, cells[i].key);
    }
    for (const c of cells) state.cells[c.key] = c.config;
    return {
      inverse: [removeCol({ index: idx })],
      targets: cells.map((c) => c.key),
    };
  },

  removeCol: (state, payload) => {
    if (payload.index >= state.columns.length) return actions.NO_OP_RESULT;
    const oldSize = state.columns[payload.index].size;
    const removedCells: Cell[] = [];
    state.columns.splice(payload.index, 1);
    // Rows missing a cell at this column are skipped, not padded. Go AddCol
    // short-circuits when len(cells) < len(rows), so the inverse repopulates
    // only the rows we captured.
    for (let i = 0; i < state.rows.length; i++) {
      if (payload.index >= state.rows[i].cells.length) continue;
      const k = state.rows[i].cells[payload.index];
      const c = state.cells[k];
      if (c != null) removedCells.push({ key: k, config: actions.snapshotDraft(c) });
      delete state.cells[k];
      state.rows[i].cells.splice(payload.index, 1);
    }
    return {
      inverse: [addCol({ index: payload.index, size: oldSize, cells: removedCells })],
      targets: removedCells.map((c) => c.key),
    };
  },

  resizeRow: (state, payload) => {
    if (payload.index >= state.rows.length) return actions.NO_OP_RESULT;
    const oldSize = state.rows[payload.index].size;
    state.rows[payload.index].size = Math.max(payload.size, MIN_CELL_DIM);
    return {
      inverse: [resizeRow({ index: payload.index, size: oldSize })],
      targets: [...state.rows[payload.index].cells],
    };
  },

  resizeCol: (state, payload) => {
    if (payload.index >= state.columns.length) return actions.NO_OP_RESULT;
    const oldSize = state.columns[payload.index].size;
    state.columns[payload.index].size = Math.max(payload.size, MIN_CELL_DIM);
    const targets = state.rows
      .map((r) => r.cells[payload.index])
      .filter((k): k is string => k != null);
    return {
      inverse: [resizeCol({ index: payload.index, size: oldSize })],
      targets,
    };
  },

  setCell: (state, payload) => {
    const existing = state.cells[payload.cell.key];
    if (existing == null) return actions.NO_OP_RESULT;
    const oldConfig = actions.snapshotDraft(existing);
    state.cells[payload.cell.key] = payload.cell.config;
    return {
      inverse: [setCell({ cell: { key: payload.cell.key, config: oldConfig } })],
      targets: [payload.cell.key],
    };
  },

  eraseCells: (state, payload) => {
    if (payload.cells.length === 0) return actions.NO_OP_RESULT;
    const selected = new Set(payload.cells);
    const rowPosOf = new Map<string, { row: number; col: number }>();
    for (let r = 0; r < state.rows.length; r++) {
      const cells = state.rows[r].cells;
      for (let c = 0; c < cells.length; c++) rowPosOf.set(cells[c], { row: r, col: c });
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
      const idx = fullRowIdx[i];
      const removed = actions.snapshotDraft(state.rows[idx]);
      const cells: Cell[] = [];
      for (const k of removed.cells) {
        const c = state.cells[k];
        if (c != null) cells.push({ key: k, config: actions.snapshotDraft(c) });
      }
      state.rows.splice(idx, 1);
      for (const k of removed.cells) delete state.cells[k];
      inverse.unshift(addRow({ index: idx, size: removed.size, cells }));
      targets.push(...removed.cells);
    }
    for (let i = fullColIdx.length - 1; i >= 0; i--) {
      const idx = fullColIdx[i];
      const oldSize = state.columns[idx].size;
      const removedCells: Cell[] = [];
      state.columns.splice(idx, 1);
      for (let r = 0; r < state.rows.length; r++) {
        if (idx >= state.rows[r].cells.length) continue;
        const k = state.rows[r].cells[idx];
        const c = state.cells[k];
        if (c != null) removedCells.push({ key: k, config: actions.snapshotDraft(c) });
        delete state.cells[k];
        state.rows[r].cells.splice(idx, 1);
      }
      inverse.unshift(addCol({ index: idx, size: oldSize, cells: removedCells }));
      targets.push(...removedCells.map((c) => c.key));
    }
    for (const k of selected) {
      const existing = state.cells[k];
      if (existing == null) continue;
      const oldConfig = actions.snapshotDraft(existing);
      state.cells[k] = payload.template;
      inverse.push(setCell({ cell: { key: k, config: oldConfig } }));
      targets.push(k);
    }
    return { inverse, targets };
  },
};

export const reduceAll = createReduceAll(handlers);
