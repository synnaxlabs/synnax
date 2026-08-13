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
import { box, direction } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useMemo, useRef } from "react";

import { CSS } from "@/css";
import { Cursor } from "@/cursor";
import { useSyncedRef } from "@/hooks";
import { Menu } from "@/menu";
import { Text } from "@/text";
import { stopPropagation } from "@/util/event";

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";

// getCellColumn maps a 0-based column index to a spreadsheet-style letter
// ("A", "B", "C", ...). Defined here so consumers building UI chrome (e.g.,
// breadcrumb labels in a toolbar) can label cells using the same convention
// the table renders.
export const getCellColumn = (index: number): string => ALPHABET[index];

export interface ColumnIndicatorsProps {
  columns: number[];
  rows: table.Row[];
  selected: string[];
  editable: boolean;
  onSelect: (index: number, ev: React.MouseEvent) => void;
  onSelectAll: () => void;
  onResize: (size: number, index: number) => void;
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
  }: ColumnIndicatorsProps): ReactElement => {
    // cellToCol indexes the rows once per layout change so selectedCols can
    // resolve each selected cell to its column in O(1).
    const cellToCol = useMemo(() => {
      const map = new Map<string, number>();
      for (const row of rows) row.cells.forEach((k, i) => map.set(k, i));
      return map;
    }, [rows]);
    const selectedCols = useMemo(() => {
      const cols = new Set<number>();
      for (const k of selected) {
        const idx = cellToCol.get(k);
        if (idx !== undefined) cols.add(idx);
      }
      return cols;
    }, [cellToCol, selected]);
    const totalCells = useMemo(
      () => rows.reduce((n, r) => n + r.cells.length, 0),
      [rows],
    );
    const allSelected = totalCells > 0 && selected.length >= totalCells;
    return (
      <tr className={CSS(CSS.BE("table", "row"), CSS.BE("table", "col-resizer"))}>
        <td
          className={CSS(CSS.BE("table", "select-all"), CSS.selected(allSelected))}
          onClick={onSelectAll}
          onContextMenu={onSelectAll}
        />
        {columns.map((size, i) => (
          <Indicator
            key={i}
            direction="x"
            index={i}
            value={size}
            selected={selectedCols.has(i)}
            editable={editable}
            onChange={onResize}
            onSelect={onSelect}
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
  value: number;
  editable: boolean;
  selected?: boolean;
  onChange: (size: number, index: number) => void;
  onSelect: (index: number, ev: React.MouseEvent) => void;
}

export const Indicator = ({
  direction: dir,
  index,
  value,
  editable,
  selected = false,
  onChange,
  onSelect,
}: IndicatorProps): ReactElement => {
  const valueRef = useSyncedRef(value);
  const sizeRef = useRef(value);
  const onDragStart = Cursor.useDrag({
    onStart: useCallback(() => {
      sizeRef.current = valueRef.current;
    }, []),
    onMove: useCallback(
      (b: box.Box) => onChange(sizeRef.current + box.dim(b, dir, true), index),
      [onChange, index, dir],
    ),
  });
  const style = useMemo(() => ({ [direction.dimension(dir)]: value }), [dir, value]);
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
      style={style}
      onClick={(e) => onSelect(index, e)}
      onContextMenu={(e) => onSelect(index, e)}
    >
      <Text.Text full="x" justify="center" align="center" square={false}>
        {dir === "x" ? ALPHABET[index] : index + 1}
      </Text.Text>
      {editable && (
        <button
          aria-label={dir === "x" ? "Resize column" : "Resize row"}
          tabIndex={-1}
          className={Cursor.DRAG_CLASS}
          onClick={stopPropagation}
          onPointerDown={onDragStart}
        />
      )}
    </td>
  );
};
