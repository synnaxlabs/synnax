// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/input/Table.css";

import { csv, type dimensions, grid, xy } from "@synnaxlabs/x";
import {
  type ClipboardEvent,
  type ComponentPropsWithRef,
  type FocusEvent,
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useEffect,
  useRef,
  useState,
} from "react";

import { Button } from "@/button";
import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Numeric } from "@/input/Numeric";
import { ItemContext, type ItemContextValue } from "@/input/scope";
import { Text as InputText } from "@/input/Text";
import { type Control, type Variant } from "@/input/types";
import { type Status } from "@/status";
import { Text } from "@/text";
import { reactElementToArray } from "@/util/children";

/** The value one {@link Table} cell holds. */
export type TableCell = string | number;

/** The props a {@link TableColumn} cell renderer receives. */
export interface TableCellProps<V extends TableCell = TableCell> extends Control<V> {
  /** Whether the cell is read-only. */
  preview?: boolean;
  /** The edit-in-place chassis every cell shares. Spread it onto the input. */
  variant?: Variant;
  "aria-label": string;
}

interface BaseTableColumnProps {
  /** The heading rendered above the column. Omitted for a lone unlabeled column. */
  name?: string;
}

/**
 * The column's type decides what a new row holds, how pasted text is read, and which
 * input a cell defaults to.
 */
export type TableColumnProps =
  | (BaseTableColumnProps & {
      type?: "number";
      /** Renders the cell. Defaults to a numeric input. */
      children?: RenderProp<TableCellProps<number>>;
    })
  | (BaseTableColumnProps & {
      type: "string";
      /** Renders the cell. Defaults to a text input. */
      children?: RenderProp<TableCellProps<string>>;
    });

/**
 * Declares one column of a {@link Table}. It renders nothing itself: the table reads
 * its props to build the header and every row's cell.
 *
 * @example <Input.TableColumn name="Pre-scaled" />
 * @example <Input.TableColumn name="Label" type="string" />
 */
export const TableColumn = (_: TableColumnProps): null => null;

const renderCell = (column: TableColumnProps, props: TableCellProps): ReactNode => {
  if (column.type === "string") {
    const cell = { ...props, value: String(props.value) };
    if (column.children != null) return column.children(cell);
    return <InputText {...cell} />;
  }
  const cell = { ...props, value: Number(props.value) };
  if (column.children != null) return column.children(cell);
  return <Numeric {...cell} showDragHandle={false} />;
};

/** The value a new row holds in this column. */
const emptyCell = (column: TableColumnProps): TableCell =>
  column.type === "string" ? "" : 0;

/**
 * Reads a pasted cell as the type its column holds.
 * @returns null when a number column is given text that is not a number. A text column
 * takes anything.
 */
const parseCell = (text: string, column: TableColumnProps): TableCell | null => {
  if (column.type === "string") return text;
  const value = Number(text);
  return isFinite(value) ? value : null;
};

/** Pasted text bound for one cell, and its value when the cell's column reads it. */
interface PastedCell {
  text: string;
  value: TableCell | null;
}

/**
 * A spreadsheet selection often carries a heading row: one with number cells, none of
 * which parse.
 */
const isHeading = (row: PastedCell[], columns: TableColumnProps[]): boolean => {
  const numbers = row.filter((_, j) => columns[j].type !== "string");
  return numbers.length > 0 && numbers.every(({ value }) => value == null);
};

/**
 * Reads clipboard text as a grid of cells, parsing each against the column it lands on.
 * A first row that reads as a heading is dropped.
 * @returns null when the text is a single cell, leaving the paste to the focused cell.
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
  if (block.length > 0 && isHeading(block[0], targets)) block.shift();
  if (block.length === 0) return null;
  if (block.length === 1 && block[0].length === 1) return null;
  return block;
};

type Move = (dims: dimensions.Dimensions, from: xy.XY, back: boolean) => xy.XY | null;

/** The cell each navigation key moves focus to. */
const MOVES: Partial<Record<string, Move>> = {
  Enter: (dims, from, back) => grid.move(dims, from, { x: 0, y: back ? -1 : 1 }),
  ArrowDown: (dims, from) => grid.move(dims, from, { x: 0, y: 1 }),
  ArrowUp: (dims, from) => grid.move(dims, from, { x: 0, y: -1 }),
};

type ColumnElement = ReactElement<TableColumnProps>;

export interface TableProps
  extends
    Omit<ComponentPropsWithRef<"table">, "onChange" | "children">,
    Control<TableCell[][]> {
  /** The {@link TableColumn} children in render order. */
  children: ColumnElement | ColumnElement[];
  /** Hides the add and remove buttons and makes every cell read-only. */
  preview?: boolean;
  /** Labels a row's gutter cell. Omitted, no gutter column renders. */
  rowLabel?: (index: number) => string;
  /** Builds the row the add button appends. Defaults to the column default values. */
  createRow?: (value: TableCell[][]) => TableCell[];
  /** Names one row in the add and remove buttons ("header"). Defaults to "row". */
  resourceName?: string;
  /** Reads a cell's status, such as an error at its form path. */
  cellStatus?: (at: xy.XY) => Status.Crude | undefined;
}

/** A focused cell's value and text from when it took focus. */
interface Initial {
  value: TableCell | undefined;
  text: string;
}

/** Pasted text that a cell could not read. */
interface Rejection {
  at: xy.XY;
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
  createRow,
  resourceName = "row",
  cellStatus,
  className,
  ...rest
}: TableProps): ReactElement => {
  // Paste and keyboard moves land on the focused cell, so the last focus is the anchor.
  const anchor = useRef<xy.XY>(xy.ZERO);
  const initial = useRef<Initial | null>(null);
  // Set once Esc puts a cell back, so the next Esc leaves the cell.
  const restored = useRef(false);
  // The row that takes focus once an add or a remove renders.
  const pendingFocus = useRef<number | null>(null);
  const table = useRef<HTMLTableElement>(null);
  const [rejections, setRejections] = useState<Rejection[]>([]);
  // A cell that the caller gives another value drops its mark for good.
  const marks = rejections.filter(({ at, kept }) => value[at.y]?.[at.x] === kept);
  if (marks.length !== rejections.length) setRejections(marks);
  const columns = reactElementToArray<TableColumnProps>(children).map(
    ({ props }) => props,
  );
  const dims: dimensions.Dimensions = { width: columns.length, height: value.length };
  const emptyRow = (): TableCell[] => columns.map(emptyCell);
  const hasGutter = rowLabel != null;
  // Accessible row names survive without a gutter.
  const rowName = (index: number): string =>
    rowLabel?.(index) ?? (index + 1).toString();
  const hasHeader = columns.some(({ name }) => name != null);

  const handleCellChange = (at: xy.XY, next: TableCell) => {
    if (marks.some((mark) => xy.equals(mark.at, at)))
      setRejections(marks.filter((mark) => !xy.equals(mark.at, at)));
    onChange(
      value.map((row, i) =>
        i === at.y ? row.map((cell, j) => (j === at.x ? next : cell)) : row,
      ),
    );
  };

  const handlePaste = (e: ClipboardEvent) => {
    if (preview) return;
    const block = parseBlock(
      e.clipboardData.getData("text/plain"),
      columns,
      anchor.current.x,
    );
    if (block == null) return;
    e.preventDefault();
    restored.current = false;
    const plan = grid.plan(dims, anchor.current, block);
    const next = value.map((row) => [...row]);
    for (let row = dims.height; row < plan.dimensions.height; row++)
      next.push(emptyRow());
    const rejected: Rejection[] = [];
    plan.writes.forEach(({ position: at, value: { text, value: parsed } }) => {
      if (parsed != null) next[at.y][at.x] = parsed;
      else rejected.push({ at, text, kept: next[at.y][at.x] });
    });
    const written = (at: xy.XY): boolean =>
      plan.writes.some(({ position }) => xy.equals(position, at));
    setRejections([...marks.filter(({ at }) => !written(at)), ...rejected]);
    onChange(next);
  };

  const statusAt = (at: xy.XY): Status.Crude | undefined => {
    const rejection = marks.find((mark) => xy.equals(mark.at, at));
    if (rejection == null) return cellStatus?.(at);
    return { variant: "error", message: `"${rejection.text}" is not a number` };
  };

  // A gutter header shifts the data columns over by one.
  const cellAt = ({ x, y }: xy.XY): HTMLInputElement | null =>
    table.current?.tBodies[0]?.rows[y]?.cells[
      x + (hasGutter ? 1 : 0)
    ]?.querySelector<HTMLInputElement>("input") ?? null;

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
    // Marks below the row move up with their rows.
    setRejections(
      marks.flatMap((mark) => {
        if (mark.at.y < index) return [mark];
        if (mark.at.y === index) return [];
        return [{ ...mark, at: { x: mark.at.x, y: mark.at.y - 1 } }];
      }),
    );
    onChange(value.filter((_, i) => i !== index));
  };

  const handleCellFocus = (e: FocusEvent<HTMLTableCellElement>, at: xy.XY) => {
    anchor.current = at;
    if (initial.current != null || !(e.target instanceof HTMLInputElement)) return;
    initial.current = { value: value[at.y]?.[at.x], text: e.target.value };
    restored.current = false;
  };

  const handleCellBlur = (e: FocusEvent<HTMLTableCellElement>) => {
    if (!e.currentTarget.contains(e.relatedTarget)) initial.current = null;
  };

  // Runs before the cell, which blurs on an Esc that is not prevented and discards
  // any text it has not committed.
  const handleKeyDownCapture = (e: KeyboardEvent) => {
    if (e.key !== "Escape") return;
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
    <table
      ref={table}
      className={CSS.cls(CSS.BE("input", "table"), className)}
      onPaste={handlePaste}
      onKeyDown={handleKeyDown}
      onKeyDownCapture={handleKeyDownCapture}
      // Any edit after a restore lets the next Esc restore again.
      onInput={() => (restored.current = false)}
      {...rest}
    >
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
        {value.map((row, i) => (
          <tr key={i}>
            {hasGutter && (
              <th scope="row" className={CSS.BE("input", "table-gutter")}>
                <Text.Text level="small" color={9}>
                  {rowName(i)}
                </Text.Text>
              </th>
            )}
            {columns.map((column, j) => {
              const status = statusAt({ x: j, y: i });
              // A cell's own status replaces the status of an item around the table.
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
                    {renderCell(column, {
                      value: row[j] ?? emptyCell(column),
                      onChange: (next) => handleCellChange({ x: j, y: i }, next),
                      preview,
                      variant: "shadow",
                      "aria-label":
                        column.name == null
                          ? rowName(i)
                          : `${column.name} ${rowName(i)}`,
                    })}
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
          </tr>
        ))}
      </tbody>
      {!preview && (
        <tfoot>
          <tr>
            {hasGutter && <td className={CSS.BE("input", "table-gutter")} />}
            <td colSpan={columns.length + 1}>
              <Button.Button
                variant="text"
                size="small"
                textColor={9}
                onClick={handleAdd}
              >
                <Icon.Add />
                Add {resourceName}
              </Button.Button>
            </td>
          </tr>
        </tfoot>
      )}
    </table>
  );
};
