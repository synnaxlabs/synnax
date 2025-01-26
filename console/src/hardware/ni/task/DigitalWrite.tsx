// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { DigitalListItem } from "@/hardware/ni/task/DigitalListItem";
import {
  DIGITAL_WRITE_TYPE,
  type DigitalWriteConfig,
  digitalWriteConfigZ,
  type DigitalWriteDetails,
  type DigitalWriteType,
  type DOChannel,
  ZERO_DIGITAL_WRITE_PAYLOAD,
  ZERO_DO_CHANNEL,
} from "@/hardware/ni/task/types";
import { type Layout } from "@/layout";

export const DIGITAL_WRITE_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  type: DIGITAL_WRITE_TYPE,
  name: ZERO_DIGITAL_WRITE_PAYLOAD.name,
  icon: "Logo.NI",
  key: DIGITAL_WRITE_TYPE,
};

export const DIGITAL_WRITE_SELECTABLE: Layout.Selectable = {
  key: DIGITAL_WRITE_TYPE,
  title: "NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  create: (key) => ({ ...DIGITAL_WRITE_LAYOUT, key }),
};

const generateChannel = (channels: DOChannel[]): DOChannel => {
  const line = Math.max(0, ...channels.map((v) => v.line)) + 1;
  return { ...ZERO_DO_CHANNEL, key: id.id(), line };
};

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<DOChannel> {}

const ChannelListItem = ({
  entry,
  entry: { cmdChannel, stateChannel },
  ...props
}: ChannelListItemProps): ReactElement => (
  <DigitalListItem {...props} entry={entry}>
    <Common.Task.ChannelName channel={cmdChannel} defaultName="No Command Channel" />
    <Common.Task.ChannelName channel={stateChannel} defaultName="No State Channel" />
  </DigitalListItem>
);

const Properties = (): ReactElement => (
  <>
    <Device.Select />
    <Common.Task.Fields.StateUpdateRate />
    <Common.Task.Fields.DataSaving />
  </>
);

const TaskForm: FC<
  Common.Task.FormProps<DigitalWriteConfig, DigitalWriteDetails, DigitalWriteType>
> = ({ isSnapshot }) => (
  <Common.Task.Layouts.List<DOChannel>
    isSnapshot={isSnapshot}
    generateChannel={generateChannel}
  >
    {(p) => <ChannelListItem {...p} />}
  </Common.Task.Layouts.List>
);

const getDeviceKey: (chan: DOChannel) => string = (chan) => `${chan.port}l${chan.line}`;

const zeroPayload: Common.Task.ZeroPayloadFunction<
  DigitalWriteConfig,
  DigitalWriteDetails,
  DigitalWriteType
> = (deviceKey) => ({
  ...ZERO_DIGITAL_WRITE_PAYLOAD,
  config: {
    ...ZERO_DIGITAL_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_DIGITAL_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure = async (
  client: Synnax,
  config: DigitalWriteConfig,
): Promise<DigitalWriteConfig> => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>(
    config.device,
  );
  let modified = false;
  let shouldCreateStateIndex = primitiveIsZero(dev.properties.digitalOutput.stateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.digitalOutput.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }
  if (shouldCreateStateIndex) {
    modified = true;
    const stateIndex = await client.channels.create({
      name: `${dev.properties.identifier}_do_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.digitalOutput.stateIndex = stateIndex.key;
    dev.properties.digitalOutput.channels = {};
  }
  const commandsToCreate: DOChannel[] = [];
  const statesToCreate: DOChannel[] = [];
  for (const channel of config.channels) {
    const key = getDeviceKey(channel);
    const exPair = dev.properties.digitalOutput.channels[key];
    if (exPair == null) {
      commandsToCreate.push(channel);
      statesToCreate.push(channel);
    } else {
      const { state, command } = exPair;
      try {
        await client.channels.retrieve(state);
      } catch (e) {
        if (NotFoundError.matches(e)) statesToCreate.push(channel);
        else throw e;
      }
      try {
        await client.channels.retrieve(command);
      } catch (e) {
        if (NotFoundError.matches(e)) commandsToCreate.push(channel);
        else throw e;
      }
    }
  }
  if (statesToCreate.length > 0) {
    modified = true;
    const states = await client.channels.create(
      statesToCreate.map((c) => ({
        name: `${dev.properties.identifier}_do_${c.port}_${c.line}_state`,
        index: dev.properties.digitalOutput.stateIndex,
        dataType: "uint8",
      })),
    );
    states.forEach((s, i) => {
      const key = getDeviceKey(statesToCreate[i]);
      if (!(key in dev.properties.digitalOutput.channels))
        dev.properties.digitalOutput.channels[key] = { state: s.key, command: 0 };
      else dev.properties.digitalOutput.channels[key].state = s.key;
    });
  }
  if (commandsToCreate.length > 0) {
    modified = true;
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.properties.identifier}_do_${c.port}_${c.line}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.properties.identifier}_do_${c.port}_${c.line}_cmd`,
        index: commandIndexes[i].key,
        dataType: "uint8",
      })),
    );
    commands.forEach((s, i) => {
      const key = `${commandsToCreate[i].port}l${commandsToCreate[i].line}`;
      if (!(key in dev.properties.digitalOutput.channels))
        dev.properties.digitalOutput.channels[key] = { state: 0, command: s.key };
      else dev.properties.digitalOutput.channels[key].command = s.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels = config.channels.map((c) => {
    const key = getDeviceKey(c);
    const pair = dev.properties.digitalOutput.channels[key];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return config;
};

export const DigitalWriteTask = Common.Task.wrapForm(<Properties />, TaskForm, {
  configSchema: digitalWriteConfigZ,
  type: DIGITAL_WRITE_TYPE,
  zeroPayload,
  onConfigure,
});
