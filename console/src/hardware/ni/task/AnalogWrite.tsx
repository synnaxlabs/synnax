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
import { Align, Form as PForm } from "@synnaxlabs/pluto";
import { primitiveIsZero } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { AOChannelForm } from "@/hardware/ni/task/AOChannelForm";
import { generateAOChannel } from "@/hardware/ni/task/generateChannel";
import { SelectAOChannelTypeField } from "@/hardware/ni/task/SelectAOChannelTypeField";
import {
  ANALOG_WRITE_TYPE,
  type AnalogWriteConfig,
  analogWriteConfigZ,
  type AnalogWriteStateDetails,
  type AnalogWriteType,
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type AOChannelType,
  ZERO_ANALOG_WRITE_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Selector } from "@/selector";

export const ANALOG_WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: ANALOG_WRITE_TYPE,
  name: ZERO_ANALOG_WRITE_PAYLOAD.name,
  icon: "Logo.NI",
};

export const ANALOG_WRITE_SELECTABLE: Selector.Selectable = {
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

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<AOChannel> {}

const ChannelListItem = ({ path, isSnapshot, ...rest }: ChannelListItemProps) => {
  const {
    entry: { port, type, cmdChannel },
  } = rest;
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...rest}
      port={port}
      hasTareButton={false}
      channel={cmdChannel}
      portMaxChars={2}
      canTare={false}
      onTare={() => {}}
      isSnapshot={isSnapshot}
      path={path}
      name={AO_CHANNEL_TYPE_NAMES[type]}
    />
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<AOChannelType>(`${path}.type`);
  return (
    <>
      <SelectAOChannelTypeField path={path} />
      <AOChannelForm type={type} path={path} />
    </>
  );
};

const Form: FC<
  Common.Task.FormProps<AnalogWriteConfig, AnalogWriteStateDetails, AnalogWriteType>
> = ({ task, isSnapshot }) => (
  <Common.Task.Layouts.ListAndDetails
    ListItem={ChannelListItem}
    Details={ChannelDetails}
    generateChannel={generateAOChannel}
    isSnapshot={isSnapshot}
    initialChannels={task.config.channels}
  />
);

const getInitialPayload: Common.Task.GetInitialPayload<
  AnalogWriteConfig,
  AnalogWriteStateDetails,
  AnalogWriteType
> = ({ deviceKey }) => ({
  ...ZERO_ANALOG_WRITE_PAYLOAD,
  config: {
    ...ZERO_ANALOG_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_ANALOG_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<AnalogWriteConfig> = async (
  client,
  config,
) => {
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
  const commandsToCreate: AOChannel[] = [];
  const statesToCreate: AOChannel[] = [];
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
  return [config, dev.rack];
};

export const AnalogWrite = Common.Task.wrapForm(Properties, Form, {
  configSchema: analogWriteConfigZ,
  type: ANALOG_WRITE_TYPE,
  getInitialPayload,
  onConfigure,
});
