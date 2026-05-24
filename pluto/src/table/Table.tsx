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
  memo,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
import { Menu } from "@/menu";
import { table as aetherTable } from "@/table/aether";
import { CELLS } from "@/table/cells/registry";
import {
  cellsInRegion,
  findCellPosition,
  MIN_CELL_DIM,
  useDispatch,
  useEnsureRetrieved,
  useSelectCell,
  useSelectColumns,
  useSelectRows,
} from "@/table/queries";
import { Text } from "@/text";
import { stopPropagation } from "@/util/event";
import { Canvas } from "@/vis/canvas";

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

// getCellColumn maps a 0-based column index to a spreadsheet-style letter
// ("A", "B", "C", ...). Defined here so consumers building UI chrome (e.g.,
// breadcrumb labels in a toolbar) can label cells using the same convention
// the table renders.
export const getCellColumn = (index: number): string => ALPHABET[index];

export interface ContextMenuTarget {
  // cellKey identifies the cell that was right-clicked, or null when the
  // user right-clicked a resizer or the empty area of the table.
  cellKey: string | null;
  // rowResizerIndex identifies a right-clicked row resizer, or null when
  // the target isn't a row resizer.
  rowResizerIndex: number | null;
  // colResizerIndex identifies a right-clicked column resizer, or null when
  // the target isn't a column resizer.
  colResizerIndex: number | null;
}

export interface TableProps extends Pick<
  z.infer<typeof aetherTable.Table.stateZ>,
  "visible"
> {
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
  // editable gates whether cell content edits, resize gestures, and select
  // gestures fire dispatch and selection changes.
  editable?: boolean;
  // onContextMenu fires when the user right-clicks anywhere inside the
  // table. The metadata identifies the click target (cell, row resizer,
  // column resizer, or empty area) so consumers can render the appropriate
  // menu items.
  onContextMenu?: (e: React.MouseEvent, target: ContextMenuTarget) => void;
  // className passes through to the rendered <table> element.
  className?: string;
  // children renders inside the table's <tbody> after the row indicators
  // and rows.
  children?: ReactNode;
}

export const Table = ({
  resourceKey,
  selected = [],
  onSelectionChange,
  editable = false,
  visible,
  onContextMenu,
  className,
  children,
}: TableProps): ReactElement => {
  useEnsureRetrieved({ key: resourceKey });
  const rows = useSelectRows({ key: resourceKey });
  const columns = useSelectColumns({ key: resourceKey });
  const { dispatch } = useDispatch();

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

  const handleCellSelect = useCallback(
    (cellKey: string, ev: MouseEvent) => {
      if (!editable) return;
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

  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      if (onContextMenu == null) return;
      const targetEl = e.target as HTMLElement;
      const tdEl = targetEl.closest<HTMLElement>("td[id]");
      let cellKey: string | null = null;
      let rowResizerIndex: number | null = null;
      let colResizerIndex: number | null = null;
      if (tdEl != null) {
        const id = tdEl.id;
        if (id.startsWith("resizer-")) {
          const [, dir, idx] = id.split("-");
          const parsed = Number.parseInt(idx, 10);
          if (Number.isFinite(parsed)) 
            if (dir === "x") colResizerIndex = parsed;
            else if (dir === "y") rowResizerIndex = parsed;
          
        } else cellKey = id;
      }
      onContextMenu(e, { cellKey, rowResizerIndex, colResizerIndex });
    },
    [onContextMenu],
  );

  const colSizes = columns.map((c) => c.size);
  const totalCol = colSizes.reduce((a, s) => a + s, 0);
  const totalRow = rows.reduce((a, r) => a + r.size, 0);

  let rowYCursor = 3.5 * 6;
  return (
    <>
      <div
        ref={canvasRef}
        style={{
          right: 0,
          bottom: 0,
          position: "absolute",
          top: 6,
          left: 6,
        }}
      />
      <table
        className={CSS(CSS.B("table"), className)}
        style={{ width: totalCol, height: totalRow }}
        onContextMenu={handleContextMenu}
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
            {children}
          </Aether.Composite>
        </tbody>
      </table>
    </>
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

export interface CellProps extends React.ComponentPropsWithRef<"td"> {
  selected?: boolean;
}

// Cell is the primitive <td> wrapper used by variant cell components in
// @/table/cells. The connected Table renders cell content through the
// variant registry; variant components compose this primitive to get
// consistent styling and selection visuals.
export const Cell = ({
  ref,
  children,
  className,
  selected = false,
  ...rest
}: CellProps): ReactElement => (
  <td
    ref={ref}
    {...rest}
    className={CSS(CSS.BE("table", "cell"), CSS.selected(selected), className)}
  >
    {children}
  </td>
);

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
