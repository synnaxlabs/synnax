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
import { digitalHardwareZ } from "@/feature/ni/task/checks";
import { createNextDOChannel } from "@/feature/ni/task/createChannel";
import { getDigitalChannelDeviceKey } from "@/feature/ni/task/getDigitalChannelDeviceKey";
import { PORT_LINE_COLUMNS, PortLineCells } from "@/feature/ni/task/PortLine";
import { digitalOutputName, digitalOutputStateName } from "@/feature/ni/task/ports";
import { LinesView } from "@/feature/ni/task/PortsView";
import {
  deployDigitalWriteConfigZ,
  DIGITAL_WRITE_SCHEMAS,
  DIGITAL_WRITE_TYPE,
  digitalWriteConfigZ,
  type DigitalWriteSchemas,
  type DOChannel,
} from "@/feature/ni/task/types";
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<DigitalWriteSchemas>>();

const Properties = () => (
  <>
    <Select />
    <Flex.Box x>
      <Task.Fields.StateUpdateRate />
    </Flex.Box>
  </>
);

const ChannelListItem = ({ itemKey, ...rest }: Task.ChannelListItemProps) => {
  const entry = root.config.channels[itemKey];
  const ch = PForm.useFieldValue(entry);
  const device = useFromConfig();
  const skipped = Task.useSkipped(
    root.config.channels,
    itemKey,
    getDigitalChannelDeviceKey,
  );
  return (
    <Task.Views.ListChannelItem
      {...rest}
      itemKey={itemKey}
      path={entry}
      channel={ch.cmdChannel}
      defaultName={device == null ? undefined : digitalOutputName(ch, device)}
      stateChannel={ch.stateChannel}
      stateDefaultName={device == null ? undefined : digitalOutputStateName(ch, device)}
      skipped={skipped}
    >
      <PortLineCells path={entry} />
    </Task.Views.ListChannelItem>
  );
};

const listItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const dev = useFromConfig();
  PForm.useCheck(root.config, digitalHardwareZ(dev, "output"));
  const resolve = (ch: DOChannel) => {
    if (dev == null) return null;
    const pair =
      dev.properties.digitalOutput.channels[getDigitalChannelDeviceKey(ch)] ??
      PlatformDevice.ZERO_COMMAND_STATE_PAIR;
    return { cmdChannel: pair.command, stateChannel: pair.state };
  };
  const map =
    dev?.properties.hardware == null
      ? undefined
      : (props: Task.Views.PortsProps<DOChannel>) => (
          <LinesView<DOChannel>
            {...props}
            device={dev}
            direction="output"
            create={(channels, placement) =>
              createNextDOChannel(channels, placement, dev)
            }
            name={digitalOutputName}
          />
        );
  return (
    <Task.Views.List<DOChannel>
      path={root.config.channels}
      createChannel={(channels, copy) => createNextDOChannel(channels, { copy }, dev)}
      listItem={listItem}
      columns={PORT_LINE_COLUMNS}
      channelColumns={Task.Views.LIST_WRITE_CHANNEL_COLUMNS}
      checkable
      contextMenuItems={Task.writeChannelContextMenuItems}
      resolve={resolve}
      map={map}
    />
  );
};

const getInitialValues: Task.GetInitialValues<DigitalWriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = digitalWriteConfigZ.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createNextDOChannel([])];
  return { name: "NI digital write task", type: DIGITAL_WRITE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<typeof digitalWriteConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const before = deep.copy(dev.properties);
  PlatformDevice.checkConfigured(dev);
  let modified = false;
  let shouldCreateStateIndex = primitive.isZero(
    dev.properties.digitalOutput.stateIndex,
  );
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.digitalOutput.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw errors.fromUnknown(e);
    }
  const identifier = channel.escapeInvalidName(dev.properties.identifier);
  try {
    if (shouldCreateStateIndex) {
      modified = true;
      const stateIndex = await client.channels.create({
        name: `${identifier}_do_state_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.digitalOutput.stateIndex = stateIndex.key;
      dev.properties.digitalOutput.channels = {};
    }
    const commandsToCreate: DOChannel[] = [];
    const statesToCreate: DOChannel[] = [];
    for (const channel of Task.firstPerPort(
      config.channels,
      getDigitalChannelDeviceKey,
    )) {
      const key = getDigitalChannelDeviceKey(channel);
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
          name: digitalOutputStateName(c, dev),
          index: dev.properties.digitalOutput.stateIndex,
          dataType: "uint8",
        })),
      );
      states.forEach((s, i) => {
        const key = getDigitalChannelDeviceKey(statesToCreate[i]);
        if (!(key in dev.properties.digitalOutput.channels))
          dev.properties.digitalOutput.channels[key] = { state: s.key, command: 0 };
        else dev.properties.digitalOutput.channels[key].state = s.key;
      });
    }
    if (commandsToCreate.length > 0) {
      modified = true;
      const commandIndexes = await client.channels.create(
        commandsToCreate.map((c) => ({
          name: `${digitalOutputName(c, dev)}_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commands = await client.channels.create(
        commandsToCreate.map((c, i) => ({
          name: digitalOutputName(c, dev),
          index: commandIndexes[i].key,
          dataType: "uint8",
        })),
      );
      commands.forEach((s, i) => {
        const key = getDigitalChannelDeviceKey(commandsToCreate[i]);
        if (!(key in dev.properties.digitalOutput.channels))
          dev.properties.digitalOutput.channels[key] = { state: 0, command: s.key };
        else dev.properties.digitalOutput.channels[key].command = s.key;
      });
    }
  } finally {
    if (modified) await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  config.channels = config.channels.map((c) => {
    const key = getDigitalChannelDeviceKey(c);
    const pair = dev.properties.digitalOutput.channels[key];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const DigitalWrite = Task.wrapForm({
  Properties,
  Form,
  schemas: DIGITAL_WRITE_SCHEMAS,
  deployConfigZ: deployDigitalWriteConfigZ,
  onConfigure,
  type: "ni_digital_write",
});

export const useCreateDigitalWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const DigitalWriteSelectable: Selector.Selectable = {
  type: DIGITAL_WRITE_TYPE,
  title: "NI digital write task",
  label: "Digital write",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateDigitalWrite,
};
