// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type Triggers } from "@synnaxlabs/lyra/triggers";
import { type ReactElement, type ReactNode } from "react";

import { getCellColumn } from "@/table/Indicator";
import {
  useCellPosition,
  useColumns,
  useRedo,
  useRows,
  useUndo,
} from "@/table/queries";

/** Erases the selected cells. Registered by {@link Table}; shown on the erase item. */
export const ERASE_TRIGGER: Triggers.Trigger = ["Delete"];

export interface DefaultContextMenuProps {
  resourceKey: table.Key;
  targetID: string | null;
  selected: string[];
  editable: boolean;
  onEditableChange?: (editable: boolean) => void;
  // showIndicators reflects whether the row/column indicator strips are currently
  // visible. When onShowIndicatorsChange is also provided and editable is false, the
  // context menu surfaces a Show / Hide indicators item.
  showIndicators?: boolean;
  onShowIndicatorsChange?: (next: boolean) => void;
  /** Whether the table is centered in its container. */
  centered?: boolean;
  /** When defined, surfaces a Center / Align item in the menu. */
  onCenteredChange?: (next: boolean) => void;
  onAddRow: (index?: number) => void;
  onAddCol: (index?: number) => void;
  /** Removes the rows with the given keys. */
  onRemoveRow: (keys: string[]) => void;
  /** Removes the columns with the given keys. */
  onRemoveCol: (keys: string[]) => void;
  onEraseCells: (cells: string[]) => void;
  extra?: ReactNode;
}

const parseResizer = (id: string): { dir: "x" | "y"; key: string } | null => {
  if (!id.startsWith("resizer-")) return null;
  const [, dir, ...key] = id.split("-");
  if (dir !== "x" && dir !== "y") return null;
  return { dir, key: key.join("-") };
};

// indexOf returns the index of the row or column with key, or null when none has it.
const indexOf = (items: Array<{ key: string }>, key: string): number | null => {
  const index = items.findIndex((item) => item.key === key);
  return index === -1 ? null : index;
};

// fullySelectedRows returns the indices of the rows whose every cell is selected.
const fullySelectedRows = (
  rows: Array<{ cells: string[] }>,
  selected: Set<string>,
): number[] => {
  const out: number[] = [];
  rows.forEach((row, i) => {
    if (row.cells.length > 0 && row.cells.every((k) => selected.has(k))) out.push(i);
  });
  return out;
};

// fullySelectedCols returns the indices of the columns whose every cell is selected.
const fullySelectedCols = (
  rows: Array<{ cells: string[] }>,
  selected: Set<string>,
): number[] => {
  if (rows.length === 0) return [];
  const colCount = rows.reduce((m, r) => Math.max(m, r.cells.length), 0);
  const out: number[] = [];
  for (let x = 0; x < colCount; x++) {
    let all = true;
    for (const row of rows) {
      const k = row.cells[x];
      if (k == null || !selected.has(k)) {
        all = false;
        break;
      }
    }
    if (all) out.push(x);
  }
  return out;
};

// targetIndices returns the indices a delete acts on: every fully selected index when
// the clicked index is one of them, and the clicked index alone otherwise.
const targetIndices = (index: number | null, fullySelected: number[]): number[] => {
  if (index == null) return [];
  return fullySelected.includes(index) ? fullySelected : [index];
};

export const DefaultContextMenu = ({
  resourceKey,
  targetID,
  selected,
  editable,
  onEditableChange,
  showIndicators = true,
  onShowIndicatorsChange,
  centered = false,
  onCenteredChange,
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
  const rows = useRows({ key: resourceKey });
  const columns = useColumns({ key: resourceKey });
  const rowIdx =
    resizer?.dir === "y" ? indexOf(rows, resizer.key) : (cellPos?.y ?? null);
  const colIdx =
    resizer?.dir === "x" ? indexOf(columns, resizer.key) : (cellPos?.x ?? null);
  const selectedSet = new Set(selected);
  const targetRowIndices = targetIndices(rowIdx, fullySelectedRows(rows, selectedSet));
  const targetColIndices = targetIndices(colIdx, fullySelectedCols(rows, selectedSet));
  const handleToggleEditable = () => onEditableChange?.(!editable);
  const handleToggleIndicators = () => onShowIndicatorsChange?.(!showIndicators);
  const handleToggleCentered = () => onCenteredChange?.(!centered);
  const showIndicatorToggle = !editable && onShowIndicatorsChange != null;
  const { undo, canUndo } = useUndo({ key: resourceKey });
  const { redo, canRedo } = useRedo({ key: resourceKey });
  return (
    <Menu.Menu level="small" gap="small">
      {editable && (
        <>
          <Menu.UndoRedoItems
            undo={undo}
            redo={redo}
            canUndo={canUndo}
            canRedo={canRedo}
          />
          <Menu.Divider />
        </>
      )}
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
            onClick={() => onRemoveRow(targetRowIndices.map((i) => rows[i].key))}
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
          onClick={() => onRemoveCol(targetColIndices.map((i) => columns[i].key))}
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
          triggerIndicator={ERASE_TRIGGER}
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
      {onCenteredChange != null && (
        <Menu.Item size="small" itemKey="toggleCentered" onClick={handleToggleCentered}>
          {centered ? <Icon.Align.BoxTopLeft /> : <Icon.Align.BoxCenter />}
          {centered ? "Align table to top left" : "Center table"}
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
