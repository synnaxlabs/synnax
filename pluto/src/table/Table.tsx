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
import { box, clamp, dimensions, direction, type record, xy } from "@synnaxlabs/x";
import {
  type ComponentPropsWithRef,
  memo,
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
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Icon } from "@/icon";
import { Menu } from "@/menu";
import { table as aetherTable } from "@/table/aether";
import { CELLS } from "@/table/cells/registry";
import { useClipboard } from "@/table/clipboard";
import {
  cellsInRegion,
  findCellPosition,
  MIN_CELL_DIM,
  useAddCol,
  useAddRow,
  useCellPosition,
  useClearSelected,
  useDispatch,
  useEnsureRetrieved,
  useRedo,
  useRemoveCol,
  useRemoveRow,
  useSelectCell,
  useSelectColumns,
  useSelectRows,
  useUndo,
} from "@/table/queries";
import { Text } from "@/text";
import { Triggers } from "@/triggers";
import { stopPropagation } from "@/util/event";
import { Canvas } from "@/vis/canvas";

type TriggerMode = "clear" | "undo" | "redo" | "default";

const TRIGGERS_CONFIG: Triggers.ModeConfig<TriggerMode> = {
  clear: [["Delete"], ["Backspace"]],
  undo: [["Control", "Z"]],
  redo: [["Control", "Shift", "Z"]],
  default: [],
  defaultMode: "default",
};

const FLATTENED_TRIGGERS_CONFIG = Triggers.flattenConfig(TRIGGERS_CONFIG);

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

// getCellColumn maps a 0-based column index to a spreadsheet-style letter
// ("A", "B", "C", ...). Defined here so consumers building UI chrome (e.g.,
// breadcrumb labels in a toolbar) can label cells using the same convention
// the table renders.
export const getCellColumn = (index: number): string => ALPHABET[index];

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
  resourceKey,
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
  useEnsureRetrieved({ key: resourceKey });
  const rows = useSelectRows({ key: resourceKey });
  const columns = useSelectColumns({ key: resourceKey });
  const { dispatch } = useDispatch();

  const addRow = useAddRow({ key: resourceKey });
  const addCol = useAddCol({ key: resourceKey });
  const removeRow = useRemoveRow({ key: resourceKey });
  const removeCol = useRemoveCol({ key: resourceKey });
  const clearSelected = useClearSelected({ key: resourceKey });
  const { undo } = useUndo({ key: resourceKey });
  const { redo } = useRedo({ key: resourceKey });

  const handlePasted = useCallback(
    (overwrittenKeys: string[]) => {
      if (overwrittenKeys.length === 0) return;
      onSelectionChange?.(overwrittenKeys);
    },
    [onSelectionChange],
  );
  const { onCopy, onPaste } = useClipboard({
    key: resourceKey,
    selected,
    onPaste: handlePasted,
  });

  const menuProps = Menu.useContextMenu();
  const renderMenu = useCallback(
    ({ keys }: Menu.ContextMenuMenuProps) => (
      <DefaultContextMenu
        resourceKey={resourceKey}
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
      resourceKey,
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
          clearSelected(selected);
        } else if (mode === "undo") undo();
        else if (mode === "redo") redo();
      },
      [editable, enableTriggers, selected, clearSelected, undo, redo],
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
      if (!editable) return;
      dispatch({
        key: resourceKey,
        actions: [table.resizeRow({ index, size: clamp(size, MIN_CELL_DIM) })],
      });
    },
    [dispatch, editable, resourceKey],
  );

  const handleColResize = useCallback(
    (size: number, index: number) => {
      if (!editable) return;
      dispatch({
        key: resourceKey,
        actions: [table.resizeCol({ index, size: clamp(size, MIN_CELL_DIM) })],
      });
    },
    [dispatch, editable, resourceKey],
  );

  const colSizes = columns.map((c) => c.size);
  const totalCol = colSizes.reduce((a, s) => a + s, 0);
  const totalRow = rows.reduce((a, r) => a + r.size, 0);

  let rowYCursor = 3.5 * 6;
  return (
    <div className={CSS(CSS.B("table-frame"), className)} {...rest}>
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
                    resourceKey={resourceKey}
                    cells={row.cells}
                    columns={colSizes}
                    position={yPos}
                    size={row.size}
                    selected={selected}
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

interface DefaultContextMenuProps {
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

const DefaultContextMenu = ({
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

interface RowProps {
  index: number;
  size: number;
  position: number;
  resourceKey: table.Key;
  cells: string[];
  columns: number[];
  selected: string[];
  onResize: (size: number, index: number) => void;
  onSelect: (index: number) => void;
  onCellSelect: (cellKey: string, ev: MouseEvent) => void;
}

const Row = ({
  index,
  size,
  position,
  resourceKey,
  cells,
  columns,
  selected,
  onResize,
  onSelect,
  onCellSelect,
}: RowProps): ReactElement => {
  let xCursor = 3.5 * 6;
  return (
    <tr className={CSS(CSS.BE("table", "row"))}>
      <Indicator
        direction="y"
        index={index}
        value={size}
        position={position}
        onChange={onResize}
        onSelect={onSelect}
      />
      {cells.map((cellKey, i) => {
        const xPos = xCursor;
        xCursor += columns[i];
        return (
          <VariantCell
            key={cellKey}
            resourceKey={resourceKey}
            cellKey={cellKey}
            box={box.construct(
              xy.construct({ x: xPos, y: position }),
              dimensions.construct(columns[i], size),
            )}
            selected={selected.includes(cellKey)}
            onSelect={onCellSelect}
          />
        );
      })}
    </tr>
  );
};

interface VariantCellProps {
  resourceKey: table.Key;
  cellKey: string;
  box: box.Box;
  selected: boolean;
  onSelect: (cellKey: string, ev: MouseEvent) => void;
}

// VariantCell is the bridge between the connected Table and the per-variant
// cell components in @/table/cells. The variant component renders its own
// <td> (via the exported Cell primitive below); VariantCell wires it to flux
// state and a dispatch-backed onChange handler.
const VariantCell = memo(
  ({
    resourceKey,
    cellKey,
    box,
    selected,
    onSelect,
  }: VariantCellProps): ReactElement | null => {
    const cell = useSelectCell({ key: resourceKey, cellKey });
    const { dispatch } = useDispatch();
    const handleChange = useCallback(
      (props: record.Unknown) => {
        if (cell == null) return;
        dispatch({
          key: resourceKey,
          actions: [
            table.setCell({ cell: { key: cellKey, variant: cell.variant, props } }),
          ],
        });
      },
      [dispatch, resourceKey, cellKey, cell],
    );
    if (cell == null) return null;
    const Spec = CELLS[cell.variant as keyof typeof CELLS];
    if (Spec == null) return null;
    return (
      <Spec.Cell
        cellKey={cellKey}
        box={box}
        selected={selected}
        onSelect={onSelect}
        onChange={handleChange}
        {...cell.props}
      />
    );
  },
);
VariantCell.displayName = "VariantCell";

interface ColumnIndicatorsProps {
  columns: number[];
  rows: table.Row[];
  selected: string[];
  onSelect: (index: number) => void;
  onResize: (size: number, index: number) => void;
}

const ColumnIndicators = ({
  columns,
  rows,
  selected,
  onSelect,
  onResize,
}: ColumnIndicatorsProps): ReactElement => {
  const selectedSet = new Set(selected);
  const selectedCols = new Set<number>();
  for (const row of rows)
    row.cells.forEach((k, i) => {
      if (selectedSet.has(k)) selectedCols.add(i);
    });
  let xCursor = 2.5 * 6;
  return (
    <tr className={CSS(CSS.BE("table", "row"), CSS.BE("table", "col-resizer"))}>
      <td />
      {columns.map((size, i) => {
        const xPos = xCursor;
        xCursor += size;
        return (
          <Indicator
            key={i}
            direction="x"
            index={i}
            value={size}
            position={xPos}
            selected={selectedCols.has(i)}
            onChange={onResize}
            onSelect={onSelect}
          />
        );
      })}
    </tr>
  );
};

interface IndicatorProps {
  direction: direction.Direction;
  index: number;
  value: number;
  position: number;
  selected?: boolean;
  onChange: (size: number, index: number) => void;
  onSelect: (index: number) => void;
}

const Indicator = ({
  direction: dir,
  index,
  value,
  position,
  selected = false,
  onChange,
  onSelect,
}: IndicatorProps): ReactElement => {
  const valueRef = useSyncedRef(value);
  const sizeRef = useRef(value);
  const onDragStart = useCursorDrag({
    onStart: useCallback(() => {
      sizeRef.current = valueRef.current;
    }, []),
    onMove: useCallback(
      (b: box.Box) => onChange(sizeRef.current + box.dim(b, dir, true), index),
      [onChange, index, dir],
    ),
  });
  return (
    <td
      id={`resizer-${dir}-${index}`}
      className={CSS(
        CSS.BE("table", "resizer"),
        CSS.dir(dir),
        CSS.selected(selected),
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
      )}
      style={{ [direction.dimension(dir)]: value }}
      onClick={() => onSelect(index)}
      onContextMenu={() => onSelect(index)}
    >
      <Text.Text full="x" justify="center" align="center" square={false}>
        {dir === "x" ? ALPHABET[index] : index + 1}
      </Text.Text>
      <button
        onClick={stopPropagation}
        style={{ [direction.location(dir)]: position + value }}
        onDragStart={onDragStart}
        draggable
      />
    </td>
  );
};
