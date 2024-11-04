// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/pluto";
import { KeyedNamed } from "@synnaxlabs/x";

import {
  ChannelType,
  DEVICES,
  InputChannelType,
  ModelKey,
  OutputChannelType,
  Port,
} from "@/hardware/labjack/device/types";

export interface SelectPortProps extends Select.SingleProps<string, Port> {
  model: ModelKey;
  channelType: ChannelType;
}

export const SelectPort = ({ model, channelType, ...props }: SelectPortProps) => {
  const data = DEVICES[model].ports[channelType];
  return (
    <Select.Single<string, Port>
      data={data}
      columns={[
        {
          key: "key",
          name: "Port",
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
];

const OUTPUT_CHANNEL_TYPES: KeyedNamed<OutputChannelType>[] = [
  { key: "AO", name: "Analog Out" },
  { key: "DO", name: "Digital Out" },
];

export interface SelectInputChannelTypeProps
  extends Omit<
    Select.ButtonProps<InputChannelType, KeyedNamed<InputChannelType>>,
    "data"
  > {}

export const SelectInputChannelType = (props: SelectInputChannelTypeProps) => (
  <Select.Button<InputChannelType, KeyedNamed<InputChannelType>>
    data={INPUT_CHANNEL_TYPES}
    entryRenderKey="name"
    {...props}
  />
);

export interface SelectOutputChannelTypeProps
  extends Omit<
    Select.ButtonProps<OutputChannelType, KeyedNamed<OutputChannelType>>,
    "data"
  > {}

export const SelectOutputChannelType = (props: SelectOutputChannelTypeProps) => (
  <Select.Button<OutputChannelType, KeyedNamed<OutputChannelType>>
    data={OUTPUT_CHANNEL_TYPES}
    entryRenderKey="name"
    {...props}
  />
);
