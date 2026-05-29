// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table } from "@synnaxlabs/client";
import { type ReactElement, type ReactNode, useCallback, useMemo } from "react";

import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { getCellColumn } from "@/table/Indicator";
import { useCellPosition, useSelectRows } from "@/table/queries";

export interface DefaultContextMenuProps {
  resourceKey: table.Key;
  targetID: string | null;
  selected: string[];
  editable: boolean;
  onEditableChange?: (editable: boolean) => void;
  // showIndicators reflects whether the row/column indicator strips are
  // currently visible. When onShowIndicatorsChange is also provided and
  // editable is false, the context menu surfaces a Show / Hide indicators
  // item.
  showIndicators?: boolean;
  onShowIndicatorsChange?: (next: boolean) => void;
  onAddRow: (index?: number) => void;
  onAddCol: (index?: number) => void;
  onRemoveRow: (indices: number[]) => void;
  onRemoveCol: (indices: number[]) => void;
  onEraseCells: (cells: string[]) => void;
  extra?: ReactNode;
}

const parseResizer = (id: string): { dir: "x" | "y"; index: number } | null => {
  if (!id.startsWith("resizer-")) return null;
  const [, dir, idx] = id.split("-");
  if (dir !== "x" && dir !== "y") return null;
  const parsed = Number.parseInt(idx, 10);
  if (!Number.isFinite(parsed)) return null;
  return { dir, index: parsed };
};

export const DefaultContextMenu = ({
  resourceKey,
  targetID,
  selected,
  editable,
  onEditableChange,
  showIndicators = true,
  onShowIndicatorsChange,
  onAddRow,
  onAddCol,
  onRemoveRow,
  onRemoveCol,
  onEraseCells,
  extra,
}: DefaultContextMenuProps): ReactElement => {
  const resizer = targetID != null ? parseResizer(targetID) : null;
  const cellKey = resizer == null ? targetID : null;
  const cellPos = useCellPosition({ key: resourceKey, cellKey: cellKey ?? "" });
  const rowIdx = resizer?.dir === "y" ? resizer.index : (cellPos?.y ?? null);
  const colIdx = resizer?.dir === "x" ? resizer.index : (cellPos?.x ?? null);
  const rows = useSelectRows({ key: resourceKey });
  const selectedSet = useMemo(() => new Set(selected), [selected]);
  // fullySelectedRows / fullySelectedCols: indices where every cell along
  // that axis is in the selection. When the right-clicked row/col is part
  // of this set, the delete action operates on the whole set; otherwise
  // just the clicked one.
  const fullySelectedRows = useMemo(() => {
    const out: number[] = [];
    rows.forEach((row, i) => {
      if (row.cells.length > 0 && row.cells.every((k) => selectedSet.has(k)))
        out.push(i);
    });
    return out;
  }, [rows, selectedSet]);
  const fullySelectedCols = useMemo(() => {
    if (rows.length === 0) return [];
    const colCount = rows.reduce((m, r) => Math.max(m, r.cells.length), 0);
    const out: number[] = [];
    for (let x = 0; x < colCount; x++) {
      let all = true;
      for (const row of rows) {
        const k = row.cells[x];
        if (k == null || !selectedSet.has(k)) {
          all = false;
          break;
        }
      }
      if (all) out.push(x);
    }
    return out;
  }, [rows, selectedSet]);
  const targetRowIndices = useMemo(() => {
    if (rowIdx == null) return [] as number[];
    return fullySelectedRows.includes(rowIdx) ? fullySelectedRows : [rowIdx];
  }, [rowIdx, fullySelectedRows]);
  const targetColIndices = useMemo(() => {
    if (colIdx == null) return [] as number[];
    return fullySelectedCols.includes(colIdx) ? fullySelectedCols : [colIdx];
  }, [colIdx, fullySelectedCols]);
  const handleToggleEditable = useCallback(
    () => onEditableChange?.(!editable),
    [onEditableChange, editable],
  );
  const handleToggleIndicators = useCallback(
    () => onShowIndicatorsChange?.(!showIndicators),
    [onShowIndicatorsChange, showIndicators],
  );
  const showIndicatorToggle = !editable && onShowIndicatorsChange != null;
  return (
    <Menu.Menu level="small" gap="small">
      {editable && rowIdx != null && (
        <>
          <Menu.Item
            size="small"
            itemKey="addRowAbove"
            onClick={() => onAddRow(rowIdx)}
          >
            <Icon.Insert.Row.Above />
            Add row above
          </Menu.Item>
          <Menu.Item
            size="small"
            itemKey="addRowBelow"
            onClick={() => onAddRow(rowIdx + 1)}
          >
            <Icon.Insert.Row.Below />
            Add row below
          </Menu.Item>
        </>
      )}
      {editable && colIdx != null && (
        <>
          <Menu.Divider />
          <Menu.Item size="small" itemKey="addColLeft" onClick={() => onAddCol(colIdx)}>
            <Icon.Insert.Col.Left />
            Add column left
          </Menu.Item>
          <Menu.Item
            size="small"
            itemKey="addColRight"
            onClick={() => onAddCol(colIdx + 1)}
          >
            <Icon.Insert.Col.Right />
            Add column right
          </Menu.Item>
        </>
      )}
      {editable && rowIdx != null && (
        <>
          <Menu.Divider />
          <Menu.Item
            size="small"
            itemKey="deleteRow"
            onClick={() => onRemoveRow(targetRowIndices)}
          >
            <Icon.Delete.Row />
            {targetRowIndices.length > 1
              ? `Delete ${targetRowIndices.length} rows`
              : `Delete row ${rowIdx + 1}`}
          </Menu.Item>
        </>
      )}
      {editable && colIdx != null && (
        <Menu.Item
          size="small"
          itemKey="deleteCol"
          onClick={() => onRemoveCol(targetColIndices)}
        >
          <Icon.Delete.Col />
          {targetColIndices.length > 1
            ? `Delete ${targetColIndices.length} columns`
            : `Delete column ${getCellColumn(colIdx)}`}
        </Menu.Item>
      )}
      {editable && selected.length > 0 && (
        <Menu.Item
          size="small"
          itemKey="eraseCells"
          onClick={() => onEraseCells(selected)}
        >
          <Icon.Eraser />
          {selected.length > 1 ? "Erase cells" : "Erase cell"}
        </Menu.Item>
      )}
      {editable && (rowIdx != null || colIdx != null || selected.length > 0) && (
        <Menu.Divider />
      )}
      {onEditableChange != null && (
        <Menu.Item size="small" itemKey="toggleEdit" onClick={handleToggleEditable}>
          {editable ? <Icon.EditOff /> : <Icon.Edit />}
          {`${editable ? "Disable" : "Enable"} editing`}
        </Menu.Item>
      )}
      {showIndicatorToggle && (
        <Menu.Item
          size="small"
          itemKey="toggleIndicators"
          onClick={handleToggleIndicators}
        >
          {showIndicators ? <Icon.Hidden /> : <Icon.Visible />}
          {`${showIndicators ? "Hide" : "Show"} indicators`}
        </Menu.Item>
      )}
      {extra != null && (
        <>
          <Menu.Divider />
          {extra}
        </>
      )}
    </Menu.Menu>
  );
};
