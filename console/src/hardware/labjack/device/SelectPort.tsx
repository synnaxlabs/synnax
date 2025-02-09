// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type List, Select, Text } from "@synnaxlabs/pluto";

import {
  DEVICES,
  type Model,
  type Port,
  type PortType,
} from "@/hardware/labjack/device/types";

export interface SelectPortProps
  extends Omit<
    Select.SingleProps<string, Port>,
    "columns" | "data" | "entryRenderKey"
  > {
  model: Model;
  portType: PortType;
}

const COLUMNS: List.ColumnSpec<string, Port>[] = [
  { key: "key", name: "Port" },
  {
    key: "aliases",
    name: "Aliases",
    render: ({ entry: { aliases } }) => (
      <Text.Text level="small" shade={8}>
        {aliases.join(", ")}
      </Text.Text>
    ),
  },
];

export const SelectPort = ({ model, portType, ...rest }: SelectPortProps) => (
  <Select.Single<string, Port>
    allowNone={false}
    {...rest}
    columns={COLUMNS}
    data={DEVICES[model].ports[portType]}
    entryRenderKey="key"
  />
);
