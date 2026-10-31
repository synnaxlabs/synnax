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
import { Icon } from "@synnaxlabs/lyra/icon";
import { errors, type optional, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/ni/device/queries";
import { Select } from "@/feature/ni/device/Select";
import { MAKE, SCHEMAS } from "@/feature/ni/device/types";
import { digitalHardwareZ } from "@/feature/ni/task/checks";
import { createNextDOChannel } from "@/feature/ni/task/createChannel";
import {
  DigitalChannelList,
  type DigitalNameComponentProps,
} from "@/feature/ni/task/DigitalChannelList";
import { getDigitalChannelDeviceKey } from "@/feature/ni/task/getDigitalChannelDeviceKey";
import { digitalOutputName, enabledFirst } from "@/feature/ni/task/ports";
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

interface NameComponentProps extends DigitalNameComponentProps<DOChannel> {}

const NameComponent = ({ path, ...rest }: NameComponentProps) => {
  const filteredRest: optional.Optional<
    typeof rest,
    "cmdChannelName" | "stateChannelName"
  > = rest;
  delete filteredRest.cmdChannelName;
  delete filteredRest.stateChannelName;
  return (
    <Task.WriteChannelNames
      {...rest}
      cmdNamePath={`${path}.cmdChannelName`}
      stateNamePath={`${path}.stateChannelName`}
    />
  );
};

const name = Component.renderProp(NameComponent);

const Form: FC = () => {
  const dev = useFromConfig();
  Task.useConfigCheck(digitalHardwareZ(dev, "output"));
  const resolve = (ch: DOChannel) => {
    if (dev == null) return null;
    const pair =
      dev.properties.digitalOutput.channels[getDigitalChannelDeviceKey(ch)] ??
      PlatformDevice.ZERO_COMMAND_STATE_PAIR;
    return { cmdChannel: pair.command, stateChannel: pair.state };
  };
  return (
    <DigitalChannelList
      createChannel={(channels) => createNextDOChannel(channels, {}, dev)}
      name={name}
      contextMenuItems={Task.writeChannelContextMenuItems}
      resolve={resolve}
      ports={
        dev?.properties.hardware == null
          ? undefined
          : (props) => (
              <LinesView<DOChannel>
                {...props}
                device={dev}
                direction="output"
                create={(channels, placement) =>
                  createNextDOChannel(channels, placement, dev)
                }
                name={digitalOutputName}
              />
            )
      }
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
    // Channels on one port share its data channel, named by an enabled one if any.
    const seen = new Set<string>();
    for (const channel of enabledFirst(config.channels)) {
      const key = getDigitalChannelDeviceKey(channel);
      if (seen.has(key)) continue;
      seen.add(key);
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
          name: primitive.isNonZero(c.stateChannelName)
            ? c.stateChannelName
            : `${identifier}_do_${c.port}_${c.line}_state`,
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
          name: primitive.isNonZero(c.cmdChannelName)
            ? `${c.cmdChannelName}_time`
            : `${identifier}_do_${c.port}_${c.line}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commands = await client.channels.create(
        commandsToCreate.map((c, i) => ({
          name: primitive.isNonZero(c.cmdChannelName)
            ? c.cmdChannelName
            : `${identifier}_do_${c.port}_${c.line}_cmd`,
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
    if (modified) await client.devices.create(dev, SCHEMAS);
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
  deviceConfigZ: async (client, config) =>
    digitalHardwareZ(
      await client.devices.retrieve({ key: config.device, schemas: SCHEMAS }),
      "output",
    ),
  getInitialValues,
  onConfigure,
  type: "ni_digital_write",
});

export const useCreateDigitalWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const DigitalWriteSelectable = Selector.createSelectable({
  type: DIGITAL_WRITE_TYPE,
  title: "NI digital write task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateDigitalWrite,
});
