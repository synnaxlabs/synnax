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
  type KeyboardEvent,
  type ReactElement,
  type ReactNode,
  useRef,
} from "react";

import { Button } from "@/button";
import { type RenderProp } from "@/component/renderProp";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Numeric } from "@/input/Numeric";
import { Text as InputText } from "@/input/Text";
import { type Control } from "@/input/types";
import { Text } from "@/text";
import { reactElementToArray } from "@/util/children";

/** The value one {@link Table} cell holds. */
export type TableCell = string | number;

/** The props a {@link TableColumn} cell renderer receives. */
export interface TableCellProps<V extends TableCell = TableCell> extends Control<V> {
  /** Whether the cell is read-only. */
  preview?: boolean;
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
 * @returns null when a numeric column is given text that is not a number. A text
 * column takes anything.
 */
const parseCell = (text: string, column: TableColumnProps): TableCell | null => {
  if (column.type === "string") return text;
  const value = Number(text);
  return isFinite(value) ? value : null;
};

const isComplete = (row: (TableCell | null)[]): row is TableCell[] =>
  row.every((cell) => cell != null);

/**
 * Reads clipboard text as a grid of cell values, parsing each against the column it
 * lands on.
 * @returns null when the text is a single cell or holds a value its column rejects,
 * leaving the paste to the focused cell.
 */
const parseBlock = (
  text: string,
  columns: TableColumnProps[],
  from: number,
): TableCell[][] | null => {
  if (from >= columns.length) return null;
  const block: TableCell[][] = [];
  for (const [i, cells] of csv.parseBlock(text).entries()) {
    // The column count is fixed, so a block wider than the table loses its tail before
    // it can reject anything.
    const row = cells
      .slice(0, columns.length - from)
      .map((cell, j) => parseCell(cell, columns[from + j]));
    if (!isComplete(row)) {
      // A spreadsheet selection often carries a heading row, so a first row that does
      // not parse is dropped. A later one rejects the whole paste.
      if (i === 0) continue;
      return null;
    }
    block.push(row);
  }
  if (block.length === 0) return null;
  if (block.length === 1 && block[0].length === 1) return null;
  return block;
};

type Move = (dims: dimensions.Dimensions, from: xy.XY, back: boolean) => xy.XY | null;

/** The cell each navigation key moves focus to. */
const MOVES: Partial<Record<string, Move>> = {
  Enter: (dims, from, back) => grid.next(dims, from, back ? -1 : 1),
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
  /** Labels a row's gutter cell. Defaults to the one-based row index. */
  rowLabel?: (index: number) => string;
  /** Builds the row the add button appends. Defaults to the column default values. */
  createRow?: (value: TableCell[][]) => TableCell[];
}

/**
 * An editable grid. The value is row-major: one entry per row, holding one value per
 * column. Enter and the up and down arrows move between cells. Pasting a tab or comma
 * delimited block into a focused cell fills the grid from that cell, adding rows as
 * needed.
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
  rowLabel = (index) => (index + 1).toString(),
  createRow,
  className,
  ...rest
}: TableProps): ReactElement => {
  // Paste and keyboard moves land on the focused cell, so the last focus is the anchor.
  const anchor = useRef<xy.XY>(xy.ZERO);
  const table = useRef<HTMLTableElement>(null);
  const columns = reactElementToArray<TableColumnProps>(children).map(
    ({ props }) => props,
  );
  const dims: dimensions.Dimensions = { width: columns.length, height: value.length };
  const emptyRow = (): TableCell[] => columns.map(emptyCell);

  const handleCellChange = (at: xy.XY, next: TableCell) =>
    onChange(
      value.map((row, i) =>
        i === at.y ? row.map((cell, j) => (j === at.x ? next : cell)) : row,
      ),
    );

  const handlePaste = (e: ClipboardEvent) => {
    if (preview) return;
    const block = parseBlock(
      e.clipboardData.getData("text/plain"),
      columns,
      anchor.current.x,
    );
    if (block == null) return;
    e.preventDefault();
    const plan = grid.plan(dims, anchor.current, block);
    const next = value.map((row) => [...row]);
    for (let row = dims.height; row < plan.dimensions.height; row++)
      next.push(emptyRow());
    plan.writes.forEach(({ position: { x, y }, value }) => (next[y][x] = value));
    onChange(next);
  };

  // The row's first cell is its gutter header, so the data columns start at one.
  const cellAt = ({ x, y }: xy.XY): HTMLInputElement | null =>
    table.current?.tBodies[0]?.rows[y]?.cells[x + 1]?.querySelector<HTMLInputElement>(
      "input",
    ) ?? null;

  const handleKeyDown = (e: KeyboardEvent) => {
    // Enter on the add and remove buttons must reach their native activation.
    if (!(e.target instanceof HTMLInputElement)) return;
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
      {...rest}
    >
      <thead>
        <tr>
          <th />
          {columns.map(({ name }, i) => (
            <th key={i} scope="col">
              <Text.Text level="small" color={9}>
                {name}
              </Text.Text>
            </th>
          ))}
          <th>
            {!preview && (
              <Button.Button
                variant="filled"
                size="small"
                tooltip="Add row"
                onClick={() => onChange([...value, createRow?.(value) ?? emptyRow()])}
              >
                <Icon.Add />
              </Button.Button>
            )}
          </th>
        </tr>
      </thead>
      <tbody>
        {value.map((row, i) => (
          <tr key={i} className={CSS.M("reveals")}>
            <th scope="row">
              <Text.Text level="small" color={9}>
                {rowLabel(i)}
              </Text.Text>
            </th>
            {columns.map((column, j) => (
              <td key={j} onFocus={() => (anchor.current = { x: j, y: i })}>
                {renderCell(column, {
                  value: row[j] ?? emptyCell(column),
                  onChange: (next) => handleCellChange({ x: j, y: i }, next),
                  preview,
                  "aria-label":
                    column.name == null ? rowLabel(i) : `${column.name} ${rowLabel(i)}`,
                })}
              </td>
            ))}
            <td>
              {!preview && (
                <Button.Button
                  variant="text"
                  size="small"
                  reveal
                  tooltip={`Remove row ${rowLabel(i)}`}
                  onClick={() => onChange(value.filter((_, j) => j !== i))}
                >
                  <Icon.Close />
                </Button.Button>
              )}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};
