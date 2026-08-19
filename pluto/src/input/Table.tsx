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
  useRef,
} from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Icon } from "@/icon";
import { Numeric } from "@/input/Numeric";
import { type Control } from "@/input/types";
import { Text } from "@/text";

/** A column of a {@link Table}. */
export interface TableColumn {
  /** The heading rendered above the column. Omitted for a lone unlabeled column. */
  name?: string;
}

export interface TableProps
  extends
    Omit<ComponentPropsWithRef<"table">, "onChange" | "children">,
    Control<number[][]> {
  /** The columns in render order. Every row holds one value per column. */
  columns: TableColumn[];
  /** Hides the add and remove buttons and makes every cell read-only. */
  preview?: boolean;
  /** Labels a row's gutter cell. Defaults to the one-based row index. */
  rowLabel?: (index: number) => string;
  /** Builds the row the add button appends. Defaults to zeros. */
  createRow?: (value: number[][]) => number[];
}

/**
 * Reads clipboard text as a grid of numbers.
 * @returns null when the text is a single cell or holds a value that is not a number,
 * leaving the paste to the focused cell.
 */
const parseBlock = (text: string): number[][] | null => {
  const block = csv
    .parseBlock(text)
    .map((row) => row.map(Number))
    // A spreadsheet selection often carries a heading row, so drop the rows holding no
    // numbers at all. A row that mixes text into numbers rejects the whole paste.
    .filter((row) => row.some((cell) => isFinite(cell)));
  if (block.length === 0) return null;
  if (block.length === 1 && block[0].length === 1) return null;
  if (block.some((row) => row.some((cell) => !isFinite(cell)))) return null;
  return block;
};

const cellID = ({ x, y }: xy.XY): string => `${y}:${x}`;

type Move = (dims: dimensions.Dimensions, from: xy.XY, back: boolean) => xy.XY | null;

/** The cell each navigation key moves focus to. */
const MOVES: Partial<Record<string, Move>> = {
  Enter: (dims, from, back) => grid.next(dims, from, back ? -1 : 1),
  ArrowDown: (dims, from) => grid.move(dims, from, { x: 0, y: 1 }),
  ArrowUp: (dims, from) => grid.move(dims, from, { x: 0, y: -1 }),
};

/**
 * An editable grid of numbers. The value is row-major: one entry per row, holding one
 * number per column. Enter and the up and down arrows move between cells. Pasting a tab
 * or comma delimited block into a focused cell fills the grid from that cell, adding
 * rows as needed.
 *
 * @example
 * <Input.Table
 *   columns={[{ name: "Pre-scaled" }, { name: "Scaled" }]}
 *   value={rows}
 *   onChange={setRows}
 * />
 */
export const Table = ({
  value,
  onChange,
  columns,
  preview = false,
  rowLabel = (index) => (index + 1).toString(),
  createRow = () => new Array<number>(columns.length).fill(0),
  className,
  ...rest
}: TableProps): ReactElement => {
  // Paste and keyboard moves land on the focused cell, so the last focus is the anchor.
  const anchor = useRef<xy.XY>(xy.ZERO);
  const cells = useRef(new Map<string, HTMLInputElement | null>());
  const dims: dimensions.Dimensions = { width: columns.length, height: value.length };

  const handleCellChange = (at: xy.XY, next: number) =>
    onChange(
      value.map((row, i) =>
        i === at.y ? row.map((cell, j) => (j === at.x ? next : cell)) : row,
      ),
    );

  const handlePaste = (e: ClipboardEvent) => {
    if (preview) return;
    const block = parseBlock(e.clipboardData.getData("text/plain"));
    if (block == null) return;
    e.preventDefault();
    const plan = grid.plan(dims, anchor.current, block);
    const next = value.map((row) => [...row]);
    for (let row = dims.height; row < plan.dimensions.height; row++)
      next.push(new Array<number>(columns.length).fill(0));
    // The column count is fixed, so a block wider than the table loses its tail.
    plan.writes.forEach(({ position: { x, y }, value }) => {
      if (x < columns.length) next[y][x] = value;
    });
    onChange(next);
  };

  const handleKeyDown = (e: KeyboardEvent) => {
    // Enter on the add and remove buttons must reach their native activation.
    if (!(e.target instanceof HTMLInputElement)) return;
    const move = MOVES[e.key];
    if (move == null) return;
    e.preventDefault();
    const to = move(dims, anchor.current, e.shiftKey);
    if (to != null) cells.current.get(cellID(to))?.focus();
  };

  return (
    <table
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
                onClick={() => onChange([...value, createRow(value)])}
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
            {columns.map(({ name }, j) => (
              <td key={j}>
                <Numeric
                  ref={(el) => {
                    cells.current.set(cellID({ x: j, y: i }), el);
                  }}
                  value={row[j] ?? 0}
                  onChange={(next) => handleCellChange({ x: j, y: i }, next)}
                  onFocus={() => (anchor.current = { x: j, y: i })}
                  preview={preview}
                  showDragHandle={false}
                  aria-label={name == null ? rowLabel(i) : `${name} ${rowLabel(i)}`}
                />
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
