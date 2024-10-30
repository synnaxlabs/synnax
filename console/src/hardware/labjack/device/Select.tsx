import { Override, Select } from "@synnaxlabs/pluto";
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

export const SelectInputChannelType = Override.createComponent<
  Select.SingleProps<InputChannelType, KeyedNamed<InputChannelType>>
>(Select.Button, {
  data: INPUT_CHANNEL_TYPES,
  entryRenderKey: "name",
});

export const SelectOutputChannelType = Override.createComponent<
  Select.SingleProps<OutputChannelType, KeyedNamed<OutputChannelType>>
>(Select.Button, { data: OUTPUT_CHANNEL_TYPES, entryRenderKey: "name" });
