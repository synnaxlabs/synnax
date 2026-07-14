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
import { box, id, math, type xy } from "@synnaxlabs/x";
import {
  type ComponentPropsWithRef,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { Menu } from "@/menu";
import { Select } from "@/select";
import { AddCountControl } from "@/table/AddCountControl";
import { table as aetherTable } from "@/table/aether";
import { Cell } from "@/table/cells";
import { useClipboard } from "@/table/clipboard";
import { DefaultContextMenu } from "@/table/ContextMenu";
import { ColumnIndicators } from "@/table/Indicator";
import {
  cellsInRegion,
  findCellPosition,
  nextCellPosition,
  useDispatch,
  useEnsureRetrieved,
  useRedo,
  useSelectColumns,
  useSelectRows,
  useUndo,
} from "@/table/queries";
import { Row } from "@/table/Row";
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

const NAV_KEYS = new Set<Triggers.Key>([
  "Tab",
  "ArrowLeft",
  "ArrowRight",
  "ArrowUp",
  "ArrowDown",
]);

const BASE_ROW_SIZE = 36;
const BASE_COL_SIZE = 72;

const newDefaultCell = (): table.Cell => ({
  key: id.create(),
  config: Cell.defaultConfig("text"),
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
  // showIndicators controls whether the row and column indicator strips
  // render. Defaults to true. Consumers typically force this to true while
  // editable is true so structural controls remain reachable, and let it
  // follow a persistent preference otherwise.
  showIndicators?: boolean;
  // onShowIndicatorsChange, when defined, surfaces a Show / Hide indicators
  // item in the context menu while editable is false. The callback receives
  // the next visibility value.
  onShowIndicatorsChange?: (next: boolean) => void;
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
  showIndicators = true,
  onShowIndicatorsChange,
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

  const addRow = useCallback(
    (atIndex?: number, count: number = 1) => {
      if (count < 1) return;
      const actions: table.Action[] = [];
      for (let i = 0; i < count; i++)
        actions.push(
          table.addRow({
            index: atIndex ?? math.MAX_UINT32,
            size: BASE_ROW_SIZE,
            cells: [],
            cellTemplate: newDefaultCell(),
          }),
        );
      dispatch({ key, actions });
    },
    [dispatch, key],
  );
  const addCol = useCallback(
    (atIndex?: number, count: number = 1) => {
      if (count < 1) return;
      const actions: table.Action[] = [];
      for (let i = 0; i < count; i++)
        actions.push(
          table.addCol({
            index: atIndex ?? math.MAX_UINT32,
            size: BASE_COL_SIZE,
            cells: [],
            cellTemplate: newDefaultCell(),
          }),
        );
      dispatch({ key, actions });
    },
    [dispatch, key],
  );
  const removeRow = useCallback(
    (indices: number[]) => {
      if (indices.length === 0) return;
      // Descending order so earlier removes don't shift later indices.
      const sorted = [...indices].sort((a, b) => b - a);
      dispatch({
        key,
        actions: sorted.map((index) => table.removeRow({ index })),
      });
    },
    [dispatch, key],
  );
  const removeCol = useCallback(
    (indices: number[]) => {
      if (indices.length === 0) return;
      const sorted = [...indices].sort((a, b) => b - a);
      dispatch({
        key,
        actions: sorted.map((index) => table.removeCol({ index })),
      });
    },
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
            template: Cell.defaultConfig("text"),
          }),
        ],
      });
    },
    [dispatch, key],
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
        selected={selected}
        editable={editable}
        onEditableChange={onEditableChange}
        showIndicators={showIndicators}
        onShowIndicatorsChange={onShowIndicatorsChange}
        onAddRow={addRow}
        onAddCol={addCol}
        onRemoveRow={removeRow}
        onRemoveCol={removeCol}
        onEraseCells={eraseSelected}
        extra={extraMenuItems}
      />
    ),
    [
      key,
      selected,
      editable,
      onEditableChange,
      showIndicators,
      onShowIndicatorsChange,
      addRow,
      addCol,
      removeRow,
      removeCol,
      eraseSelected,
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
    (cellKey: string, ev: React.MouseEvent) => {
      if (!editable) return;
      tableElRef.current?.focus({ preventScroll: true });
      const { shiftKey, ctrlKey, metaKey, type } = ev;
      if (type === "contextmenu" && selectedRef.current.includes(cellKey)) return;
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
    (index: number, ev: React.MouseEvent) => {
      if (!editable) return;
      tableElRef.current?.focus({ preventScroll: true });
      const row = rowsRef.current[index];
      if (row == null) return;
      const { shiftKey, ctrlKey, metaKey, type } = ev;
      if (
        type === "contextmenu" &&
        row.cells.every((k) => selectedRef.current.includes(k))
      )
        return;
      if (shiftKey && lastSelectedRef.current != null) {
        const anchor = findCellPosition(rowsRef.current, lastSelectedRef.current);
        if (anchor != null) {
          const minY = Math.min(anchor.y, index);
          const maxY = Math.max(anchor.y, index);
          const cells: string[] = [];
          for (let y = minY; y <= maxY; y++)
            for (const k of rowsRef.current[y]?.cells ?? []) cells.push(k);
          lastSelectedRef.current = row.cells[row.cells.length - 1] ?? null;
          onSelectionChange?.(cells);
          return;
        }
      }
      if (ctrlKey || metaKey) {
        const next = new Set(selectedRef.current);
        for (const k of row.cells) next.add(k);
        lastSelectedRef.current = row.cells[row.cells.length - 1] ?? null;
        onSelectionChange?.(Array.from(next));
        return;
      }
      lastSelectedRef.current = row.cells[row.cells.length - 1] ?? null;
      onSelectionChange?.(row.cells);
    },
    [editable, onSelectionChange],
  );

  const handleColSelect = useCallback(
    (index: number, ev: React.MouseEvent) => {
      if (!editable) return;
      tableElRef.current?.focus({ preventScroll: true });
      const colCells = rowsRef.current
        .map((r) => r.cells[index])
        .filter((k): k is string => k != null);
      const { shiftKey, ctrlKey, metaKey, type } = ev;
      if (
        type === "contextmenu" &&
        colCells.every((k) => selectedRef.current.includes(k))
      )
        return;
      if (shiftKey && lastSelectedRef.current != null) {
        const anchor = findCellPosition(rowsRef.current, lastSelectedRef.current);
        if (anchor != null) {
          const minX = Math.min(anchor.x, index);
          const maxX = Math.max(anchor.x, index);
          const cells: string[] = [];
          for (const row of rowsRef.current)
            for (let x = minX; x <= maxX; x++) {
              const k = row.cells[x];
              if (k != null) cells.push(k);
            }
          lastSelectedRef.current = colCells[colCells.length - 1] ?? null;
          onSelectionChange?.(cells);
          return;
        }
      }
      if (ctrlKey || metaKey) {
        const next = new Set(selectedRef.current);
        for (const k of colCells) next.add(k);
        lastSelectedRef.current = colCells[colCells.length - 1] ?? null;
        onSelectionChange?.(Array.from(next));
        return;
      }
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

  const handleSelectAll = useCallback(() => {
    if (!editable) return;
    tableElRef.current?.focus({ preventScroll: true });
    const all: string[] = [];
    for (const row of rowsRef.current) for (const k of row.cells) all.push(k);
    lastSelectedRef.current = all[all.length - 1] ?? null;
    onSelectionChange?.(all);
  }, [editable, onSelectionChange]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTableElement>) => {
      if (!editable) return;
      const key = Triggers.eventKey(e);
      if (!NAV_KEYS.has(key)) return;
      const anchor = lastSelectedRef.current;
      if (anchor == null) return;
      const pos = findCellPosition(rowsRef.current, anchor);
      if (pos == null) return;
      e.preventDefault();
      let next: xy.XY | null = null;
      if (key === "Tab")
        next = nextCellPosition(rowsRef.current, pos, e.shiftKey ? -1 : 1);
      else if (key === "ArrowRight" && pos.x + 1 < rowsRef.current[pos.y].cells.length)
        next = { x: pos.x + 1, y: pos.y };
      else if (key === "ArrowLeft" && pos.x > 0) next = { x: pos.x - 1, y: pos.y };
      else if (key === "ArrowDown" && pos.y + 1 < rowsRef.current.length)
        next = { x: pos.x, y: pos.y + 1 };
      else if (key === "ArrowUp" && pos.y > 0) next = { x: pos.x, y: pos.y - 1 };
      if (next == null) return;
      const nextKey = rowsRef.current[next.y]?.cells[next.x];
      if (nextKey == null) return;
      lastSelectedRef.current = nextKey;
      onSelectionChange?.([nextKey]);
    },
    [editable, onSelectionChange],
  );

  const colSizes = useMemo(() => columns.map((c) => c.size), [columns]);
  const totalCol = colSizes.reduce((a, s) => a + s, 0);
  const totalRow = rows.reduce((a, r) => a + r.size, 0);

  let rowYCursor = showIndicators ? 4.5 * 6 : 0;
  return (
    <div
      className={CSS(CSS.B("table-surface"), className)}
      onContextMenu={menuProps.open}
      {...rest}
    >
      <div ref={canvasRef} className={CSS.BE("table-surface", "canvas")} />
      <div className={CSS(CSS.B("table-frame"), CSS.editable(editable))}>
        <Menu.ContextMenu menu={renderMenu} {...menuProps}>
          <table
            ref={tableElRef}
            className={CSS(CSS.B("table"), menuProps.className)}
            style={{ width: totalCol, height: totalRow }}
            onCopy={onCopy}
            onPaste={editable ? onPaste : undefined}
            onKeyDown={handleKeyDown}
            tabIndex={-1}
          >
            <tbody>
              <Aether.Composite path={path}>
                <Select.Provider value={selected}>
                  {showIndicators && (
                    <ColumnIndicators
                      columns={colSizes}
                      rows={rows}
                      selected={selected}
                      editable={editable}
                      onSelect={handleColSelect}
                      onSelectAll={handleSelectAll}
                      onResize={handleColResize}
                    />
                  )}
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
                        editable={editable}
                        showIndicator={showIndicators}
                        onSelect={handleRowSelect}
                        onResize={handleRowResize}
                        onCellSelect={handleCellSelect}
                      />
                    );
                  })}
                </Select.Provider>
              </Aether.Composite>
            </tbody>
          </table>
        </Menu.ContextMenu>
        {editable && (
          <>
            <AddCountControl
              className={CSS.BE("table-frame", "add-col")}
              resourceName="column"
              onAdd={(n) => addCol(undefined, n)}
            />
            <AddCountControl
              className={CSS.BE("table-frame", "add-row")}
              resourceName="row"
              onAdd={(n) => addRow(undefined, n)}
            />
          </>
        )}
      </div>
    </div>
  );
};
