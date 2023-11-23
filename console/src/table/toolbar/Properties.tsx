// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Align, Status, TableCell } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { useSelectSelectedCells } from "@/table/selectors";
import { setCellProps } from "@/table/slice";

export interface PropertiesProps {
  layoutKey: string;
}

export const Properties = ({ layoutKey }: PropertiesProps): ReactElement => {
  const cells = useSelectSelectedCells(layoutKey);
  const d = useDispatch();

  if (cells.length === 0)
    return (
      <Status.Text.Centered variant="disabled" hideIcon>
        Select a table cell to configure its properties.
      </Status.Text.Centered>
    );

  const f = cells[0];
  const Spec = TableCell.REGISTRY[f.type];

  const handleChange = (p: UnknownRecord): void => {
    d(
      setCellProps({
        positions: [f.pos],
        props: [p],
      }),
    );
  };

  return (
    <Align.Space size="small">
      <Spec.Form value={f.props} onChange={handleChange} />
    </Align.Space>
  );
};
