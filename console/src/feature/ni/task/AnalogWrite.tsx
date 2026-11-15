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
import { deep, errors, primitive } from "@synnaxlabs/x";
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
  holdState,
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
  deployAnalogWriteConfigZ,
  isGenerated,
} from "@/feature/ni/task/types";
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<AnalogWriteSchemas>>();

const portOf = (c: AOChannel): string => c.port.toString();

const isCommanded = (c: AOChannel) => !isGenerated(c);

const contextMenuItems = (props: Task.ContextMenuItemProps<AOChannel>) => {
  const channel = props.channels.find(({ key }) => key === props.keys[0]);
  if (channel == null || isGenerated(channel)) return null;
  return Task.writeChannelContextMenuItems(props);
};

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
  const entry = root.config.channels[itemKey];
  const item = PForm.useFieldValue(entry);
  const device = useFromConfig();
  const skipped = Task.useSkipped(root.config.channels, itemKey, portOf, isCommanded);
  if (item == null) return null;
  const Icon = AO_CHANNEL_TYPE_ICONS[item.type];
  const names = isGenerated(item)
    ? {}
    : {
        channel: item.cmdChannel,
        defaultName: device == null ? undefined : ANALOG_OUTPUT.name(item, device),
        stateChannel: item.stateChannel,
        stateDefaultName:
          device == null ? undefined : analogOutputStateName(item, device),
      };
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      {...names}
      port={item.port}
      hasTareButton={false}
      skipped={skipped}
      portMaxChars={2}
      canTare={false}
      path={entry}
      icon={{ icon: <Icon />, name: AO_CHANNEL_TYPE_NAMES[item.type] }}
    />
  );
};

const ChannelDetails = ({ path }: Task.Views.DetailsProps<AOChannel>) => {
  const type = PForm.useFieldValue(path.type);
  return <AOChannelForm type={type} path={path} channelsPath={root.config.channels} />;
};

const channelDetails = Component.renderProp(ChannelDetails);

const DetailsTitle = ({ path }: Task.Views.DetailsProps<AOChannel>) => {
  const value = PForm.useFieldValue(path);
  const channels = PForm.useFieldValue(root.config.channels);
  const device = useFromConfig();
  const skipped = Task.useSkipped(root.config.channels, value.key, portOf, isCommanded);
  const Icon = AO_CHANNEL_TYPE_ICONS[value.type];
  return (
    <Title
      port={{
        label: String(value.port),
        state: holdState(ANALOG_OUTPUT, channels, value, device?.properties.hardware),
        disabled: value.disabled,
      }}
      type={{ icon: <Icon />, name: AO_CHANNEL_TYPE_NAMES[value.type] }}
      name={
        isGenerated(value)
          ? undefined
          : {
              channel: value.cmdChannel,
              path: deep.variant(path, "type", value.type).cmdChannelName,
              defaultName:
                device == null ? undefined : ANALOG_OUTPUT.name(value, device),
            }
      }
      skipped={skipped}
      model={device?.model}
    />
  );
};

const detailsTitle = Component.renderProp(DetailsTitle);
const channelListItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const dev = useFromConfig();
  PForm.useCheck(root.config, analogWriteHardwareZ(dev));
  const resolve = (ch: AOChannel) => {
    if (dev == null || isGenerated(ch)) return null;
    const pair =
      dev.properties.analogOutput.channels[ch.port.toString()] ??
      PlatformDevice.ZERO_COMMAND_STATE_PAIR;
    return { cmdChannel: pair.command, stateChannel: pair.state };
  };
  const ports =
    dev?.properties.hardware == null
      ? undefined
      : (props: Task.Views.PortsProps<AOChannel>) => (
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
      path={root.config.channels}
      listItem={channelListItem}
      details={channelDetails}
      detailsTitle={detailsTitle}
      createChannel={(channels, copy) => createNextAOChannel(channels, { copy }, dev)}
      contextMenuItems={contextMenuItems}
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
  const before = deep.copy(dev.properties);
  PlatformDevice.checkConfigured(dev);
  const commanded = config.channels.filter(isCommanded);
  if (commanded.length === 0) return [config, dev.rack];
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
    const commandsToCreate: typeof commanded = [];
    const statesToCreate: typeof commanded = [];
    for (const channel of Task.firstPerPort(commanded, portOf)) {
      const key = portOf(channel);
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
    if (modified) await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  config.channels = config.channels.map((c) => {
    if (isGenerated(c)) return c;
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
  type: "ni_analog_write",
  onConfigure,
});

export const useCreateAnalogWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const AnalogWriteSelectable: Selector.Selectable = {
  type: ANALOG_WRITE_TYPE,
  title: "NI analog write task",
  label: "Analog write",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogWrite,
};
