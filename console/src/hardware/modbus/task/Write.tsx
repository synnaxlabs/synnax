// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form as PForm, List, Select } from "@synnaxlabs/pluto";
import { deep, id } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/modbus/device";
import { SelectOutputChannelType } from "@/hardware/modbus/task/SelectOutputChannelType";
import {
  HOLDING_REGISTER_OUTPUT_TYPE,
  OUTPUT_CHANNEL_SCHEMAS,
  type OutputChannel,
  type OutputChannelType,
  WRITE_TYPE,
  type WriteConfig,
  writeConfigZ,
  type WriteStateDetails,
  type WriteType,
  ZERO_OUTPUT_CHANNELS,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/modbus/task/types";
import { type Selector } from "@/selector";

export const WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.Modbus",
};

export const WRITE_SELECTABLE: Selector.Selectable = {
  key: WRITE_TYPE,
  title: "Modbus Write Task",
  icon: <Icon.Logo.Modbus />,
  create: async ({ layoutKey }) => ({ ...WRITE_LAYOUT, key: layoutKey }),
};

const Properties = () => (
  <>
    <Device.Select />
    <Common.Task.Fields.DataSaving />
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<OutputChannel> {
  path: string;
  isSnapshot: boolean;
}

const ChannelListItem = ({ path, isSnapshot, ...rest }: ChannelListItemProps) => {
  const { entry } = rest;
  return (
    <List.ItemFrame
      {...rest}
      style={{ width: "100%" }}
      justify="spaceBetween"
      align="center"
      direction="x"
    >
      <Align.Pack direction="x" align="center" className={CSS.B("channel-item")}>
        <SelectOutputChannelType
          path={path}
          onChange={(value, { get, set, path }) => {
            const prevType = get<OutputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(ZERO_OUTPUT_CHANNELS[value]);
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<OutputChannel>(parentPath).value;
            const schema = OUTPUT_CHANNEL_SCHEMAS[value];
            set(parentPath, {
              ...deep.overrideValidItems(next, prevParent, schema),
              type: value,
            });
          }}
        />
        <PForm.NumericField
          inputProps={{ showDragHandle: false }}
          hideIfNull
          showLabel={false}
          showHelpText={false}
          path={`${path}.address`}
        />
        {entry.type === HOLDING_REGISTER_OUTPUT_TYPE && (
          <PForm.Field<string>
            path={`${path}.dataType`}
            showLabel={false}
            showHelpText={false}
            hideIfNull
          >
            {({ value, onChange }) => (
              <Select.DataType
                value={value}
                onChange={(v) => onChange(v)}
                hideVariableDensity={true}
              />
            )}
          </PForm.Field>
        )}
      </Align.Pack>
      <Common.Task.EnableDisableButton
        path={`${path}.enabled`}
        isSnapshot={isSnapshot}
      />
    </List.ItemFrame>
  );
};

const getOpenChannel = (channels: OutputChannel[]): OutputChannel => {
  if (channels.length === 0)
    return {
      type: "coil_output",
      address: 0,
      channel: 0,
      enabled: true,
      key: id.create(),
    };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    key: id.create(),
    address: channelToCopy.address + 1,
  };
};

const Form: FC<Common.Task.FormProps<WriteConfig, WriteStateDetails, WriteType>> = ({
  isSnapshot,
}) => {
  const createChannel = useCallback(
    (channels: OutputChannel[]) => getOpenChannel(channels),
    [],
  );
  const listItem = useCallback(
    ({ key, ...p }: Common.Task.ChannelListItemProps<OutputChannel>) => (
      <ChannelListItem key={key} {...p} />
    ),
    [],
  );
  return (
    <Common.Task.Layouts.List<OutputChannel>
      isSnapshot={isSnapshot}
      createChannel={createChannel}
      listItem={listItem}
    />
  );
};

const writeMapKey = (channel: OutputChannel) =>
  `${channel.type}-${channel.address.toString()}`.replace("_", "-");

const getInitialPayload: Common.Task.GetInitialPayload<
  WriteConfig,
  WriteStateDetails,
  WriteType
> = ({ deviceKey }) => ({
  ...ZERO_WRITE_PAYLOAD,
  config: {
    ...ZERO_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<WriteConfig> = async (client, config) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties>(config.device);
  const commandsToCreate: OutputChannel[] = [];
  for (const channel of config.channels) {
    const key = writeMapKey(channel);
    const existing = dev.properties.write.channels[key];
    if (existing == null) {
      commandsToCreate.push(channel);
      continue;
    }
    try {
      await client.channels.retrieve(existing);
    } catch (e) {
      if (NotFoundError.matches(e)) commandsToCreate.push(channel);
      else throw e;
    }
  }

  if (commandsToCreate.length > 0) {
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.name}_${c.type}_${c.address}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.name}_${c.type}_${c.address}_cmd`,
        dataType: c.type === "holding_register_output" ? c.dataType : "uint8",
        index: commandIndexes[i].key,
      })),
    );
    commands.forEach((c, i) => {
      const channel = commandsToCreate[i];
      dev.properties.write.channels[writeMapKey(channel)] = c.key;
    });
    await client.hardware.devices.create(dev);
  }

  config.channels = config.channels.map((c) => ({
    ...c,
    channel: dev.properties.write.channels[writeMapKey(c)],
  }));

  return [config, dev.rack];
};

export const Write = Common.Task.wrapForm({
  Properties,
  Form,
  configSchema: writeConfigZ,
  type: WRITE_TYPE,
  getInitialPayload,
  onConfigure,
});
