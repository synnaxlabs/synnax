// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { AOChannelForm } from "@/hardware/ni/task/AOChannelForm";
import { createAOChannel } from "@/hardware/ni/task/createChannel";
import { SelectAOChannelTypeField } from "@/hardware/ni/task/SelectAOChannelTypeField";
import {
  ANALOG_WRITE_SCHEMAS,
  ANALOG_WRITE_TYPE,
  analogWriteConfigZ,
  type analogWriteStatusDataZ,
  type analogWriteTypeZ,
  AO_CHANNEL_TYPE_ICONS,
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
    <Flex.Box x>
      <Common.Task.Fields.StateUpdateRate />
      <Common.Task.Fields.DataSaving />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const item = PForm.useFieldValue<AOChannel>(path);
  if (item == null) return null;
  const { port, cmdChannel, stateChannel, type } = item;
  const Icon = AO_CHANNEL_TYPE_ICONS[type];
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...props}
      port={port}
      hasTareButton={false}
      channel={cmdChannel}
      stateChannel={stateChannel}
      portMaxChars={2}
      canTare={false}
      path={itemKey}
      icon={{ icon: <Icon />, name: AO_CHANNEL_TYPE_NAMES[type] }}
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

const channelDetails = Component.renderProp(ChannelDetails);
const channelListItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<
    typeof analogWriteTypeZ,
    typeof analogWriteConfigZ,
    typeof analogWriteStatusDataZ
  >
> = () => (
  <Common.Task.Layouts.ListAndDetails
    listItem={channelListItem}
    details={channelDetails}
    createChannel={createAOChannel}
    contextMenuItems={Common.Task.writeChannelContextMenuItems}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof analogWriteTypeZ,
  typeof analogWriteConfigZ,
  typeof analogWriteStatusDataZ
> = ({ deviceKey, config }) => {
  const cfg =
    config != null
      ? analogWriteConfigZ.parse(config)
      : ZERO_ANALOG_WRITE_PAYLOAD.config;
  return {
    ...ZERO_ANALOG_WRITE_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof analogWriteConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>({
    key: config.device,
  });
  Common.Device.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateStateIndex = primitive.isZero(dev.properties.analogOutput.stateIndex);
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

export const AnalogWrite = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: ANALOG_WRITE_SCHEMAS,
  type: ANALOG_WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
