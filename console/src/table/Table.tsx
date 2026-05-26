// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/Table.css";

import { table } from "@synnaxlabs/client";
import { Access, Button, Icon, Menu, Table as Base, Triggers } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { ContextMenu, Controls } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { useSelectEditable, useSelectSelectedCellKeys } from "@/table/selectors";
import { setEditable, setSelectedCells } from "@/table/slice";
import { useAutoUpload } from "@/table/useUpload";

export { create, LAYOUT_TYPE, type LayoutType } from "@/table/layout";

type ShortcutMode = "clear" | "undo" | "redo" | "default";

const SHORTCUT_CONFIG: Triggers.ModeConfig<ShortcutMode> = {
  clear: [["Delete"], ["Backspace"]],
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};

const SHORTCUT_TRIGGERS = Triggers.flattenConfig(SHORTCUT_CONFIG);

const Loaded: Layout.Renderer = ({ layoutKey, visible }) => {
  const editable = useSelectEditable(layoutKey);
  const selected = useSelectSelectedCellKeys(layoutKey);
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(layoutKey));
  const canEdit = hasUpdatePermission && editable;
  const dispatch = useDispatch();

  const addRow = Base.useAddRow({ key: layoutKey });
  const addCol = Base.useAddCol({ key: layoutKey });
  const removeRow = Base.useRemoveRow({ key: layoutKey });
  const removeCol = Base.useRemoveCol({ key: layoutKey });
  const clearSelected = Base.useClearSelected({ key: layoutKey });
  const { undo } = Base.useUndo({ key: layoutKey });
  const { redo } = Base.useRedo({ key: layoutKey });
  const tableRef = useRef<HTMLDivElement>(null);

  const handlePasted = useCallback(
    (overwrittenKeys: string[]) => {
      if (overwrittenKeys.length === 0) return;
      dispatch(
        setSelectedCells({
          key: layoutKey,
          cells: overwrittenKeys,
          anchor: overwrittenKeys[0],
        }),
      );
    },
    [dispatch, layoutKey],
  );
  const { onCopy, onPaste } = Base.useClipboard({
    key: layoutKey,
    selected,
    onPaste: handlePasted,
  });

  Triggers.use({
    triggers: SHORTCUT_TRIGGERS,
    region: tableRef,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start" || !canEdit) return;
        const mode = Triggers.determineMode(SHORTCUT_CONFIG, triggers);
        if (mode === "clear") {
          if (selected.length === 0) return;
          clearSelected(selected);
        } else if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [canEdit, selected, clearSelected, undo, redo],
    ),
  });

  const handleSelectionChange = useCallback(
    (cells: string[]) =>
      dispatch(
        setSelectedCells({ key: layoutKey, cells, anchor: cells.at(-1) ?? null }),
      ),
    [dispatch, layoutKey],
  );

  const [menuTarget, setMenuTarget] = useState<Base.ContextMenuTarget | null>(null);
  const menuProps = Menu.useContextMenu();

  const handleTableContextMenu = useCallback(
    (e: React.MouseEvent, target: Base.ContextMenuTarget) => {
      setMenuTarget(target);
      menuProps.open(e);
    },
    [menuProps],
  );

  const handleAddRowAt = useCallback((atIndex: number) => addRow(atIndex), [addRow]);
  const handleAddColAt = useCallback((atIndex: number) => addCol(atIndex), [addCol]);

  // The context menu uses cellKey to derive the row/col index of the
  // right-clicked cell. For that derivation we need rows from flux; defer
  // to a child that calls the selector so the context menu only mounts
  // when open.
  const contextMenu = () =>
    menuTarget == null ? null : (
      <TableContextMenu
        layoutKey={layoutKey}
        target={menuTarget}
        canEdit={canEdit}
        hasUpdatePermission={hasUpdatePermission}
        onAddRowAt={handleAddRowAt}
        onAddColAt={handleAddColAt}
        onRemoveRow={removeRow}
        onRemoveCol={removeCol}
      />
    );

  return (
    <div ref={tableRef} className={CSS.B("table")} tabIndex={0}>
      <Menu.ContextMenu menu={contextMenu} {...menuProps}>
        <Base.Table
          resourceKey={layoutKey}
          selected={selected}
          onSelectionChange={handleSelectionChange}
          editable={canEdit}
          visible={visible}
          onContextMenu={handleTableContextMenu}
          onCopy={onCopy}
          onPaste={onPaste}
          className={menuProps.className}
        />
        {canEdit && (
          <>
            <Button.Button
              className={CSS.BE("table", "add-col")}
              justify="center"
              align="center"
              size="tiny"
              variant="filled"
              onClick={() => addCol()}
            >
              <Icon.Add />
            </Button.Button>
            <Button.Button
              className={CSS.BE("table", "add-row")}
              justify="center"
              variant="filled"
              align="center"
              size="tiny"
              onClick={() => addRow()}
            >
              <Icon.Add />
            </Button.Button>
          </>
        )}
        <TableControls tableKey={layoutKey} />
      </Menu.ContextMenu>
    </div>
  );
};

interface TableContextMenuProps {
  layoutKey: string;
  target: Base.ContextMenuTarget;
  canEdit: boolean;
  hasUpdatePermission: boolean;
  onAddRowAt: (atIndex: number) => void;
  onAddColAt: (atIndex: number) => void;
  onRemoveRow: (atIndex: number) => void;
  onRemoveCol: (atIndex: number) => void;
}

const TableContextMenu = ({
  layoutKey,
  target,
  canEdit,
  hasUpdatePermission,
  onAddRowAt,
  onAddColAt,
  onRemoveRow,
  onRemoveCol,
}: TableContextMenuProps): ReactElement => {
  const cellPos = Base.useCellPosition({
    key: layoutKey,
    cellKey: target.cellKey ?? "",
  });
  const rowIdx = target.rowResizerIndex ?? cellPos?.y ?? null;
  const colIdx = target.colResizerIndex ?? cellPos?.x ?? null;
  return (
    <ContextMenu.Menu>
      {canEdit && rowIdx != null && (
        <>
          <Menu.Item
            size="small"
            itemKey="addRowBelow"
            onClick={() => onAddRowAt(rowIdx + 1)}
          >
            <Icon.Add />
            Add row below
          </Menu.Item>
          <Menu.Item
            size="small"
            itemKey="addRowAbove"
            onClick={() => onAddRowAt(rowIdx)}
          >
            <Icon.Add />
            Add row above
          </Menu.Item>
        </>
      )}
      {canEdit && colIdx != null && (
        <>
          <Menu.Divider />
          <Menu.Item
            size="small"
            itemKey="addColRight"
            onClick={() => onAddColAt(colIdx + 1)}
          >
            <Icon.Add />
            Add column right
          </Menu.Item>
          <Menu.Item
            size="small"
            itemKey="addColLeft"
            onClick={() => onAddColAt(colIdx)}
          >
            <Icon.Add />
            Add column left
          </Menu.Item>
        </>
      )}
      {canEdit && rowIdx != null && (
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
      {canEdit && colIdx != null && (
        <Menu.Item size="small" itemKey="deleteCol" onClick={() => onRemoveCol(colIdx)}>
          <Icon.Delete />
          Delete column
        </Menu.Item>
      )}
      {canEdit && (rowIdx != null || colIdx != null) && <Menu.Divider />}
      {hasUpdatePermission && <EditToggleMenuItem layoutKey={layoutKey} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

interface EditToggleMenuItemProps {
  layoutKey: string;
}

const EditToggleMenuItem = ({ layoutKey }: EditToggleMenuItemProps): ReactElement => {
  const editable = useSelectEditable(layoutKey);
  const dispatch = useDispatch();
  const handleClick = useCallback(
    () => dispatch(setEditable({ key: layoutKey })),
    [dispatch, layoutKey],
  );
  return (
    <Menu.Item itemKey="toggleEdit" onClick={handleClick}>
      {editable ? <Icon.EditOff /> : <Icon.Edit />}
      {`${editable ? "Disable" : "Enable"} editing`}
    </Menu.Item>
  );
};

interface TableControlsProps {
  tableKey: string;
}

const TableControls = ({ tableKey }: TableControlsProps): ReactElement | null => {
  const editable = useSelectEditable(tableKey);
  const hasUpdatePermission = Access.useUpdateGranted(table.ontologyID(tableKey));
  const dispatch = useDispatch();
  const handleEdit = useCallback(
    () => dispatch(setEditable({ key: tableKey })),
    [dispatch, tableKey],
  );
  if (!hasUpdatePermission) return null;
  const canEdit = hasUpdatePermission && editable;
  return (
    <Controls>
      <Button.Toggle
        value={canEdit}
        onChange={handleEdit}
        size="small"
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={`${canEdit ? "Disable" : "Enable"} editing`}
      >
        {canEdit ? <Icon.EditOff /> : <Icon.Edit />}
      </Button.Toggle>
    </Controls>
  );
};

export const Table: Layout.Renderer = (props) => {
  const uploaded = useAutoUpload(props.layoutKey);
  if (!uploaded) return null;
  return <Loaded {...props} />;
};

Table.useName = Layout.createUseFluxName(
  Base.useRename,
  Base.useRetrieveObservableName,
);
