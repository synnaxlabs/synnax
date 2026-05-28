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
import { memo, type ReactElement, useCallback } from "react";

import { CSS } from "@/css";
import { CELLS } from "@/table/cells/registry";
import { Indicator } from "@/table/Indicator";
import { useDispatch, useSelectCell } from "@/table/queries";

export interface RowProps {
  index: number;
  size: number;
  position: number;
  resourceKey: table.Key;
  cells: string[];
  columns: number[];
  selected: string[];
  editable: boolean;
  onResize: (size: number, index: number) => void;
  onSelect: (index: number) => void;
  onCellSelect: (cellKey: string, ev: MouseEvent) => void;
}

export const Row = ({
  index,
  size,
  position,
  resourceKey,
  cells,
  columns,
  selected,
  editable,
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
        editable={editable}
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
// <td> (via the Cell primitive); VariantCell wires it to flux state and a
// dispatch-backed onChange handler.
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
