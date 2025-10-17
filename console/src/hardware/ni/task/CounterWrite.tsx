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
import { COChannelForm } from "@/hardware/ni/task/COChannelForm";
import { createCOChannel } from "@/hardware/ni/task/createChannel";
import { SelectCOChannelTypeField } from "@/hardware/ni/task/SelectCOChannelTypeField";
import {
  CO_CHANNEL_TYPE_ICONS,
  CO_CHANNEL_TYPE_NAMES,
  type COChannel,
  type COChannelType,
  COUNTER_WRITE_SCHEMAS,
  COUNTER_WRITE_TYPE,
  counterWriteConfigZ,
  type counterWriteStatusDataZ,
  type counterWriteTypeZ,
  ZERO_COUNTER_WRITE_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Selector } from "@/selector";

export const COUNTER_WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: COUNTER_WRITE_TYPE,
  name: ZERO_COUNTER_WRITE_PAYLOAD.name,
  icon: "Logo.NI",
};

export const COUNTER_WRITE_SELECTABLE: Selector.Selectable = {
  key: COUNTER_WRITE_TYPE,
  title: "NI Counter Write Task",
  icon: <Icon.Logo.NI />,
  create: async ({ layoutKey }) => ({ ...COUNTER_WRITE_LAYOUT, key: layoutKey }),
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
  const item = PForm.useFieldValue<COChannel>(path);
  if (item == null) return null;
  const { port, cmdChannel, stateChannel, type } = item;
  const Icon = CO_CHANNEL_TYPE_ICONS[type];
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
      icon={{ icon: <Icon />, name: CO_CHANNEL_TYPE_NAMES[type] }}
    />
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<COChannelType>(`${path}.type`);
  return (
    <>
      <SelectCOChannelTypeField path={path} />
      <COChannelForm type={type} path={path} />
    </>
  );
};

const channelDetails = Component.renderProp(ChannelDetails);
const channelListItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<
    typeof counterWriteTypeZ,
    typeof counterWriteConfigZ,
    typeof counterWriteStatusDataZ
  >
> = () => (
  <Common.Task.Layouts.ListAndDetails
    listItem={channelListItem}
    details={channelDetails}
    createChannel={createCOChannel}
    contextMenuItems={Common.Task.writeChannelContextMenuItems}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof counterWriteTypeZ,
  typeof counterWriteConfigZ,
  typeof counterWriteStatusDataZ
> = ({ deviceKey, config }) => {
  const cfg =
    config != null
      ? counterWriteConfigZ.parse(config)
      : ZERO_COUNTER_WRITE_PAYLOAD.config;
  return {
    ...ZERO_COUNTER_WRITE_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof counterWriteConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>({
    key: config.device,
  });
  Common.Device.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateStateIndex = primitive.isZero(dev.properties.counterOutput.stateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.counterOutput.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }
  if (shouldCreateStateIndex) {
    modified = true;
    const stateIndex = await client.channels.create({
      name: `${dev.properties.identifier}_co_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.counterOutput.stateIndex = stateIndex.key;
    dev.properties.counterOutput.channels = {};
  }
  const commandsToCreate: COChannel[] = [];
  const statesToCreate: COChannel[] = [];
  for (const channel of config.channels) {
    const exPair = dev.properties.counterOutput.channels[channel.port.toString()];
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
        name: `${dev.properties.identifier}_co_${c.port}_state`,
        index: dev.properties.counterOutput.stateIndex,
        dataType: "float32",
      })),
    );
    states.forEach((s, i) => {
      const key = statesToCreate[i].port.toString();
      if (!(key in dev.properties.counterOutput.channels))
        dev.properties.counterOutput.channels[key] = { state: s.key, command: 0 };
      else dev.properties.counterOutput.channels[key].state = s.key;
    });
  }
  if (commandsToCreate.length > 0) {
    modified = true;
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.properties.identifier}_co_${c.port}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.properties.identifier}_co_${c.port}_cmd`,
        index: commandIndexes[i].key,
        dataType: "float32",
      })),
    );
    commands.forEach((s, i) => {
      const key = commandsToCreate[i].port.toString();
      if (!(key in dev.properties.counterOutput.channels))
        dev.properties.counterOutput.channels[key] = { state: 0, command: s.key };
      else dev.properties.counterOutput.channels[key].command = s.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels = config.channels.map((c) => {
    const pair = dev.properties.counterOutput.channels[c.port.toString()];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const CounterWrite = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: COUNTER_WRITE_SCHEMAS,
  type: COUNTER_WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
