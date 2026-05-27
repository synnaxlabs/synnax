// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { current, type Draft, isDraft } from "immer";

import {
  addCol,
  addRow,
  createReduceAll,
  type HandlerResult,
  type Handlers,
  removeCol,
  removeRow,
  rename,
  resizeCol,
  resizeRow,
  setCell,
} from "@/table/actions.gen";
import { type Cell } from "@/table/types.gen";

const NO_OP: HandlerResult = { inverse: [], targets: [] };

// snapshot pulls a value out of an Immer draft so the result is safe to embed
// in an action stored on the undo stack. When reduceAll applies multiple
// actions inside one produce(), an earlier wholesale assignment can leave a
// slot as a plain object; a later action that calls current() unconditionally
// would crash.
const snapshot = <T>(v: T): T => (isDraft(v) ? current(v as Draft<T>) : v);

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
    const idx = Math.min(payload.index, state.rows.length);
    const keys = payload.cells.map((c) => c.key);
    state.rows.splice(idx, 0, { size: payload.size, cells: keys });
    for (const c of payload.cells) state.cells[c.key] = c;
    return {
      inverse: [removeRow({ index: idx })],
      targets: keys,
    };
  },

  removeRow: (state, payload) => {
    if (payload.index >= state.rows.length) return NO_OP;
    const removed = snapshot(state.rows[payload.index]);
    const cells: Cell[] = [];
    for (const k of removed.cells) {
      const c = state.cells[k];
      if (c != null) cells.push(snapshot(c));
    }
    state.rows.splice(payload.index, 1);
    for (const k of removed.cells) delete state.cells[k];
    return {
      inverse: [addRow({ index: payload.index, size: removed.size, cells })],
      targets: removed.cells,
    };
  },

  addCol: (state, payload) => {
    const idx = Math.min(payload.index, state.columns.length);
    state.columns.splice(idx, 0, { size: payload.size });
    for (let i = 0; i < state.rows.length; i++) {
      if (i >= payload.cells.length) break;
      const rowIdx = Math.min(idx, state.rows[i].cells.length);
      state.rows[i].cells.splice(rowIdx, 0, payload.cells[i].key);
    }
    for (const c of payload.cells) state.cells[c.key] = c;
    return {
      inverse: [removeCol({ index: idx })],
      targets: payload.cells.map((c) => c.key),
    };
  },

  removeCol: (state, payload) => {
    if (payload.index >= state.columns.length) return NO_OP;
    const oldSize = state.columns[payload.index].size;
    const removedCells: Cell[] = [];
    state.columns.splice(payload.index, 1);
    // Rows are kept aligned with columns by construction. A row that doesn't
    // have a cell at the removed index is a corrupted state; treat it as a
    // gap and don't add a placeholder to the inverse's cells array. The
    // generated Go AddCol handler short-circuits when len(cells) < len(rows),
    // so the inverse will repopulate as many rows as we captured.
    for (let i = 0; i < state.rows.length; i++) {
      if (payload.index >= state.rows[i].cells.length) continue;
      const k = state.rows[i].cells[payload.index];
      const c = state.cells[k];
      if (c != null) removedCells.push(snapshot(c));
      delete state.cells[k];
      state.rows[i].cells.splice(payload.index, 1);
    }
    return {
      inverse: [addCol({ index: payload.index, size: oldSize, cells: removedCells })],
      targets: removedCells.map((c) => c.key),
    };
  },

  resizeRow: (state, payload) => {
    if (payload.index >= state.rows.length) return NO_OP;
    const oldSize = state.rows[payload.index].size;
    state.rows[payload.index].size = payload.size;
    return {
      inverse: [resizeRow({ index: payload.index, size: oldSize })],
      targets: [...state.rows[payload.index].cells],
    };
  },

  resizeCol: (state, payload) => {
    if (payload.index >= state.columns.length) return NO_OP;
    const oldSize = state.columns[payload.index].size;
    state.columns[payload.index].size = payload.size;
    const targets: string[] = [];
    for (const r of state.rows) {
      const k = r.cells[payload.index];
      if (k != null) targets.push(k);
    }
    return {
      inverse: [resizeCol({ index: payload.index, size: oldSize })],
      targets,
    };
  },

  setCell: (state, payload) => {
    const existing = state.cells[payload.cell.key];
    if (existing == null) return NO_OP;
    const oldCell = snapshot(existing);
    state.cells[payload.cell.key] = payload.cell;
    return {
      inverse: [setCell({ cell: oldCell })],
      targets: [payload.cell.key],
    };
  },
};

export const reduceAll = createReduceAll(handlers);
