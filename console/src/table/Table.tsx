// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Table as Core, TableCell } from "@synnaxlabs/pluto";
import { type UnknownRecord, type xy } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { type Layout } from "@/layout";
import { useSelect } from "@/table/selectors";
import { setCellProps, setSelected } from "@/table/slice";

export interface CellProps extends Omit<TableCell.CellProps, "onChange"> {
  type: string;
  pos: xy.XY;
  onChange: (pos: xy.XY, props: UnknownRecord) => void;
}

export const Cell = ({ type, props, onChange, onSelect }: CellProps): ReactElement => {
  const Spec = TableCell.REGISTRY[type];
  return <Spec.Cell {...props} onSelect={onSelect} onChange={onChange} />;
};

export const Table: Layout.Renderer = ({ layoutKey }) => {
  const table = useSelect(layoutKey);
  const d = useDispatch();
  const selectedStrings = table.selected.map((p) => `${p.x}-${p.y}`);

  const handleChange = (pos: xy.XY, props: UnknownRecord): void => {
    d(setCellProps({ key: layoutKey, positions: [pos], props: [props] }));
  };

  const handleSelect = (pos: xy.XY): void => {
    d(setSelected({ key: layoutKey, selected: [pos] }));
  };

  return (
    <Core.Table numColumns={table.dimensions.width}>
      {table.rows.map((r, y) => (
        <Core.TR key={y}>
          {r.cells.map((c, x) => {
            const strKey = `${y}-${x}`;
            const pos = { x, y };
            return (
              <Cell
                key={strKey}
                pos={pos}
                type={c.type}
                props={c.props}
                onChange={(p) => handleChange(pos, p)}
                selected={selectedStrings.includes(strKey)}
                onSelect={() => handleSelect(pos)}
              />
            );
          })}
        </Core.TR>
      ))}
    </Core.Table>
  );
};
