// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { errors, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Device } from "@/feature/ni/device";
import { AOChannelForm } from "@/feature/ni/task/AOChannelForm";
import { createAOChannel } from "@/feature/ni/task/createChannel";
import { SelectAOChannelTypeField } from "@/feature/ni/task/SelectAOChannelTypeField";
import {
  ANALOG_WRITE_SCHEMAS,
  ANALOG_WRITE_TYPE,
  analogWriteConfigZ,
  type AnalogWriteSchemas,
  AO_CHANNEL_TYPE_ICONS,
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type AOChannelType,
  ZERO_ANALOG_WRITE_PAYLOAD,
} from "@/feature/ni/task/types";
import { Task as ServiceTask } from "@/feature/task";
import { Device as CommonDevice } from "@/primitive/device";
import { Selector } from "@/primitive/selector";
import { Task } from "@/primitive/task";

export const ANALOG_WRITE_LAYOUT: ServiceTask.Layout = {
  ...ServiceTask.LAYOUT,
  type: ANALOG_WRITE_TYPE,
  name: ZERO_ANALOG_WRITE_PAYLOAD.name,
  icon: "Logo.NI",
};

export const AnalogWriteSelectable = Selector.createSimpleItem({
  title: "NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  layout: ANALOG_WRITE_LAYOUT,
});

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x>
      <Task.Fields.StateUpdateRate />
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const item = PForm.useFieldValue<AOChannel>(path);
  if (item == null) return null;
  const { port, cmdChannel, stateChannel, type } = item;
  const Icon = AO_CHANNEL_TYPE_ICONS[type];
  return (
    <Task.Layouts.ListAndDetailsChannelItem
      {...props}
      port={port}
      hasTareButton={false}
      channel={cmdChannel}
      stateChannel={stateChannel}
      portMaxChars={2}
      canTare={false}
      path={path}
      icon={{ icon: <Icon />, name: AO_CHANNEL_TYPE_NAMES[type] }}
    />
  );
};

const ChannelDetails = ({ path }: Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<AOChannelType>(`${path}.type`);
  return (
    <>
      <SelectAOChannelTypeField path={path} />
      <AOChannelForm type={type} path={path} />
    </>
  );
};

const channelDetails = Component.renderProp(ChannelDetails);
const channelListItem = Component.renderProp(ChannelListItem);

const Form: FC<Task.FormProps<AnalogWriteSchemas>> = () => (
  <Task.Layouts.ListAndDetails
    listItem={channelListItem}
    details={channelDetails}
    createChannel={createAOChannel}
    contextMenuItems={Task.writeChannelContextMenuItems}
  />
);

const getInitialValues: Task.GetInitialValues<AnalogWriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg =
    config != null
      ? analogWriteConfigZ.parse(config)
      : ZERO_ANALOG_WRITE_PAYLOAD.config;
  return {
    ...ZERO_ANALOG_WRITE_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Task.OnConfigure<typeof analogWriteConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  CommonDevice.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateStateIndex = primitive.isZero(dev.properties.analogOutput.stateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.analogOutput.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw errors.fromUnknown(e);
    }
  const identifier = channel.escapeInvalidName(dev.properties.identifier);
  try {
    if (shouldCreateStateIndex) {
      modified = true;
      const stateIndex = await client.channels.create({
        name: `${identifier}_ao_state_time`,
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
          else throw errors.fromUnknown(e);
        }
        try {
          await client.channels.retrieve(command);
        } catch (e) {
          if (NotFoundError.matches(e)) commandsToCreate.push(channel);
          else throw errors.fromUnknown(e);
        }
      }
    }
    if (statesToCreate.length > 0) {
      modified = true;
      const states = await client.channels.create(
        statesToCreate.map((c) => ({
          name: primitive.isNonZero(c.stateChannelName)
            ? c.stateChannelName
            : `${identifier}_ao_${c.port}_state`,
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
          name: primitive.isNonZero(c.cmdChannelName)
            ? `${c.cmdChannelName}_time`
            : `${identifier}_ao_${c.port}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commands = await client.channels.create(
        commandsToCreate.map((c, i) => ({
          name: primitive.isNonZero(c.cmdChannelName)
            ? c.cmdChannelName
            : `${identifier}_ao_${c.port}_cmd`,
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
  } finally {
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }
  config.channels = config.channels.map((c) => {
    const pair = dev.properties.analogOutput.channels[c.port.toString()];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const AnalogWrite = ServiceTask.wrapForm({
  Properties,
  Form,
  schemas: ANALOG_WRITE_SCHEMAS,
  type: "ni_analog_write",
  getInitialValues,
  onConfigure,
});
