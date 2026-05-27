// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type table } from "@synnaxlabs/client";
import { box, direction } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useRef } from "react";

import { CSS } from "@/css";
import { useSyncedRef } from "@/hooks";
import { useCursorDrag } from "@/hooks/useCursorDrag";
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
  onSelect: (index: number) => void;
  onResize: (size: number, index: number) => void;
}

export const ColumnIndicators = ({
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

export interface IndicatorProps {
  direction: direction.Direction;
  index: number;
  value: number;
  position: number;
  selected?: boolean;
  onChange: (size: number, index: number) => void;
  onSelect: (index: number) => void;
}

export const Indicator = ({
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
