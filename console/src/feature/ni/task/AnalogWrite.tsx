// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { errors, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/ni/device/queries";
import { Select } from "@/feature/ni/device/Select";
import { MAKE, SCHEMAS } from "@/feature/ni/device/types";
import { AOChannelForm } from "@/feature/ni/task/AOChannelForm";
import { analogWriteHardwareZ } from "@/feature/ni/task/checks";
import { createNextAOChannel } from "@/feature/ni/task/createChannel";
import { DetailsTitle as Title } from "@/feature/ni/task/DetailsTitle";
import {
  ANALOG_OUTPUT,
  analogOutputStateName,
  enabledFirst,
  holdState,
  portClaims,
} from "@/feature/ni/task/ports";
import { PortsView } from "@/feature/ni/task/PortsView";
import {
  ANALOG_WRITE_SCHEMAS,
  ANALOG_WRITE_TYPE,
  analogWriteConfigZ,
  type AnalogWriteSchemas,
  AO_CHANNEL_TYPE_ICONS,
  AO_CHANNEL_TYPE_NAMES,
  type AOChannel,
  type AOChannelType,
  deployAnalogWriteConfigZ,
} from "@/feature/ni/task/types";
import { Device as PlatformDevice } from "@/platform/device";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <Select />
    <Flex.Box x>
      <Task.Fields.StateUpdateRate />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const item = PForm.useFieldValue<AOChannel>(path);
  const device = useFromConfig();
  if (item == null) return null;
  const { port, cmdChannel, stateChannel, type } = item;
  const Icon = AO_CHANNEL_TYPE_ICONS[type];
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      port={port}
      hasTareButton={false}
      channel={cmdChannel}
      defaultName={device == null ? undefined : ANALOG_OUTPUT.name(item, device)}
      stateChannel={stateChannel}
      stateDefaultName={
        device == null ? undefined : analogOutputStateName(item, device)
      }
      portMaxChars={2}
      canTare={false}
      path={path}
      icon={{ icon: <Icon />, name: AO_CHANNEL_TYPE_NAMES[type] }}
    />
  );
};

const ChannelDetails = ({ path }: Task.Views.DetailsProps) => {
  const type = PForm.useFieldValue<AOChannelType>(`${path}.type`);
  return <AOChannelForm type={type} path={path} />;
};

const channelDetails = Component.renderProp(ChannelDetails);

const DetailsTitle = ({ path }: Task.Views.DetailsProps) => {
  const value = PForm.useFieldValue<AOChannel>(path);
  const channels = PForm.useFieldValue<AOChannel[]>("config.channels");
  const device = useFromConfig();
  const Icon = AO_CHANNEL_TYPE_ICONS[value.type];
  return (
    <Title
      port={{
        label: String(value.port),
        state: holdState(portClaims(channels), [value.port]),
        disabled: value.disabled,
      }}
      type={{ icon: <Icon />, name: AO_CHANNEL_TYPE_NAMES[value.type] }}
      channel={value.cmdChannel}
      namePath={`${path}.cmdChannelName`}
      defaultName={device == null ? undefined : ANALOG_OUTPUT.name(value, device)}
      model={device?.model}
    />
  );
};

const detailsTitle = Component.renderProp(DetailsTitle);
const channelListItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const dev = useFromConfig();
  Task.useConfigCheck(analogWriteHardwareZ(dev));
  const resolve = (ch: AOChannel) => {
    if (dev == null) return null;
    const pair =
      dev.properties.analogOutput.channels[ch.port.toString()] ??
      PlatformDevice.ZERO_COMMAND_STATE_PAIR;
    return { cmdChannel: pair.command, stateChannel: pair.state };
  };
  const ports =
    dev?.properties.hardware == null
      ? undefined
      : (props: Task.Views.PortsProps) => (
          <PortsView<AOChannel>
            {...props}
            kind={ANALOG_OUTPUT}
            devices={[dev]}
            deviceOf={() => dev.key}
            create={(channels, _, port) => createNextAOChannel(channels, { port }, dev)}
          />
        );
  return (
    <Task.Views.ListAndDetails<AOChannel>
      listItem={channelListItem}
      details={channelDetails}
      detailsTitle={detailsTitle}
      createChannel={(channels, copy) => createNextAOChannel(channels, { copy }, dev)}
      contextMenuItems={Task.writeChannelContextMenuItems}
      resolve={resolve}
      ports={ports}
    />
  );
};

const getInitialValues: Task.GetInitialValues<AnalogWriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = analogWriteConfigZ.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createNextAOChannel([])];
  return { name: "NI analog write task", type: ANALOG_WRITE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<typeof analogWriteConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  PlatformDevice.checkConfigured(dev);
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
    // Channels on one port share its data channel, named by an enabled one if any.
    const seen = new Set<string>();
    for (const channel of enabledFirst(config.channels)) {
      const key = channel.port.toString();
      if (seen.has(key)) continue;
      seen.add(key);
      const exPair = dev.properties.analogOutput.channels[key];
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
          name: analogOutputStateName(c, dev),
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
          name: ANALOG_OUTPUT.name(c, dev),
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
    if (modified) await client.devices.create(dev, SCHEMAS);
  }
  config.channels = config.channels.map((c) => {
    const pair = dev.properties.analogOutput.channels[c.port.toString()];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const AnalogWrite = Task.wrapForm({
  Properties,
  Form,
  schemas: ANALOG_WRITE_SCHEMAS,
  deployConfigZ: deployAnalogWriteConfigZ,
  deviceConfigZ: async (client, config) =>
    analogWriteHardwareZ(
      await client.devices.retrieve({ key: config.device, schemas: SCHEMAS }),
    ),
  type: "ni_analog_write",
  getInitialValues,
  onConfigure,
});

export const useCreateAnalogWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const AnalogWriteSelectable = Selector.createSelectable({
  type: ANALOG_WRITE_TYPE,
  title: "NI analog write task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogWrite,
});
