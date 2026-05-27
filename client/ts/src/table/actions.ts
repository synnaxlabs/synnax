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

// MIN_CELL_DIM is the floor enforced on row and column sizes.
const MIN_CELL_DIM = 32;
// BASE_ROW_DIM / BASE_COL_DIM are the defaults used when AddRow / AddCol
// fire against an empty table and need to bootstrap the missing axis.
const BASE_ROW_DIM = 36;
const BASE_COL_DIM = 72;

// snapshot pulls a value out of an Immer draft so the result is safe to embed
// in an action stored on the undo stack. When reduceAll applies multiple
// actions inside one produce(), an earlier wholesale assignment can leave a
// slot as a plain object; a later action that calls current() unconditionally
// would crash.
const snapshot = <T>(v: T): T => (isDraft(v) ? current(v as Draft<T>) : v);

// deriveCellKey derives a unique-per-index cell key from a template key by
// replacing the last four hex digits with the index encoded in hex. The
// template key is assumed to be a 36-character UUID; positions 14 (version)
// and 19 (variant) sit in the prefix so the derived key keeps the UUID v4
// layout. Both Go and TS reducers run the same scheme so the optimistic
// flux store agrees with the server.
const deriveCellKey = (templateKey: string, index: number): string => {
  const suffix = index.toString(16).padStart(4, "0");
  if (templateKey.length < 36) return `${templateKey}-${suffix}`;
  return templateKey.slice(0, 32) + suffix;
};

// expandTemplate returns a slice of replica cells derived from a template,
// one per axis position. The first replica gets key deriveCellKey(template,
// 0), the second deriveCellKey(template, 1), and so on.
const expandTemplate = (template: Cell, count: number): Cell[] =>
  Array.from({ length: count }, (_, i) => ({
    key: deriveCellKey(template.key, i),
    variant: template.variant,
    props: template.props,
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
      ? expandTemplate(payload.cellTemplate, Math.max(state.columns.length, 1))
      : payload.cells;
    if (state.columns.length === 0 && cells.length > 0)
      for (let i = 0; i < cells.length; i++) state.columns.push({ size: BASE_COL_DIM });
    const idx = Math.min(payload.index, state.rows.length);
    const keys = cells.map((c) => c.key);
    state.rows.splice(idx, 0, {
      size: Math.max(payload.size, MIN_CELL_DIM),
      cells: keys,
    });
    for (const c of cells) state.cells[c.key] = c;
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
    const cells = payload.cellTemplate
      ? expandTemplate(payload.cellTemplate, Math.max(state.rows.length, 1))
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
    for (const c of cells) state.cells[c.key] = c;
    return {
      inverse: [removeCol({ index: idx })],
      targets: cells.map((c) => c.key),
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
    state.rows[payload.index].size = Math.max(payload.size, MIN_CELL_DIM);
    return {
      inverse: [resizeRow({ index: payload.index, size: oldSize })],
      targets: [...state.rows[payload.index].cells],
    };
  },

  resizeCol: (state, payload) => {
    if (payload.index >= state.columns.length) return NO_OP;
    const oldSize = state.columns[payload.index].size;
    state.columns[payload.index].size = Math.max(payload.size, MIN_CELL_DIM);
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
