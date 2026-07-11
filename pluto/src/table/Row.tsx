// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { box, dimensions, type record, xy } from "@synnaxlabs/x";
import { memo, type ReactElement, useCallback, useMemo } from "react";

import { CSS } from "@/css";
import { Select } from "@/select/base";
import { Cell } from "@/table/cells";
import { Indicator } from "@/table/Indicator";
import { useDispatch, useSelectCell } from "@/table/queries";

export interface RowProps {
  index: number;
  size: number;
  position: number;
  resourceKey: table.Key;
  cells: string[];
  columns: number[];
  editable: boolean;
  showIndicator?: boolean;
  onResize: (size: number, index: number) => void;
  onSelect: (index: number, ev: React.MouseEvent) => void;
  onCellSelect: (cellKey: string, ev: MouseEvent) => void;
}

export const Row = memo(
  ({
    index,
    size,
    position,
    resourceKey,
    cells,
    columns,
    editable,
    showIndicator = true,
    onResize,
    onSelect,
    onCellSelect,
  }: RowProps): ReactElement => {
    let xCursor = showIndicator ? 4.5 * 6 : 0;
    return (
      <tr className={CSS(CSS.BE("table", "row"))}>
        {showIndicator && (
          <Indicator
            direction="y"
            index={index}
            value={size}
            editable={editable}
            onChange={onResize}
            onSelect={onSelect}
          />
        )}
        {cells.map((cellKey, i) => {
          const xPos = xCursor;
          xCursor += columns[i];
          return (
            <VariantCell
              key={cellKey}
              resourceKey={resourceKey}
              cellKey={cellKey}
              x={xPos}
              y={position}
              width={columns[i]}
              height={size}
              editable={editable}
              onSelect={onCellSelect}
            />
          );
        })}
      </tr>
    );
  },
);
Row.displayName = "Row";

interface VariantCellProps {
  resourceKey: table.Key;
  cellKey: string;
  x: number;
  y: number;
  width: number;
  height: number;
  editable: boolean;
  onSelect: (cellKey: string, ev: MouseEvent) => void;
}

// VariantCell is the bridge between the connected Table and the per-variant
// cell components in @/table/cells. The variant component renders its own
// <td> (via the Cell primitive); VariantCell wires it to flux state and a
// dispatch-backed onChange handler. It takes x/y/width/height as primitives
// (not a Box) so the memo barrier compares stable scalars; the Box is
// constructed once per geometry change inside. Selection is read via
// Selection.useItemState so a cell re-renders only when its own selection flips,
// not on every selection event elsewhere in the table.
const VariantCell = memo(
  ({
    resourceKey,
    cellKey,
    x,
    y,
    width,
    height,
    editable,
    onSelect,
  }: VariantCellProps): ReactElement | null => {
    const cell = useSelectCell({ key: resourceKey, cellKey });
    const { selected } = Select.useItemState(cellKey);
    const { dispatch } = useDispatch();
    const b = useMemo(
      () => box.construct(xy.construct({ x, y }), dimensions.construct(width, height)),
      [x, y, width, height],
    );
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
    const Spec = Cell.REGISTRY[cell.variant];
    return (
      <Spec.Cell
        cellKey={cellKey}
        box={b}
        selected={selected}
        editable={editable}
        onSelect={onSelect}
        onChange={handleChange}
        {...cell.props}
      />
    );
  },
);
VariantCell.displayName = "VariantCell";
