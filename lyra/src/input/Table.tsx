// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Table.css";

import { csv, type dimensions, grid, strings, xy } from "@synnaxlabs/x";
import {
  type ClipboardEvent,
  type ComponentPropsWithRef,
  type DragEvent,
  type FocusEvent,
  type KeyboardEvent,
  type PropsWithChildren,
  type ReactElement,
  type ReactNode,
  type SyntheticEvent,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";

import { Button } from "@/button";
import type { RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Haul } from "@/haul";
import { Icon } from "@/icon";
import { Checkbox } from "@/input/Checkbox";
import { Numeric } from "@/input/Numeric";
import { ItemContext, type ItemContextValue } from "@/input/scope";
import { Text as InputText } from "@/input/Text";
import type { Control, Variant } from "@/input/types";
import type { Status } from "@/status";
import { Text } from "@/text";
import { reactElementToArray } from "@/util/children";

/** The value one {@link Table} cell holds. The cell's column decides its type. */
export type TableCell = unknown;

/** The props a {@link TableColumn} cell renderer receives. */
export interface TableCellProps<V extends TableCell = TableCell> extends Control<V> {
  /** Whether the cell is read-only. */
  preview?: boolean;
  /** The edit-in-place chassis every cell shares. Spread it onto the input. */
  variant?: Variant;
  "aria-label": string;
}

/** The props a custom column's cell renderer receives. */
export interface TableCustomCellProps<
  V extends TableCell = TableCell,
> extends TableCellProps<V> {
  /** The index of the cell's row. */
  row: number;
}

interface BaseTableColumnProps {
  /** The heading rendered above the column. Omitted for a lone unlabeled column. */
  name?: string;
  /**
   * The width of the column in rem. Omitted, the column shares the width the others
   * leave, down to a minimum. Past the minimum, the table scrolls.
   */
  width?: number;
}

/** The narrowest a column with no width gets, in rem. */
const MIN_SHARED_WIDTH = 16;

/**
 * The column's type decides what a new row holds, how pasted text is read, and which
 * input a cell defaults to. A custom column holds any value and renders it itself.
 */
export type TableColumnProps<V = unknown> =
  | (BaseTableColumnProps & {
      type?: "number";
      /** Reads pasted text, or returns null to refuse it. Defaults to `Number`. */
      parse?: (text: string) => number | null;
      /** Renders the cell. Defaults to a numeric input. */
      children?: RenderProp<TableCellProps<number>>;
    })
  | (BaseTableColumnProps & {
      type: "string";
      /** Reads pasted text, or returns null to refuse it. Defaults to the text. */
      parse?: (text: string) => string | null;
      /** Renders the cell. Defaults to a text input. */
      children?: RenderProp<TableCellProps<string>>;
    })
  | (BaseTableColumnProps & {
      type: "boolean";
      /** Reads pasted text, or returns null to refuse it. Defaults to true or false. */
      parse?: (text: string) => boolean | null;
      /** Renders the cell. Defaults to a checkbox. */
      children?: RenderProp<TableCellProps<boolean>>;
    })
  | (BaseTableColumnProps & {
      type: "custom";
      /** The value a new row holds. */
      empty: V;
      /** Reads pasted text, or returns null to refuse it. Omitted, pastes fail. */
      parse?: (text: string) => V | null;
      children: RenderProp<TableCustomCellProps<V>>;
    });

/**
 * Declares one column of a {@link Table}. It renders nothing itself: the table reads
 * its props to build the header and every row's cell.
 *
 * @example <Input.TableColumn name="Pre-scaled" />
 * @example <Input.TableColumn name="Label" type="string" />
 * @example <Input.TableColumn name="Enabled" type="boolean" />
 */
export const TableColumn = <V,>(_: TableColumnProps<V>): null => null;

// Only a custom renderer gets the row. A typed column's renderer often spreads its
// props onto an input, where the row would land on the DOM.
const renderCell = (
  column: TableColumnProps,
  props: TableCellProps,
  row: number,
): ReactNode => {
  switch (column.type) {
    case "string": {
      const cell = { ...props, value: String(props.value) };
      if (column.children != null) return column.children(cell);
      return <InputText {...cell} />;
    }
    case "boolean": {
      const cell = { ...props, value: props.value === true };
      if (column.children != null) return column.children(cell);
      // A read-only box still shows its state as a control.
      return <Checkbox {...cell} preview={false} disabled={props.preview} />;
    }
    case "custom":
      return column.children({ ...props, row });
    case "number":
    case undefined: {
      const cell = { ...props, value: Number(props.value) };
      if (column.children != null) return column.children(cell);
      return <Numeric {...cell} showDragHandle={false} />;
    }
  }
};

/** The value a new row holds in this column. */
const emptyCell = (column: TableColumnProps): TableCell => {
  switch (column.type) {
    case "string":
      return "";
    case "boolean":
      return false;
    case "custom":
      return column.empty;
    case "number":
    case undefined:
      return 0;
  }
};

/** Whether the column's default reader refuses words, as a heading holds. */
const refusesWords = (column: TableColumnProps): boolean =>
  column.type == null || column.type === "number" || column.type === "boolean";

const BOOLEANS = new Map<string, boolean>([
  ["true", true],
  ["false", false],
  ["1", true],
  ["0", false],
]);

/**
 * Reads a pasted cell with its column's parse, or as the type its column holds.
 * @returns null when the column refuses the text. By default, a number column refuses
 * text that is not a number, a text column takes anything, a boolean column takes true,
 * false, 1, and 0, and a custom column refuses all text.
 */
const parseCell = (text: string, column: TableColumnProps): TableCell | null => {
  if (column.parse != null) return column.parse(text);
  switch (column.type) {
    case "string":
      return text;
    case "boolean":
      return BOOLEANS.get(text.trim().toLowerCase()) ?? null;
    case "custom":
      return null;
    case "number":
    case undefined: {
      const value = Number(text);
      return isFinite(value) ? value : null;
    }
  }
};

/** Pasted text bound for one cell, and its value when the cell's column reads it. */
interface PastedCell {
  text: string;
  value: TableCell | null;
}

/**
 * A spreadsheet selection often carries a heading row: one with number or boolean
 * cells, none of which parse.
 */
const isHeading = (row: PastedCell[], columns: TableColumnProps[]): boolean => {
  const typed = row.filter((_, j) => refusesWords(columns[j]));
  return typed.length > 0 && typed.every(({ value }) => value === null);
};

/**
 * Reads clipboard text as a grid of cells, parsing each against the column it lands on.
 * A first row that reads as a heading is dropped, unless it is a single cell.
 * @returns null when the text holds no cells.
 */
const parseBlock = (
  text: string,
  columns: TableColumnProps[],
  from: number,
): PastedCell[][] | null => {
  const targets = columns.slice(from);
  if (targets.length === 0) return null;
  // The column count is fixed, so a block wider than the table loses its tail.
  const block = csv
    .parseBlock(text)
    .map((cells) =>
      cells
        .slice(0, targets.length)
        .map((text, j) => ({ text, value: parseCell(text, targets[j]) })),
    );
  if (block.length > 0 && !isSingle(block) && isHeading(block[0], targets))
    block.shift();
  return block.length === 0 ? null : block;
};

const isSingle = (block: unknown[][]): boolean =>
  block.length === 1 && block[0].length === 1;

/** Whether the element takes pasted text itself, as a text field does. */
const takesText = (el: EventTarget): boolean =>
  el instanceof HTMLTextAreaElement ||
  (el instanceof HTMLInputElement && el.type !== "checkbox" && el.type !== "radio");

/**
 * Whether a single value pasted on a cell that is not a text field, such as a select,
 * reaches the cell. The default string and number readers take values that such a cell
 * may not hold, so they do not.
 */
const checksText = (column: TableColumnProps): boolean =>
  column.parse != null || column.type === "boolean" || column.type === "custom";

type Move = (dims: dimensions.Dimensions, from: xy.XY, back: boolean) => xy.XY | null;

/** The cell each navigation key moves focus to. */
const MOVES: Partial<Record<string, Move>> = {
  Enter: (dims, from, back) => grid.move(dims, from, { x: 0, y: back ? -1 : 1 }),
  ArrowDown: (dims, from) => grid.move(dims, from, { x: 0, y: 1 }),
  ArrowUp: (dims, from) => grid.move(dims, from, { x: 0, y: -1 }),
};

// A dialog that a cell opens portals out of the table and keeps its own events.
const inTable = (e: SyntheticEvent): boolean =>
  e.target instanceof Node && e.currentTarget.contains(e.target);

type ColumnElement = ReactElement<TableColumnProps>;

const HANDLE_CLASS = CSS.BE("input", "table-handle");

interface RowProps extends PropsWithChildren {
  index: number;
  /** The key of the row, which its drag carries. */
  rowKey: string;
  /** @returns the index of the row with the given key, or -1. */
  indexOf: (key: string) => number;
  /** The haul type of one table's rows, so a row drops only in its own table. */
  haulType: string;
  /** Renders the preview of the row while it drags. */
  haulPreview: () => ReactNode;
  /** Omitted while the table is in preview, so the row takes no drop. */
  onMove?: (from: number, to: number) => void;
}

/** A body row that drags by its handle and takes drops of its table's other rows. */
const Row = ({
  index,
  rowKey,
  indexOf,
  haulType,
  haulPreview,
  onMove,
  children,
}: RowProps): ReactElement => {
  const [over, setOver] = useState<"above" | "below" | null>(null);
  const draggedIndex = (items: Haul.Item[]): number | null => {
    const dragged = Haul.filterByType(haulType, items).at(0);
    if (dragged == null) return null;
    const from = indexOf(String(dragged.key));
    return from === -1 ? null : from;
  };
  const { startDrag, onDragEnd, onDragOver, onDrop } = Haul.useDragAndDrop({
    type: haulType,
    key: rowKey,
    canDrop: (state) => onMove != null && Haul.canDropOfType(haulType)(state),
    onDragOver: ({ items }) => {
      const from = draggedIndex(items);
      if (from == null || from === index) setOver(null);
      else setOver(from < index ? "below" : "above");
    },
    onDrop: ({ items }) => {
      setOver(null);
      const from = draggedIndex(items);
      if (from != null && from !== index) onMove?.(from, index);
      return items;
    },
  });
  // Only the handle drags, so text in a cell still selects and drags as text. The row
  // starts the drag, so the whole row dims.
  const handleDragStart = (event: DragEvent): void => {
    if (!(event.target instanceof Element) || !event.target.closest(`.${HANDLE_CLASS}`))
      return;
    startDrag({
      event,
      items: [{ type: haulType, key: rowKey }],
      preview: haulPreview(),
    });
  };
  return (
    <tr
      className={CSS.cls(over != null && CSS.BEM("input", "table-row", over))}
      onDragStart={handleDragStart}
      onDragEnd={onDragEnd}
      onDragOver={onDragOver}
      onDragLeave={() => setOver(null)}
      onDrop={onDrop}
    >
      {children}
    </tr>
  );
};

interface BaseTableProps
  extends
    Omit<ComponentPropsWithRef<"table">, "onChange" | "children">,
    Control<TableCell[][]> {
  /** The {@link TableColumn} children in render order. */
  children: ColumnElement | ColumnElement[];
  /** Hides the add and remove buttons and makes every cell read-only. */
  preview?: boolean;
  /**
   * Labels each row's gutter cell. A row's move handle takes the place of its label and
   * carries it in its name. With no label and no handle, no gutter column renders.
   */
  rowLabel?: (index: number) => string;
  /**
   * Builds each new row: the one the add button appends, and each one a paste adds.
   * Defaults to the column default values.
   */
  createRow?: (value: TableCell[][]) => TableCell[];
  /** Names one row in the add and remove buttons ("header"). Defaults to "row". */
  resourceName?: string;
  /** Reads a cell's status, such as an error at its form path. */
  cellStatus?: (at: xy.XY) => Status.Crude | undefined;
  /**
   * Called with the index of the row the user removes. When set, a removal calls it in
   * place of onChange, so the caller can remove the row by identity.
   */
  onRemove?: (index: number) => void;
  /** Renders the preview of a dragged row. Defaults to the row's name. */
  renderHaulPreview?: (index: number) => ReactNode;
}

interface StaticTableProps {
  /**
   * Identifies a row, so the state of its cells, such as an open dialog, stays with it
   * when the rows above it change. Defaults to the index.
   */
  rowKey?: (index: number) => string;
  onMove?: undefined;
}

interface MovableTableProps {
  /** Identifies a row, so its cells, marks, and focus stay with it when it moves. */
  rowKey: (index: number) => string;
  /**
   * Called with the index of a row the user moves and the index it moves to. When set,
   * each row's gutter holds a handle. A drag drops the row in the place of the row it
   * lands on, and the up and down arrows on the handle move it by one row.
   */
  onMove?: (from: number, to: number) => void;
}

export type TableProps = BaseTableProps & (StaticTableProps | MovableTableProps);

/** A focused cell's value and text from when it took focus. */
interface Initial {
  value: TableCell | undefined;
  text: string;
}

/** Pasted text that a cell could not read. */
interface Rejection {
  /**
   * The key of the cell's row, so the mark moves with it. It is the index while the row
   * has no key: in a table with no rowKey, or in a row that a paste adds.
   */
  row: string | number;
  x: number;
  text: string;
  /** The value the cell kept in place of the text. */
  kept: TableCell | undefined;
}

/**
 * An editable grid. The value is row-major: one entry per row, holding one value per
 * column. Enter and the down arrow move down a cell, and Shift+Enter and the up arrow
 * move up. Esc puts back the value a cell held when it took focus, and a second Esc
 * leaves the cell. Pasting a tab or comma delimited block into a focused cell fills the
 * grid from that cell, adding rows as needed. A pasted cell that does not parse keeps
 * its value and is marked until an edit.
 *
 * @example
 * <Input.Table value={rows} onChange={setRows}>
 *   <Input.TableColumn name="Pre-scaled" />
 *   <Input.TableColumn name="Scaled" />
 * </Input.Table>
 */
export const Table = ({
  value,
  onChange,
  children,
  preview = false,
  rowLabel,
  rowKey,
  createRow,
  resourceName = "row",
  cellStatus,
  onRemove,
  onMove,
  renderHaulPreview,
  className,
  style,
  ...rest
}: TableProps): ReactElement => {
  // Paste and keyboard moves land on the focused cell, so the last focus is the anchor.
  const anchor = useRef<xy.XY>(xy.ZERO);
  const initial = useRef<Initial | null>(null);
  // Set once Esc puts a cell back, so the next Esc leaves the cell.
  const restored = useRef(false);
  // Set by a block paste, which commits: Esc then puts back what the paste wrote.
  const pasted = useRef(false);
  // The row that takes focus once an add or a remove renders.
  const pendingFocus = useRef<number | null>(null);
  const haulType = `${CSS.BE("input", "table-row")}-${useId()}`;
  const table = useRef<HTMLTableElement>(null);
  // The key of the row whose handle a keyboard move left.
  const movedRow = useRef<string | null>(null);
  const [rejections, setRejections] = useState<Rejection[]>([]);
  const keys = value.map((_, i) => rowKey?.(i) ?? String(i));
  const keyAt = (y: number): string | number =>
    rowKey != null && y < value.length ? keys[y] : y;
  const rowOf = ({ row }: Rejection): number =>
    typeof row === "number" ? row : keys.indexOf(row);
  const markAt = (mark: Rejection, at: xy.XY): boolean =>
    mark.x === at.x && rowOf(mark) === at.y;
  // A cell that the caller gives another value drops its mark for good.
  const marks = rejections.filter((mark) => {
    const y = rowOf(mark);
    return y !== -1 && value[y]?.[mark.x] === mark.kept;
  });
  if (marks.length !== rejections.length) setRejections(marks);
  const columns = reactElementToArray<TableColumnProps>(children).map(
    ({ props }) => props,
  );
  const dims: dimensions.Dimensions = { width: columns.length, height: value.length };
  const emptyRow = (): TableCell[] => columns.map(emptyCell);
  const movable = onMove != null && !preview;
  const hasGutter = rowLabel != null || movable;
  // Accessible row names survive without a gutter.
  const rowName = (index: number): string =>
    rowLabel?.(index) ?? (index + 1).toString();
  const hasHeader = columns.some(({ name }) => name != null);
  const columnsWidth = columns.reduce(
    (sum, { width }) => sum + (width ?? MIN_SHARED_WIDTH),
    0,
  );
  const tableStyle: CSS.VarProperties = {
    "--pluto-input-table-columns-width": `${columnsWidth}rem`,
    ...style,
  };

  const handleCellChange = (at: xy.XY, next: TableCell) => {
    if (marks.some((mark) => markAt(mark, at)))
      setRejections(marks.filter((mark) => !markAt(mark, at)));
    onChange(
      value.map((row, i) =>
        i === at.y ? row.map((cell, j) => (j === at.x ? next : cell)) : row,
      ),
    );
  };

  const handlePaste = (e: ClipboardEvent) => {
    if (preview || !inTable(e)) return;
    const block = parseBlock(
      e.clipboardData.getData("text/plain"),
      columns,
      anchor.current.x,
    );
    if (block == null) return;
    if (
      isSingle(block) &&
      (takesText(e.target) || !checksText(columns[anchor.current.x]))
    )
      return;
    e.preventDefault();
    restored.current = false;
    pasted.current = true;
    const plan = grid.plan(dims, anchor.current, block);
    const next = value.map((row) => [...row]);
    for (let row = dims.height; row < plan.dimensions.height; row++)
      next.push(createRow?.(next) ?? emptyRow());
    const rejected: Rejection[] = [];
    plan.writes.forEach(({ position: at, value: { text, value: parsed } }) => {
      if (parsed !== null) next[at.y][at.x] = parsed;
      else rejected.push({ row: keyAt(at.y), x: at.x, text, kept: next[at.y][at.x] });
    });
    const written = (mark: Rejection): boolean =>
      plan.writes.some(({ position }) => markAt(mark, position));
    setRejections([...marks.filter((mark) => !written(mark)), ...rejected]);
    onChange(next);
  };

  const statusAt = (at: xy.XY): Status.Crude | undefined => {
    const rejection = marks.find((mark) => markAt(mark, at));
    if (rejection == null) return cellStatus?.(at);
    const column = columns[at.x].name ?? "this column";
    return {
      variant: "error",
      message: `"${rejection.text}" is not a valid value for ${column}`,
    };
  };

  // A gutter header shifts the data columns over by one.
  const cellAt = ({ x, y }: xy.XY): HTMLInputElement | null =>
    table.current?.tBodies[0]?.rows[y]?.cells[
      x + (hasGutter ? 1 : 0)
    ]?.querySelector<HTMLInputElement>("input") ?? null;

  useEffect(() => {
    if (!pasted.current) return;
    pasted.current = false;
    const at = anchor.current;
    const input = cellAt(at);
    if (input == null || input !== document.activeElement) return;
    initial.current = { value: value[at.y]?.[at.x], text: input.value };
  });

  useEffect(() => {
    if (pendingFocus.current == null) return;
    const y = Math.min(pendingFocus.current, value.length - 1);
    pendingFocus.current = null;
    if (y < 0) table.current?.tFoot?.querySelector("button")?.focus();
    else cellAt({ x: 0, y })?.focus();
  }, [value.length]);

  const handleAdd = () => {
    pendingFocus.current = value.length;
    onChange([...value, createRow?.(value) ?? emptyRow()]);
  };

  const handleRemove = (index: number) => {
    pendingFocus.current = index;
    // Marks below the row move up with their rows, unless a key holds them.
    setRejections(
      marks.flatMap((mark) => {
        const y = rowOf(mark);
        if (y === index) return [];
        if (typeof mark.row === "string" || y < index) return [mark];
        return [{ ...mark, row: y - 1 }];
      }),
    );
    if (onRemove != null) onRemove(index);
    else onChange(value.filter((_, i) => i !== index));
  };

  const handleMove = (from: number, to: number) => {
    // A mark of a row that a paste added takes the row's key, so it moves with the row.
    if (marks.some(({ row }) => typeof row === "number"))
      setRejections(marks.map((mark) => ({ ...mark, row: keyAt(rowOf(mark)) })));
    onMove?.(from, to);
  };

  // A move can take the row out of the document, which drops the focus of its handle.
  useEffect(() => {
    const key = movedRow.current;
    movedRow.current = null;
    if (key == null || document.activeElement !== document.body) return;
    table.current?.tBodies[0]?.rows[keys.indexOf(key)]
      ?.querySelector<HTMLElement>(`.${HANDLE_CLASS}`)
      ?.focus();
  }, [value]);

  const handleHandleKeyDown = (e: KeyboardEvent, index: number) => {
    const step = e.key === "ArrowUp" ? -1 : e.key === "ArrowDown" ? 1 : 0;
    const to = index + step;
    if (step === 0 || to < 0 || to >= value.length) return;
    e.preventDefault();
    movedRow.current = keys[index];
    handleMove(index, to);
  };

  const haulPreview = (index: number): ReactNode =>
    renderHaulPreview?.(index) ?? (
      <Haul.Preview name={`${strings.capitalize(resourceName)} ${rowName(index)}`} />
    );

  const handleCellFocus = (e: FocusEvent<HTMLTableCellElement>, at: xy.XY) => {
    if (!inTable(e)) return;
    anchor.current = at;
    movedRow.current = null;
    if (initial.current != null || !(e.target instanceof HTMLInputElement)) return;
    initial.current = { value: value[at.y]?.[at.x], text: e.target.value };
    restored.current = false;
  };

  const handleCellBlur = (e: FocusEvent<HTMLTableCellElement>) => {
    if (inTable(e) && !e.currentTarget.contains(e.relatedTarget))
      initial.current = null;
  };

  // Runs before the cell, which blurs on an Esc that is not prevented and discards
  // any text it has not committed.
  const handleKeyDownCapture = (e: KeyboardEvent) => {
    if (e.key !== "Escape" || !inTable(e)) return;
    if (!(e.target instanceof HTMLInputElement) || initial.current == null) return;
    const { value: was, text } = initial.current;
    const at = anchor.current;
    const committed = value[at.y]?.[at.x];
    if (restored.current || (committed === was && e.target.value === text)) return;
    e.preventDefault();
    restored.current = true;
    if (committed !== was && was != null) handleCellChange(at, was);
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    if (!inTable(e)) return;
    // Enter on the add and remove buttons must reach their native activation.
    if (!(e.target instanceof HTMLInputElement)) return;
    // An Esc that put a cell back stops here, so it does not close a parent dialog.
    if (e.key === "Escape") {
      if (e.defaultPrevented) e.stopPropagation();
      return;
    }
    const move = MOVES[e.key];
    if (move == null) return;
    e.preventDefault();
    const to = move(dims, anchor.current, e.shiftKey);
    if (to != null) cellAt(to)?.focus();
  };

  return (
    <div className={CSS.BE("input", "table-scroll")}>
      <table
        ref={table}
        className={CSS.cls(
          CSS.BE("input", "table"),
          columns.every(({ width }) => width != null) && CSS.M("sized"),
          className,
        )}
        style={tableStyle}
        onPaste={handlePaste}
        onKeyDown={handleKeyDown}
        onKeyDownCapture={handleKeyDownCapture}
        // Any edit after a restore lets the next Esc restore again.
        onInput={(e) => {
          if (inTable(e)) restored.current = false;
        }}
        {...rest}
      >
        <colgroup>
          {hasGutter && <col className={CSS.BE("input", "table-gutter")} />}
          {columns.map(({ width }, i) => (
            <col key={i} style={width == null ? undefined : { width: `${width}rem` }} />
          ))}
          <col className={CSS.BE("input", "table-action")} />
        </colgroup>
        {hasHeader && value.length > 0 && (
          <thead>
            <tr>
              {hasGutter && <th className={CSS.BE("input", "table-gutter")} />}
              {columns.map(({ name }, i) => (
                <th key={i} scope="col">
                  <Text.Text level="small" weight={450} color={9}>
                    {name}
                  </Text.Text>
                </th>
              ))}
              <th />
            </tr>
          </thead>
        )}
        <tbody>
          {value.map((row, i) => {
            const cells = (
              <>
                {hasGutter && (
                  <th scope="row" className={CSS.BE("input", "table-gutter")}>
                    {movable ? (
                      <Button.Button
                        variant="text"
                        size="small"
                        textColor={9}
                        draggable
                        className={HANDLE_CLASS}
                        aria-label={`Move ${resourceName} ${rowName(i)}`}
                        tooltip={`Move ${resourceName} ${rowName(i)}`}
                        onKeyDown={(e) => handleHandleKeyDown(e, i)}
                      >
                        <Icon.Drag />
                      </Button.Button>
                    ) : (
                      rowLabel != null && (
                        <Text.Text level="small" color={9}>
                          {rowName(i)}
                        </Text.Text>
                      )
                    )}
                  </th>
                )}
                {columns.map((column, j) => {
                  const status = statusAt({ x: j, y: i });
                  // A cell's own status replaces the status of an item around the
                  // table.
                  const cellItem: ItemContextValue = {
                    status: status?.variant,
                    message: status?.message,
                    helpTextHidden: true,
                  };
                  return (
                    <td
                      key={j}
                      onFocus={(e) => handleCellFocus(e, { x: j, y: i })}
                      onBlur={handleCellBlur}
                    >
                      <ItemContext value={cellItem}>
                        {renderCell(
                          column,
                          {
                            value: row[j] ?? emptyCell(column),
                            onChange: (next) => handleCellChange({ x: j, y: i }, next),
                            preview,
                            variant: "shadow",
                            "aria-label":
                              column.name == null
                                ? rowName(i)
                                : `${column.name} ${rowName(i)}`,
                          },
                          i,
                        )}
                      </ItemContext>
                    </td>
                  );
                })}
                <td>
                  {!preview && (
                    <Button.Button
                      variant="text"
                      size="small"
                      textColor={9}
                      tooltip={`Remove ${resourceName} ${rowName(i)}`}
                      onClick={() => handleRemove(i)}
                    >
                      <Icon.Close />
                    </Button.Button>
                  )}
                </td>
              </>
            );
            // A preview keeps the row, so its inputs and focus stay when it changes.
            if (onMove == null) return <tr key={keys[i]}>{cells}</tr>;
            return (
              <Row
                key={keys[i]}
                index={i}
                rowKey={keys[i]}
                indexOf={(key) => keys.indexOf(key)}
                haulType={haulType}
                haulPreview={() => haulPreview(i)}
                onMove={movable ? handleMove : undefined}
              >
                {cells}
              </Row>
            );
          })}
        </tbody>
        {!preview && (
          <tfoot>
            <tr>
              {hasGutter && <td className={CSS.BE("input", "table-gutter")} />}
              {/* A cell under each column keeps its width while the table is empty. */}
              {columns.map((_, i) => (
                <td key={i}>
                  {i === 0 && (
                    <Button.Button
                      variant="text"
                      size="small"
                      textColor={9}
                      onClick={handleAdd}
                    >
                      <Icon.Add />
                      Add {resourceName}
                    </Button.Button>
                  )}
                </td>
              ))}
              <td />
            </tr>
          </tfoot>
        )}
      </table>
    </div>
  );
};
