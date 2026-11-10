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
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { box, id, math, xy } from "@synnaxlabs/x";
import {
  type ComponentPropsWithRef,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { AddCountControl } from "@/table/AddCountControl";
import { table as aetherTable } from "@/table/aether";
import { Cell } from "@/table/cells";
import { useClipboard } from "@/table/clipboard";
import { DefaultContextMenu, ERASE_TRIGGER } from "@/table/ContextMenu";
import { ColumnIndicators, INDICATOR_SIZE } from "@/table/Indicator";
import { Overlay, shifts, steps, useMove } from "@/table/move";
import {
  cellsInRegion,
  findCellPosition,
  nextCellPosition,
  useColumns,
  useDispatch,
  useRedo,
  useRows,
  useUndo,
} from "@/table/queries";
import { Row } from "@/table/Row";
import { Selection } from "@/table/selection";
import { useKey } from "@/table/Suspended";
import { Canvas } from "@/vis/canvas";

export { getCellColumn } from "@/table/Indicator";

const ERASE_TRIGGERS: Triggers.Trigger[] = [ERASE_TRIGGER, ["Backspace"]];

const NAV_KEYS = new Set<Triggers.Key>([
  "Tab",
  "ArrowLeft",
  "ArrowRight",
  "ArrowUp",
  "ArrowDown",
]);

// The base sizes come from the schema, which declares the default size of a row and a
// column.
const BASE_ROW_SIZE = table.rowZ.parse({}).size;
const BASE_COL_SIZE = table.columnZ.parse({}).size;
const NONE_HIDDEN: string[] = [];

// fullySelectedRows returns the keys of the rows whose every cell is selected.
const fullySelectedRows = (rows: table.Row[], selected: string[]): string[] => {
  const set = new Set(selected);
  return rows.flatMap((row) =>
    row.cells.length > 0 && row.cells.every((k) => set.has(k)) ? [row.key] : [],
  );
};

// fullySelectedCols returns the keys of the columns whose every cell is selected.
const fullySelectedCols = (
  rows: table.Row[],
  columns: table.Column[],
  selected: string[],
): string[] => {
  const set = new Set(selected);
  return columns.flatMap((col, x) =>
    rows.length > 0 && rows.every((row) => set.has(row.cells[x])) ? [col.key] : [],
  );
};

// displacedCells returns the keys of the cells a move shifts out of place.
const displacedCells = (
  rows: table.Row[],
  rowShifts: number[] | undefined,
  colShifts: number[] | undefined,
): string[] => {
  if (rowShifts == null && colShifts == null) return NONE_HIDDEN;
  const keys: string[] = [];
  rows.forEach((row, y) =>
    row.cells.forEach((cellKey, x) => {
      if ((rowShifts?.[y] ?? 0) !== 0 || (colShifts?.[x] ?? 0) !== 0)
        keys.push(cellKey);
    }),
  );
  return keys;
};

// centerOffset returns the offset that centers content of the given size in region.
const centerOffset = (region: box.Box, width: number, height: number): xy.XY => ({
  x: Math.max(0, Math.floor((box.width(region) - width) / 2)),
  y: Math.max(0, Math.floor((box.height(region) - height) / 2)),
});

const newDefaultCell = (): table.Cell => ({
  key: id.create(),
  config: Cell.defaultConfig("text"),
});

export interface TableProps
  extends
    Omit<ComponentPropsWithRef<"div">, "onCopy" | "onPaste" | "onContextMenu">,
    Pick<z.infer<typeof aetherTable.Table.stateZ>, "visible"> {
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
  /** Centers the table in its container. No effect on an axis it overflows. */
  centered?: boolean;
  /** When defined, surfaces a Center / Align item in the context menu. */
  onCenteredChange?: (next: boolean) => void;
  // extraMenuItems is appended to the default context menu items so
  // consumers can add app-specific entries (e.g. "Reload Console").
  extraMenuItems?: ReactNode;
  // enableTriggers gates the in-table keyboard shortcuts (Delete/Backspace
  // to clear, Cmd+Z to undo, Cmd+Shift+Z to redo). Defaults to true.
  enableTriggers?: Triggers.Condition;
}

export const Table = ({
  selected = [],
  onSelectionChange,
  editable = false,
  onEditableChange,
  showIndicators = true,
  onShowIndicatorsChange,
  centered = false,
  onCenteredChange,
  extraMenuItems,
  enableTriggers = true,
  visible,
  className,
  ...rest
}: TableProps): ReactElement => {
  const key = useKey();
  const rows = useRows({ key });
  const columns = useColumns({ key });
  const { dispatch } = useDispatch();

  const addRow = (atIndex?: number, count: number = 1) => {
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
  };
  const addCol = (atIndex?: number, count: number = 1) => {
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
  };
  const removeRow = (keys: string[]) => {
    if (keys.length === 0) return;
    dispatch({
      key,
      actions: keys.map((rowKey) => table.removeRow({ key: rowKey })),
    });
  };
  const removeCol = (keys: string[]) => {
    if (keys.length === 0) return;
    dispatch({
      key,
      actions: keys.map((colKey) => table.removeCol({ key: colKey })),
    });
  };
  const eraseSelected = (selected: string[]) => {
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
  };
  const { undo } = useUndo({ key });
  const { redo } = useRedo({ key });

  const handlePaste = (overwritten: string[]) => {
    if (overwritten.length !== 0) onSelectionChange?.(overwritten);
  };
  const { onCopy, onPaste } = useClipboard({
    key,
    selected,
    onPaste: handlePaste,
  });

  const menuProps = Menu.useContextMenu();
  const renderMenu = ({ keys }: Menu.ContextMenuMenuProps) => (
    <DefaultContextMenu
      resourceKey={key}
      targetID={keys[0] ?? null}
      selected={selected}
      editable={editable}
      onEditableChange={onEditableChange}
      showIndicators={showIndicators}
      onShowIndicatorsChange={onShowIndicatorsChange}
      centered={centered}
      onCenteredChange={onCenteredChange}
      onAddRow={addRow}
      onAddCol={addCol}
      onRemoveRow={removeRow}
      onRemoveCol={removeCol}
      onEraseCells={eraseSelected}
      extra={extraMenuItems}
    />
  );

  const { path, setState } = Aether.useLifecycle({
    type: aetherTable.Table.TYPE,
    schema: aetherTable.Table.stateZ,
    initialState: { region: box.ZERO, visible },
  });

  useEffect(() => setState((s) => ({ ...s, visible })), [visible]);

  const [region, setRegion] = useState<box.Box>(box.ZERO);
  const canvasRef = Canvas.useRegion((b) => {
    setRegion(b);
    setState((s) => ({ ...s, region: b }));
  });

  const handleScroll = ({
    currentTarget: { scrollLeft, scrollTop },
  }: React.UIEvent<HTMLDivElement>) =>
    setState((s) => ({ ...s, scroll: { x: scrollLeft, y: scrollTop } }));

  const selectedRef = useSyncedRef(selected);
  const lastSelectedRef = useRef<string | null>(null);
  const rowsRef = useSyncedRef(rows);
  const tableElRef = useRef<HTMLTableElement>(null);

  const selectedRows = fullySelectedRows(rows, selected);
  const selectedCols = fullySelectedCols(rows, columns, selected);

  // A move selects what it moved. Cell keys travel with their row or column, so the
  // keys read before the dispatch name the cells at their new position.
  const finishMove = (cells: string[], actions: table.Action[]): void => {
    if (actions.length > 0) dispatch({ key, actions });
    tableElRef.current?.focus({ preventScroll: true });
    lastSelectedRef.current = cells.at(-1) ?? null;
    onSelectionChange?.(cells);
  };
  const rowMove = useMove({
    direction: "y",
    items: rows,
    selected: selectedRows,
    ref: tableElRef,
    onMove: (move) =>
      finishMove(
        move.indices.flatMap((y) => rows[y].cells),
        steps(move).map(({ from, to }) => table.moveRow({ key: rows[from].key, to })),
      ),
  });
  const colMove = useMove({
    direction: "x",
    items: columns,
    selected: selectedCols,
    ref: tableElRef,
    onMove: (move) =>
      finishMove(
        rows.flatMap((row) => move.indices.flatMap((x) => row.cells[x] ?? [])),
        steps(move).map(({ from, to }) =>
          table.moveCol({ key: columns[from].key, to }),
        ),
      ),
  });
  // A method call would make the whole move object, new on each render, a dependency.
  const consumeRowMoved = rowMove.consumeMoved;
  const consumeColMoved = colMove.consumeMoved;

  const rowSizes = rows.map((r) => r.size);
  const totalRow = rowSizes.reduce((a, s) => a + s, 0);
  const colSizes = columns.map((c) => c.size);
  const totalCol = colSizes.reduce((a, s) => a + s, 0);
  const rowShifts = rowMove.move == null ? undefined : shifts(rows, rowMove.move);
  const colShifts = colMove.move == null ? undefined : shifts(columns, colMove.move);
  // The canvas does not slide, so value cells a move displaces draw nothing until it
  // ends.
  const hidden = displacedCells(rows, rowShifts, colShifts);
  useEffect(() => setState((s) => ({ ...s, hidden })), [hidden]);

  // Erase stays bound to the table's own region so Delete pressed elsewhere in the tab
  // cannot wipe cells. Undo and redo are tab-wide, matching the other visualizations.
  Triggers.use({
    triggers: ERASE_TRIGGERS,
    region: tableElRef,
    enabled: enableTriggers,
    callback: ({ stage }: Triggers.UseEvent) => {
      if (stage !== "start" || !editable || selected.length === 0) return;
      eraseSelected(selected);
    },
  });
  Triggers.useUndoRedo({
    undo,
    redo,
    enabled: () => editable && Triggers.resolveCondition(enableTriggers),
  });

  const handleCellSelect = (cellKey: string, ev: React.MouseEvent) => {
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
  };

  const handleRowSelect = (index: number, ev: React.MouseEvent) => {
    if (!editable || consumeRowMoved()) return;
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
  };

  const handleColSelect = (index: number, ev: React.MouseEvent) => {
    if (!editable || consumeColMoved()) return;
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
  };

  const handleRowResize = (size: number, rowKey: string) => {
    if (editable) dispatch({ key, actions: [table.resizeRow({ key: rowKey, size })] });
  };

  const handleColResize = (size: number, colKey: string) => {
    if (editable) dispatch({ key, actions: [table.resizeCol({ key: colKey, size })] });
  };

  const handleSelectAll = () => {
    if (!editable) return;
    tableElRef.current?.focus({ preventScroll: true });
    const all: string[] = [];
    for (const row of rowsRef.current) for (const k of row.cells) all.push(k);
    lastSelectedRef.current = all[all.length - 1] ?? null;
    onSelectionChange?.(all);
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTableElement>) => {
    if (!editable) return;
    const key = Triggers.eventKey(e);
    if (key == null || !NAV_KEYS.has(key)) return;
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
  };

  const tableStyle = { width: totalCol, height: totalRow };

  const indicatorSize = showIndicators ? INDICATOR_SIZE : 0;
  // One offset feeds the frame's translate and the canvas cell boxes.
  const liveOffset = centered
    ? centerOffset(region, totalCol + indicatorSize, totalRow + indicatorSize)
    : xy.ZERO;
  // A live offset would slide a dragged edge out from under the cursor.
  const [held, setHeld] = useState<xy.XY | null>(null);
  const handlePointerDown = () => {
    if (!centered) return;
    setHeld(liveOffset);
    const listening = new AbortController();
    const release = (): void => {
      listening.abort();
      setHeld(null);
    };
    window.addEventListener("pointerup", release, { signal: listening.signal });
    window.addEventListener("pointercancel", release, { signal: listening.signal });
  };
  const offset = centered ? (held ?? liveOffset) : xy.ZERO;
  const frameStyle =
    offset.x === 0 && offset.y === 0
      ? undefined
      : { transform: `translate(${offset.x}px, ${offset.y}px)` };

  const rowYs: number[] = [];
  let rowYCursor = offset.y + indicatorSize;
  for (const row of rows) {
    rowYs.push(rowYCursor);
    rowYCursor += row.size;
  }
  const cellXOrigin = offset.x + indicatorSize;
  return (
    <div
      className={CSS.cls(CSS.B("table-surface"), className)}
      onContextMenu={menuProps.open}
      {...rest}
    >
      <div ref={canvasRef} className={CSS.BE("table-surface", "canvas")} />
      <div className={CSS.BE("table-surface", "scroll")} onScroll={handleScroll}>
        <div
          className={CSS.cls(
            CSS.B("table-frame"),
            CSS.editable(editable),
            (rowShifts != null || colShifts != null) && CSS.BM("table-frame", "moving"),
          )}
          style={frameStyle}
          onPointerDown={handlePointerDown}
        >
          <Menu.ContextMenu menu={renderMenu} {...menuProps}>
            <table
              ref={tableElRef}
              className={CSS.cls(CSS.B("table"), menuProps.className)}
              style={tableStyle}
              onCopy={onCopy}
              onPaste={editable ? onPaste : undefined}
              onKeyDown={handleKeyDown}
              tabIndex={-1}
            >
              <tbody>
                <Aether.Composite path={path}>
                  <Selection.Context value={selected}>
                    {showIndicators && (
                      <ColumnIndicators
                        columns={columns}
                        rows={rows}
                        selected={selected}
                        editable={editable}
                        onSelect={handleColSelect}
                        onSelectAll={handleSelectAll}
                        onResize={handleColResize}
                        onMovePress={editable ? colMove.press : undefined}
                        shifts={colShifts}
                      />
                    )}
                    {rows.map((row, rowIndex) => (
                      <Row
                        key={row.key}
                        index={rowIndex}
                        itemKey={row.key}
                        resourceKey={key}
                        cells={row.cells}
                        columns={colSizes}
                        x={cellXOrigin}
                        y={rowYs[rowIndex]}
                        size={row.size}
                        editable={editable}
                        last={rowIndex === rows.length - 1}
                        showIndicator={showIndicators}
                        onSelect={handleRowSelect}
                        onResize={handleRowResize}
                        onCellSelect={handleCellSelect}
                        onMovePress={editable ? rowMove.press : undefined}
                        shift={rowShifts?.[rowIndex]}
                        colShifts={colShifts}
                      />
                    ))}
                  </Selection.Context>
                </Aether.Composite>
              </tbody>
            </table>
          </Menu.ContextMenu>
          {rowMove.move != null && rowShifts != null && (
            <Overlay
              direction="y"
              move={rowMove.move}
              sizes={rowSizes}
              shifts={rowShifts}
              span={totalCol}
            />
          )}
          {colMove.move != null && colShifts != null && (
            <Overlay
              direction="x"
              move={colMove.move}
              sizes={colSizes}
              shifts={colShifts}
              span={totalRow}
            />
          )}
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
    </div>
  );
};
