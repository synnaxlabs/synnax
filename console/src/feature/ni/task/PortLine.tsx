// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { List } from "@synnaxlabs/lyra/list";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type DigitalChannel } from "@/feature/ni/task/types";

/** The port and line columns of a digital task's record table. */
export const PORT_LINE_COLUMNS: List.Column[] = [
  { name: "Port", width: "18rem", field: true },
  { name: "Line", width: "18rem", field: true },
];

interface NumberCellProps {
  path: deep.Target<number>;
  label: string;
}

const NumberCell = ({ path, label }: NumberCellProps): ReactElement => (
  <List.Cell>
    <Form.NumericField
      path={path}
      showLabel={false}
      showHelpText={false}
      inputProps={{ variant: "shadow", showDragHandle: false, "aria-label": label }}
    />
  </List.Cell>
);

export interface PortLineCellsProps<C extends DigitalChannel> {
  path: deep.Target<C>;
}

/** The port and line cells of a digital channel. */
export const PortLineCells = <C extends DigitalChannel>({
  path,
}: PortLineCellsProps<C>): ReactElement => {
  const { port, line } = deep.widen<DigitalChannel>(path);
  return (
    <>
      <NumberCell path={port} label="Port" />
      <NumberCell path={line} label="Line" />
    </>
  );
};
