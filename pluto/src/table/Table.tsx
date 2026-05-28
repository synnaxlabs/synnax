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
import { box, id, math } from "@synnaxlabs/x";
import {
  type ComponentPropsWithRef,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { table as aetherTable } from "@/table/aether";
import { CELLS } from "@/table/cells/registry";
import { useClipboard } from "@/table/clipboard";
import { DefaultContextMenu } from "@/table/ContextMenu";
import { ColumnIndicators } from "@/table/Indicator";
import {
  cellsInRegion,
  findCellPosition,
  useDispatch,
  useEnsureRetrieved,
  useRedo,
  useSelectColumns,
  useSelectRows,
  useUndo,
} from "@/table/queries";
import { Row } from "@/table/Row";
import { Theming } from "@/theming";
import { Triggers } from "@/triggers";
import { Canvas } from "@/vis/canvas";

export { getCellColumn } from "@/table/Indicator";

type TriggerMode = "clear" | "undo" | "redo" | "default";

const TRIGGERS_CONFIG: Triggers.ModeConfig<TriggerMode> = {
  clear: [["Delete"], ["Backspace"]],
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};

const FLATTENED_TRIGGERS_CONFIG = Triggers.flattenConfig(TRIGGERS_CONFIG);

const BASE_ROW_SIZE = 36;
const BASE_COL_SIZE = 72;

const newDefaultCell = (theme: ReturnType<typeof Theming.use>): table.Cell => ({
  key: id.create(),
  variant: "text",
  props: CELLS.text.defaultProps(theme),
});

export interface TableProps
  extends
    Omit<ComponentPropsWithRef<"div">, "onCopy" | "onPaste" | "onContextMenu">,
    Pick<z.infer<typeof aetherTable.Table.stateZ>, "visible"> {
  // resourceKey is the table key the component reads from the Pluto flux
  // store. The table data must be loaded into flux before the component
  // mounts; useEnsureRetrieved kicks off the fetch.
  resourceKey: table.Key;
  // selected is the set of cell keys currently selected. The component
  // never owns selection state itself; it only reflects this prop visually
  // and emits onSelectionChange in response to user gestures.
  selected?: string[];
  // onSelectionChange fires when a user interaction would alter the
  // selection. The component computes the next array (including
  // shift-click region geometry) and hands it to the consumer to store.
  onSelectionChange?: (next: string[]) => void;
  // editable gates whether cell content edits, resize gestures, select
  // gestures, structural mutations (add/remove row/col, paste, clear), and
  // the in-menu structural items fire.
  editable?: boolean;
  // onEditableChange, when defined, surfaces a "Enable/Disable editing"
  // item in the context menu. The callback receives the toggled value.
  onEditableChange?: (editable: boolean) => void;
  // extraMenuItems is appended to the default context menu items so
  // consumers can add app-specific entries (e.g. "Reload Console").
  extraMenuItems?: ReactNode;
  // enableTriggers gates the in-table keyboard shortcuts (Delete/Backspace
  // to clear, Cmd+Z to undo, Cmd+Shift+Z to redo). Defaults to true; pass a
  // function to gate dynamically (e.g. only the focused mosaic tab).
  enableTriggers?: boolean | (() => boolean);
}

export const Table = ({
  resourceKey: key,
  selected = [],
  onSelectionChange,
  editable = false,
  onEditableChange,
  extraMenuItems,
  enableTriggers = true,
  visible,
  className,
  ...rest
}: TableProps): ReactElement => {
  useEnsureRetrieved({ key });
  const rows = useSelectRows({ key });
  const columns = useSelectColumns({ key });
  const { dispatch } = useDispatch();
  const theme = Theming.use();

  const addRow = useCallback(
    (atIndex?: number) =>
      dispatch({
        key,
        actions: [
          table.addRow({
            index: atIndex ?? math.MAX_UINT32,
            size: BASE_ROW_SIZE,
            cells: [],
            cellTemplate: newDefaultCell(theme),
          }),
        ],
      }),
    [dispatch, key, theme],
  );
  const addCol = useCallback(
    (atIndex?: number) =>
      dispatch({
        key,
        actions: [
          table.addCol({
            index: atIndex ?? math.MAX_UINT32,
            size: BASE_COL_SIZE,
            cells: [],
            cellTemplate: newDefaultCell(theme),
          }),
        ],
      }),
    [dispatch, key, theme],
  );
  const removeRow = useCallback(
    (index: number) => dispatch({ key, actions: [table.removeRow({ index })] }),
    [dispatch, key],
  );
  const removeCol = useCallback(
    (index: number) => dispatch({ key, actions: [table.removeCol({ index })] }),
    [dispatch, key],
  );
  const eraseSelected = useCallback(
    (selected: string[]) => {
      if (selected.length === 0) return;
      dispatch({
        key,
        actions: [
          table.eraseCells({
            cells: selected,
            template: {
              key: "",
              variant: "text",
              props: CELLS.text.defaultProps(theme),
            },
          }),
        ],
      });
    },
    [dispatch, key, theme],
  );
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });

  const handlePaste = useCallback(
    (overwritten: string[]) => {
      if (overwritten.length !== 0) onSelectionChange?.(overwritten);
    },
    [onSelectionChange],
  );
  const { onCopy, onPaste } = useClipboard({
    key,
    selected,
    onPaste: handlePaste,
  });

  const menuProps = Menu.useContextMenu();
  const renderMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => (
      <DefaultContextMenu
        resourceKey={key}
        targetID={keys[0] ?? null}
        editable={editable}
        onEditableChange={onEditableChange}
        onAddRow={addRow}
        onAddCol={addCol}
        onRemoveRow={removeRow}
        onRemoveCol={removeCol}
        extra={extraMenuItems}
      />
    ),
    [
      key,
      editable,
      onEditableChange,
      addRow,
      addCol,
      removeRow,
      removeCol,
      extraMenuItems,
    ],
  );

  const [{ path }, , setState] = Aether.use({
    type: aetherTable.Table.TYPE,
    schema: aetherTable.Table.stateZ,
    initialState: { region: box.ZERO, visible },
  });

  useEffect(() => setState((s) => ({ ...s, visible })), [visible]);

  const canvasRef = Canvas.useRegion((b) => setState((s) => ({ ...s, region: b })));

  const selectedRef = useSyncedRef(selected);
  const lastSelectedRef = useRef<string | null>(null);
  const rowsRef = useSyncedRef(rows);
  const tableElRef = useRef<HTMLTableElement>(null);

  Triggers.use({
    triggers: FLATTENED_TRIGGERS_CONFIG,
    region: tableElRef,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start" || !editable) return;
        if (enableTriggers === false) return;
        if (typeof enableTriggers === "function" && !enableTriggers()) return;
        const mode = Triggers.determineMode(TRIGGERS_CONFIG, triggers);
        if (mode === "clear") {
          if (selected.length === 0) return;
          eraseSelected(selected);
        } else if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [editable, enableTriggers, selected, eraseSelected, undo, redo],
    ),
  });

  const handleCellSelect = useCallback(
    (cellKey: string, ev: MouseEvent) => {
      if (!editable) return;
      tableElRef.current?.focus({ preventScroll: true });
      const { shiftKey, ctrlKey, metaKey } = ev;
      if (shiftKey && lastSelectedRef.current != null) {
        const start = findCellPosition(rowsRef.current, lastSelectedRef.current);
        const end = findCellPosition(rowsRef.current, cellKey);
        if (start != null && end != null) {
          onSelectionChange?.(cellsInRegion(rowsRef.current, start, end));
          return;
        }
      }
      if (ctrlKey || metaKey) {
        const next = new Set(selectedRef.current);
        if (next.has(cellKey)) next.delete(cellKey);
        else next.add(cellKey);
        lastSelectedRef.current = cellKey;
        onSelectionChange?.(Array.from(next));
        return;
      }
      lastSelectedRef.current = cellKey;
      onSelectionChange?.([cellKey]);
    },
    [editable, onSelectionChange],
  );

  const handleRowSelect = useCallback(
    (index: number) => {
      if (!editable) return;
      tableElRef.current?.focus({ preventScroll: true });
      const row = rowsRef.current[index];
      if (row == null) return;
      lastSelectedRef.current = row.cells[row.cells.length - 1] ?? null;
      onSelectionChange?.(row.cells);
    },
    [editable, onSelectionChange],
  );

  const handleColSelect = useCallback(
    (index: number) => {
      if (!editable) return;
      tableElRef.current?.focus({ preventScroll: true });
      const colCells = rowsRef.current
        .map((r) => r.cells[index])
        .filter((k): k is string => k != null);
      lastSelectedRef.current = colCells[colCells.length - 1] ?? null;
      onSelectionChange?.(colCells);
    },
    [editable, onSelectionChange],
  );

  const handleRowResize = useCallback(
    (size: number, index: number) => {
      if (editable) dispatch({ key, actions: [table.resizeRow({ index, size })] });
    },
    [dispatch, editable, key],
  );

  const handleColResize = useCallback(
    (size: number, index: number) => {
      if (editable) dispatch({ key, actions: [table.resizeCol({ index, size })] });
    },
    [dispatch, editable, key],
  );

  const colSizes = columns.map((c) => c.size);
  const totalCol = colSizes.reduce((a, s) => a + s, 0);
  const totalRow = rows.reduce((a, r) => a + r.size, 0);

  let rowYCursor = 3.5 * 6;
  return (
    <div
      className={CSS(CSS.B("table-frame"), CSS.editable(editable), className)}
      {...rest}
    >
      <Menu.ContextMenu menu={renderMenu} {...menuProps}>
        <div ref={canvasRef} className={CSS.BE("table-frame", "canvas")} />
        <table
          ref={tableElRef}
          className={CSS(CSS.B("table"), menuProps.className)}
          style={{ width: totalCol, height: totalRow }}
          onContextMenu={menuProps.open}
          onCopy={onCopy}
          onPaste={editable ? onPaste : undefined}
          tabIndex={-1}
        >
          <tbody>
            <Aether.Composite path={path}>
              <ColumnIndicators
                columns={colSizes}
                rows={rows}
                selected={selected}
                editable={editable}
                onSelect={handleColSelect}
                onResize={handleColResize}
              />
              {rows.map((row, rowIndex) => {
                const yPos = rowYCursor;
                rowYCursor += row.size;
                return (
                  <Row
                    key={rowIndex}
                    index={rowIndex}
                    resourceKey={key}
                    cells={row.cells}
                    columns={colSizes}
                    position={yPos}
                    size={row.size}
                    selected={selected}
                    editable={editable}
                    onSelect={handleRowSelect}
                    onResize={handleRowResize}
                    onCellSelect={handleCellSelect}
                  />
                );
              })}
            </Aether.Composite>
          </tbody>
        </table>
      </Menu.ContextMenu>
      {editable && (
        <>
          <Button.Button
            className={CSS.BE("table-frame", "add-col")}
            justify="center"
            align="center"
            size="tiny"
            variant="filled"
            onClick={() => addCol()}
          >
            <Icon.Add />
          </Button.Button>
          <Button.Button
            className={CSS.BE("table-frame", "add-row")}
            justify="center"
            align="center"
            size="tiny"
            variant="filled"
            onClick={() => addRow()}
          >
            <Icon.Add />
          </Button.Button>
        </>
      )}
    </div>
  );
};
