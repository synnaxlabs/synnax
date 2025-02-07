// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form as PForm, List, Text } from "@synnaxlabs/pluto";
import { primitiveIsZero } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { AnalogOutputChannelForm } from "@/hardware/ni/task/AnalogOutputChannelForm";
import { generateAnalogOutputChannel } from "@/hardware/ni/task/generateChannel";
import { SelectAnalogOutputChannelTypeField } from "@/hardware/ni/task/SelectAnalogOutputChannelTypeField";
import {
  ANALOG_WRITE_TYPE,
  type AnalogOutputChannel,
  type AnalogOutputChannelType,
  type AnalogWriteConfig,
  analogWriteConfigZ,
  type AnalogWriteStateDetails,
  type AnalogWriteType,
  AO_CHANNEL_TYPE_NAMES,
  ZERO_ANALOG_WRITE_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Layout } from "@/layout";

export const ANALOG_WRITE_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  key: ANALOG_WRITE_TYPE,
  type: ANALOG_WRITE_TYPE,
  name: ZERO_ANALOG_WRITE_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const ANALOG_WRITE_SELECTABLE: Layout.Selectable = {
  key: ANALOG_WRITE_TYPE,
  title: "NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  create: async ({ layoutKey }) => ({ ...ANALOG_WRITE_LAYOUT, key: layoutKey }),
};

const Properties = () => (
  <>
    <Device.Select />
    <Align.Space direction="x" grow>
      <Common.Task.Fields.StateUpdateRate />
      <Common.Task.Fields.DataSaving />
    </Align.Space>
  </>
);

interface ChannelListItemProps
  extends Common.Task.ChannelListItemProps<AnalogOutputChannel> {}

const ChannelListItem = ({ path, isSnapshot, ...rest }: ChannelListItemProps) => {
  const {
    entry: { port, type },
  } = rest;
  return (
    <List.ItemFrame {...rest} justify="spaceBetween" align="center">
      <Align.Space direction="x">
        <Text.Text level="p" shade={6}>
          {port}
        </Text.Text>
        <Text.Text level="p" shade={9}>
          {AO_CHANNEL_TYPE_NAMES[type]}
        </Text.Text>
      </Align.Space>
      <Common.Task.EnableDisableButton
        path={`${path}.enabled`}
        isSnapshot={isSnapshot}
      />
    </List.ItemFrame>
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<AnalogOutputChannelType>(`${path}.type`);
  return (
    <>
      <SelectAnalogOutputChannelTypeField path={path} />
      <AnalogOutputChannelForm type={type} path={path} />
    </>
  );
};

const Form: FC<
  Common.Task.FormProps<AnalogWriteConfig, AnalogWriteStateDetails, AnalogWriteType>
> = ({ task, isSnapshot }) => (
  <Common.Task.Layouts.ListAndDetails
    ListItem={ChannelListItem}
    Details={ChannelDetails}
    generateChannel={generateAnalogOutputChannel}
    isSnapshot={isSnapshot}
    initalChannels={task.config.channels}
  />
);

const getInitialPayload: Common.Task.GetInitialPayload<
  AnalogWriteConfig,
  AnalogWriteStateDetails,
  AnalogWriteType
> = (deviceKey) => ({
  ...ZERO_ANALOG_WRITE_PAYLOAD,
  config: {
    ...ZERO_ANALOG_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_ANALOG_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure = async (client: Synnax, config: AnalogWriteConfig) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>(
    config.device,
  );
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateStateIndex = primitiveIsZero(dev.properties.analogOutput.stateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.analogOutput.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }
  if (shouldCreateStateIndex) {
    modified = true;
    const stateIndex = await client.channels.create({
      name: `${dev.properties.identifier}_ao_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.analogOutput.stateIndex = stateIndex.key;
    dev.properties.analogOutput.channels = {};
  }
  const commandsToCreate: AnalogOutputChannel[] = [];
  const statesToCreate: AnalogOutputChannel[] = [];
  for (const channel of config.channels) {
    const exPair = dev.properties.analogOutput.channels[channel.port.toString()];
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
        name: `${dev.properties.identifier}_ao_${c.port}_state`,
        index: dev.properties.analogOutput.stateIndex,
        dataType: "float32",
      })),
    );
    states.forEach((s, i) => {
      const key = statesToCreate[i].port.toString();
      if (!(key in dev.properties.analogOutput.channels))
        dev.properties.analogOutput.channels[key] = { state: s.key, command: 0 };
      else dev.properties.analogOutput.channels[key].state = s.key;
    });
  }
  if (commandsToCreate.length > 0) {
    modified = true;
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.properties.identifier}_ao_${c.port}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.properties.identifier}_ao_${c.port}_cmd`,
        index: commandIndexes[i].key,
        dataType: "float32",
      })),
    );
    commands.forEach((s, i) => {
      const key = commandsToCreate[i].port.toString();
      if (!(key in dev.properties.analogOutput.channels))
        dev.properties.analogOutput.channels[key] = { state: 0, command: s.key };
      else dev.properties.analogOutput.channels[key].command = s.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels = config.channels.map((c) => {
    const pair = dev.properties.analogOutput.channels[c.port.toString()];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return config;
};

export const AnalogWrite = Common.Task.wrapForm(() => <Properties />, Form, {
  configSchema: analogWriteConfigZ,
  type: ANALOG_WRITE_TYPE,
  getInitialPayload,
  onConfigure,
});
