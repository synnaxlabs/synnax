// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/table/Table.css";

import { type table } from "@synnaxlabs/client";
import { Cursor } from "@synnaxlabs/lyra/cursor";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { box, direction, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, useRef } from "react";

import { CSS } from "@/css";
import { transform } from "@/table/shift";

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

/** Pixel size of an indicator strip, mirroring --pluto-table-indicator-size. */
export const INDICATOR_SIZE = 4.5 * 6;

// getCellColumn maps a 0-based column index to a spreadsheet-style label ("A", "B",
// ... "Z", "AA", "AB", ...). Exported so consumers labeling cells outside the table
// (toolbar breadcrumbs, menu items) use the same convention the table renders.
export const getCellColumn = (index: number): string => {
  let label = "";
  for (let i = index; i >= 0; i = Math.floor(i / ALPHABET.length) - 1)
    label = ALPHABET[i % ALPHABET.length] + label;
  return label;
};

export interface ColumnIndicatorsProps extends Pick<IndicatorProps, "onMovePress"> {
  columns: table.Column[];
  rows: table.Row[];
  selected: string[];
  editable: boolean;
  onSelect: (index: number, ev: React.MouseEvent) => void;
  onSelectAll: () => void;
  onResize: (size: number, key: string) => void;
  /** How far each column slides while a move previews where it lands. */
  shifts?: number[];
}

export const ColumnIndicators = memo(
  ({
    columns,
    rows,
    selected,
    editable,
    onSelect,
    onSelectAll,
    onResize,
    onMovePress,
    shifts,
  }: ColumnIndicatorsProps): ReactElement => {
    // cellToCol indexes the rows once per layout change so selectedCols can
    // resolve each selected cell to its column in O(1).
    const map = new Map<string, number>();
    for (const row of rows) row.cells.forEach((k, i) => map.set(k, i));
    const cellToCol = map;
    const cols = new Set<number>();
    for (const k of selected) {
      const idx = cellToCol.get(k);
      if (idx !== undefined) cols.add(idx);
    }
    const selectedCols = cols;
    const totalCells = rows.reduce((n, r) => n + r.cells.length, 0);
    const allSelected = totalCells > 0 && selected.length >= totalCells;
    return (
      <tr className={CSS.cls(CSS.BE("table", "row"), CSS.BE("table", "col-resizer"))}>
        <td
          className={CSS.cls(CSS.BE("table", "select-all"), CSS.selected(allSelected))}
          onClick={onSelectAll}
          onContextMenu={onSelectAll}
        />
        {columns.map((col, i) => (
          <Indicator
            key={col.key}
            direction="x"
            index={i}
            itemKey={col.key}
            value={col.size}
            selected={selectedCols.has(i)}
            editable={editable}
            onChange={onResize}
            onSelect={onSelect}
            onMovePress={onMovePress}
            shift={shifts?.[i]}
          />
        ))}
      </tr>
    );
  },
);
ColumnIndicators.displayName = "ColumnIndicators";

export interface IndicatorProps {
  direction: direction.Direction;
  index: number;
  /** Key of the row or column. */
  itemKey: string;
  value: number;
  editable: boolean;
  selected?: boolean;
  onChange: (size: number, key: string) => void;
  onSelect: (index: number, ev: React.MouseEvent) => void;
  /**
   * Handles a press on the label, which moves the row or column once the pointer
   * travels. The label does not move anything when absent.
   */
  onMovePress?: (key: string, ev: React.PointerEvent) => void;
  /** How far the row or column slides while a move previews where it lands. */
  shift?: number;
}

export const Indicator = ({
  direction: dir,
  index,
  itemKey,
  value,
  editable,
  selected = false,
  onChange,
  onSelect,
  onMovePress,
  shift = 0,
}: IndicatorProps): ReactElement => {
  const valueRef = useSyncedRef(value);
  const sizeRef = useRef(value);
  const onDragStart = Cursor.useDrag({
    onStart: () => {
      sizeRef.current = valueRef.current;
    },
    onMove: (b: box.Box) => onChange(sizeRef.current + box.dim(b, dir, true), itemKey),
  });
  const dim = direction.dimension(dir);
  const style = { [dim]: value, transform: transform(xy.construct(dir, shift)) };
  const label = dir === "x" ? getCellColumn(index) : index + 1;
  return (
    <td
      id={`resizer-${dir}-${itemKey}`}
      className={CSS.cls(
        CSS.BE("table", "resizer"),
        CSS.dir(dir),
        CSS.selected(selected),
        Menu.CONTEXT_TARGET,
        selected && Menu.CONTEXT_SELECTED,
      )}
      style={style}
      onClick={(e) => onSelect(index, e)}
      onContextMenu={(e) => onSelect(index, e)}
    >
      <Text.Text
        className={CSS.cls(
          CSS.BE("table", "label"),
          onMovePress != null && Cursor.DRAG_CLASS,
        )}
        justify="center"
        align="center"
        square={false}
        onPointerDown={onMovePress == null ? undefined : (e) => onMovePress(itemKey, e)}
      >
        {label}
      </Text.Text>
      {editable && (
        <button
          aria-label={`Resize ${dir === "x" ? "column" : "row"} ${label}`}
          tabIndex={-1}
          className={Cursor.DRAG_CLASS}
          onClick={stopPropagation}
          onPointerDown={onDragStart}
        />
      )}
    </td>
  );
};
