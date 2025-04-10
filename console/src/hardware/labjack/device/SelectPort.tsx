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
  type Model,
  type Port,
  PORTS,
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
  { key: "port", name: "Port", stringer: ({ alias, key }) => alias ?? key },
  {
    key: "aliases",
    name: "Aliases",
    render: ({ entry: { alias, key } }) =>
      alias != null ? (
        <Text.Text level="small" shade={10}>
          {key}
        </Text.Text>
      ) : null,
  },
];

const getEntryRenderKey = ({ alias, key }: Port) => alias ?? key;

export const SelectPort = ({ model, portType, ...rest }: SelectPortProps) => (
  <Select.Single<string, Port>
    allowNone={false}
    {...rest}
    columns={COLUMNS}
    data={PORTS[model][portType]}
    entryRenderKey={getEntryRenderKey}
  />
);
