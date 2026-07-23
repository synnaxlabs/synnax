// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { type record, uuid } from "@synnaxlabs/x";
import { type ClipboardEventHandler, useCallback } from "react";

import { Flux } from "@/flux";
import { useSyncedRef } from "@/hooks";
import { Cell } from "@/table/cells";
import { findCellPosition, type FluxSubStore, useDispatch } from "@/table/queries";
import { Theming } from "@/theming";

// The "web " prefix is required: Chrome silently drops custom MIME types from
// the clipboard without it.
const MIME = "web application/synnax-table+json";
const VERSION = 1;
const BASE_ROW_SIZE = 36;
const BASE_COL_SIZE = 72;

interface CellPayload {
  row: number;
  col: number;
  variant: string;
  props: record.Unknown;
}

interface Payload {
  version: number;
  cells: CellPayload[];
}

const describe = (p: Payload): string =>
  `${p.cells.length} cell${p.cells.length === 1 ? "" : "s"}`;

export interface UseClipboardParams {
  key: table.Key;
  selected?: string[];
  // onPaste fires after a successful paste with the keys of every cell whose
  // contents were overwritten. Consumers can use this to update the visible
  // selection to the just-pasted region. Paste uses the last entry in
  // `selected` as the top-left anchor; if `selected` is empty when paste
  // fires, paste is a no-op.
  onPaste?: (overwrittenKeys: string[]) => void;
}

export interface UseClipboardReturn {
  onCopy: ClipboardEventHandler;
  onPaste: ClipboardEventHandler;
}

export const useClipboard = ({
  key,
  selected,
  onPaste,
}: UseClipboardParams): UseClipboardReturn => {
  const { dispatch } = useDispatch();
  const store = Flux.useStore<FluxSubStore>();
  const selectedRef = useSyncedRef(selected ?? []);
  const theme = Theming.use();

  const handleCopy = useCallback<ClipboardEventHandler>(
    (e) => {
      // Defer to the browser if the user has a real text selection.
      const text = window.getSelection()?.toString();
      if (text != null && text.length > 0) return;
      const t = store.tables.get(key);
      if (t == null) return;
      const sel = selectedRef.current;
      if (sel.length === 0) return;
      const cells: CellPayload[] = [];
      let minRow = Number.POSITIVE_INFINITY;
      let minCol = Number.POSITIVE_INFINITY;
      const intermediate: Array<{ row: number; col: number; cell: table.Cell }> = [];
      for (const cellKey of sel) {
        const p = findCellPosition(t.rows, cellKey);
        const c = t.cells[cellKey];
        if (p == null || c == null) continue;
        if (p.y < minRow) minRow = p.y;
        if (p.x < minCol) minCol = p.x;
        intermediate.push({ row: p.y, col: p.x, cell: c });
      }
      if (intermediate.length === 0) return;
      for (const { row, col, cell } of intermediate)
        cells.push({
          row: row - minRow,
          col: col - minCol,
          variant: cell.variant,
          props: cell.props,
        });
      const payload: Payload = { version: VERSION, cells };
      e.preventDefault();
      e.clipboardData.setData(MIME, JSON.stringify(payload));
      e.clipboardData.setData("text/plain", describe(payload));
    },
    [key, store],
  );

  const handlePaste = useCallback<ClipboardEventHandler>(
    (e) => {
      const raw = e.clipboardData.getData(MIME);
      if (raw === "") return;
      let payload: Payload;
      try {
        payload = JSON.parse(raw) as Payload;
      } catch {
        return;
      }
      if (payload.version !== VERSION || payload.cells.length === 0) return;
      const sel = selectedRef.current;
      const anchorKey = sel[sel.length - 1] ?? null;
      if (anchorKey == null) return;
      const t = store.tables.get(key);
      if (t == null) return;
      const anchorPos = findCellPosition(t.rows, anchorKey);
      if (anchorPos == null) return;
      e.preventDefault();

      const existingRows = t.rows.length;
      const existingCols = t.columns.length;
      let maxRow = existingRows - 1;
      let maxCol = existingCols - 1;
      for (const c of payload.cells) {
        const r = anchorPos.y + c.row;
        const col = anchorPos.x + c.col;
        if (r > maxRow) maxRow = r;
        if (col > maxCol) maxCol = col;
      }
      const targetRows = maxRow + 1;
      const targetCols = maxCol + 1;

      const actions: table.Action[] = [];
      const defaultProps = Cell.REGISTRY.text.defaultProps(theme);
      // keyAt maps a final-state grid position to the cell key at that
      // position. Existing positions use the current store keys; newly-added
      // positions get fresh UUIDs that are baked into the addCol/addRow
      // actions below so the post-batch state is fully resolved.
      const keyAt: Record<string, string> = {};
      const posKey = (r: number, c: number): string => `${r},${c}`;
      for (let r = 0; r < existingRows; r++)
        for (let c = 0; c < existingCols; c++) {
          const k = t.rows[r]?.cells[c];
          if (k != null) keyAt[posKey(r, c)] = k;
        }

      const newColCount = Math.max(0, targetCols - existingCols);
      for (let i = 0; i < newColCount; i++) {
        const colIdx = existingCols + i;
        const cellsForExistingRows: table.Cell[] = [];
        for (let r = 0; r < existingRows; r++) {
          const newKey = uuid.create();
          keyAt[posKey(r, colIdx)] = newKey;
          cellsForExistingRows.push({
            key: newKey,
            variant: "text",
            props: defaultProps,
          });
        }
        actions.push(
          table.addCol({
            index: colIdx,
            size: BASE_COL_SIZE,
            cells: cellsForExistingRows,
          }),
        );
      }

      const newRowCount = Math.max(0, targetRows - existingRows);
      for (let i = 0; i < newRowCount; i++) {
        const rowIdx = existingRows + i;
        const cellsForAllCols: table.Cell[] = [];
        for (let c = 0; c < targetCols; c++) {
          const newKey = uuid.create();
          keyAt[posKey(rowIdx, c)] = newKey;
          cellsForAllCols.push({
            key: newKey,
            variant: "text",
            props: defaultProps,
          });
        }
        actions.push(
          table.addRow({
            index: rowIdx,
            size: BASE_ROW_SIZE,
            cells: cellsForAllCols,
          }),
        );
      }

      const overwritten: string[] = [];
      for (const c of payload.cells) {
        const r = anchorPos.y + c.row;
        const col = anchorPos.x + c.col;
        const k = keyAt[posKey(r, col)];
        if (k == null) continue;
        actions.push(
          table.setCell({
            cell: { key: k, variant: c.variant, props: c.props },
          }),
        );
        overwritten.push(k);
      }
      dispatch({ key, actions });
      if (actions.length > 0) onPaste?.(overwritten);
    },
    [key, store, theme, dispatch, onPaste],
  );

  return { onCopy: handleCopy, onPaste: handlePaste };
};
