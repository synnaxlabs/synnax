// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table } from "@synnaxlabs/client";
import { type ReactElement, type ReactNode, useCallback } from "react";

import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { useCellPosition } from "@/table/queries";

export interface DefaultContextMenuProps {
  resourceKey: table.Key;
  targetID: string | null;
  editable: boolean;
  onEditableChange?: (editable: boolean) => void;
  onAddRow: (index?: number) => void;
  onAddCol: (index?: number) => void;
  onRemoveRow: (index: number) => void;
  onRemoveCol: (index: number) => void;
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
  editable,
  onEditableChange,
  onAddRow,
  onAddCol,
  onRemoveRow,
  onRemoveCol,
  extra,
}: DefaultContextMenuProps): ReactElement => {
  const resizer = targetID != null ? parseResizer(targetID) : null;
  const cellKey = resizer == null ? targetID : null;
  const cellPos = useCellPosition({ key: resourceKey, cellKey: cellKey ?? "" });
  const rowIdx = resizer?.dir === "y" ? resizer.index : (cellPos?.y ?? null);
  const colIdx = resizer?.dir === "x" ? resizer.index : (cellPos?.x ?? null);
  const handleToggleEditable = useCallback(
    () => onEditableChange?.(!editable),
    [onEditableChange, editable],
  );
  return (
    <Menu.Menu level="small" gap="small">
      {editable && rowIdx != null && (
        <>
          <Menu.Item
            size="small"
            itemKey="addRowBelow"
            onClick={() => onAddRow(rowIdx + 1)}
          >
            <Icon.Add />
            Add row below
          </Menu.Item>
          <Menu.Item
            size="small"
            itemKey="addRowAbove"
            onClick={() => onAddRow(rowIdx)}
          >
            <Icon.Add />
            Add row above
          </Menu.Item>
        </>
      )}
      {editable && colIdx != null && (
        <>
          <Menu.Divider />
          <Menu.Item
            size="small"
            itemKey="addColRight"
            onClick={() => onAddCol(colIdx + 1)}
          >
            <Icon.Add />
            Add column right
          </Menu.Item>
          <Menu.Item size="small" itemKey="addColLeft" onClick={() => onAddCol(colIdx)}>
            <Icon.Add />
            Add column left
          </Menu.Item>
        </>
      )}
      {editable && rowIdx != null && (
        <>
          <Menu.Divider />
          <Menu.Item
            size="small"
            itemKey="deleteRow"
            onClick={() => onRemoveRow(rowIdx)}
          >
            <Icon.Delete />
            Delete row
          </Menu.Item>
        </>
      )}
      {editable && colIdx != null && (
        <Menu.Item size="small" itemKey="deleteCol" onClick={() => onRemoveCol(colIdx)}>
          <Icon.Delete />
          Delete column
        </Menu.Item>
      )}
      {editable && (rowIdx != null || colIdx != null) && <Menu.Divider />}
      {onEditableChange != null && (
        <Menu.Item size="small" itemKey="toggleEdit" onClick={handleToggleEditable}>
          {editable ? <Icon.EditOff /> : <Icon.Edit />}
          {`${editable ? "Disable" : "Enable"} editing`}
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
