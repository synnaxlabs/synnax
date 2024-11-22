// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form, Select, Text } from "@synnaxlabs/pluto";
import { type KeyedNamed } from "@synnaxlabs/x";

import {
  type ChannelType,
  DEVICES,
  type InputChannelType,
  type ModelKey,
  type OutputChannelType,
  type Port,
} from "@/hardware/labjack/device/types";

export interface SelectPortProps extends Select.SingleProps<string, Port> {
  model: ModelKey;
  channelType: ChannelType;
}

export const SelectPort = ({ model, channelType, ...props }: SelectPortProps) => {
  const data = DEVICES[model].ports[channelType === "TC" ? "AI" : channelType];
  return (
    <Select.Single<string, Port>
      data={data}
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
};

const INPUT_CHANNEL_TYPES: KeyedNamed<InputChannelType>[] = [
  { key: "AI", name: "Analog In" },
  { key: "DI", name: "Digital In" },
  { key: "TC", name: "Thermocouple" },
];

const OUTPUT_CHANNEL_TYPES: KeyedNamed<OutputChannelType>[] = [
  { key: "AO", name: "Analog Out" },
  { key: "DO", name: "Digital Out" },
];

export const SelectInputChannelTypeField = Form.buildDropdownButtonSelectField<
  InputChannelType,
  KeyedNamed<InputChannelType>
>({
  fieldKey: "type",
  fieldProps: { label: "Channel Type" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: INPUT_CHANNEL_TYPES,
  },
});

export const SelectOutputChannelTypeField = Form.buildDropdownButtonSelectField<
  OutputChannelType,
  KeyedNamed<OutputChannelType>
>({
  fieldKey: "type",
  fieldProps: { label: "Channel Type" },
  inputProps: {
    entryRenderKey: "name",
    columns: [{ key: "name", name: "Name" }],
    data: OUTPUT_CHANNEL_TYPES,
  },
});
