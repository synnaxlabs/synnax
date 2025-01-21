// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select, Text } from "@synnaxlabs/pluto";

import {
  DEVICES,
  type ModelKey,
  type Port,
  type PortType,
} from "@/hardware/labjack/device/types";

export interface SelectPortProps extends Select.SingleProps<string, Port> {
  model: ModelKey;
  portType: PortType;
}

export const SelectPort = ({ model, portType, ...props }: SelectPortProps) => (
  <Select.Single<string, Port>
    data={DEVICES[model].ports[portType]}
    columns={[
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
    ]}
    allowNone={false}
    entryRenderKey="key"
    {...props}
  />
);
