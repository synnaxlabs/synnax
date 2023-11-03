// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Table as Core, TableElement } from "@synnaxlabs/pluto";
import { type UnknownRecord, type xy } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { type Layout } from "@/layout";
import { useSelect } from "@/table/selectors";
import { setCellProps } from "@/table/slice";

export interface CellProps extends Omit<TableElement.ElementProps, "onChange"> {
  type: string;
  pos: xy.XY;
  onChange: (pos: xy.XY, props: UnknownRecord) => void;
}

export const Cell = ({ type, props, onChange }: CellProps): ReactElement => {
  const Spec = TableElement.REGISTRY[type];
  return <Spec.Element {...props} onChange={onChange} />;
};

export const Table: Layout.Renderer = ({ layoutKey }) => {
  const table = useSelect(layoutKey);
  const d = useDispatch();

  const handleChange = (pos: xy.XY, props: UnknownRecord): void => {
    d(setCellProps({ positions: [pos], props: [props] }));
  };

  return (
    <Core.Table numColumns={table.dimensions.width}>
      {table.rows.map((r, y) => (
        <Core.TR key={y}>
          {r.cells.map((c, x) => (
            <Cell
              key={`${y}-${x}`}
              pos={{ x, y }}
              type={c.type}
              props={c.props}
              onChange={handleChange}
            />
          ))}
        </Core.TR>
      ))}
    </Core.Table>
  );
};
