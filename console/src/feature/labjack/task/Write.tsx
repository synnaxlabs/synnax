// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/labjack/task/Write.css";

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { deep, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { use } from "@/feature/labjack/device/queries";
import { Select } from "@/feature/labjack/device/Select";
import {
  AO_PORT_TYPE,
  type AOPort,
  type Device,
  DO_PORT_TYPE,
  type DOPort,
  MAKE,
  type Model,
  PORTS,
  SCHEMAS,
} from "@/feature/labjack/device/types";
import { uniqueTerminalsZ } from "@/feature/labjack/task/checks";
import {
  convertPortTypeToWriteChannelType,
  convertWriteChannelTypeToPortType,
} from "@/feature/labjack/task/convertChannelTypeToPortType";
import { getOpenPort } from "@/feature/labjack/task/getOpenPort";
import { PortPicker } from "@/feature/labjack/task/PortPicker";
import { enabledFirst } from "@/feature/labjack/task/ports";
import { PortsView } from "@/feature/labjack/task/PortsView";
import { SelectWriteChannelType } from "@/feature/labjack/task/SelectWriteChannelType";
import {
  createWriteChannel,
  deployWriteConfigZ,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteChannelType,
  type WriteSchemas,
} from "@/feature/labjack/task/types";
import { useConfigure } from "@/feature/labjack/task/useConfigure";
import { CSS } from "@/platform/css";
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

/** The name a channel shows by: its command channel's, or the one configuring gives. */
const commandName = (c: WriteChannel, identifier: string): string =>
  primitive.isNonZero(c.cmdChannelName)
    ? c.cmdChannelName
    : `${identifier}_${c.port}_cmd`;

/** The name a channel's state channel shows by. */
const stateName = (c: WriteChannel, identifier: string): string =>
  primitive.isNonZero(c.stateChannelName)
    ? c.stateChannelName
    : `${identifier}_${c.port}_state`;

interface ChannelListItemProps extends Task.ChannelListItemProps {
  device: Device;
}

const ChannelListItem = ({ device, ...rest }: ChannelListItemProps) => {
  const { model } = device;
  const identifier = channel.escapeInvalidName(device.properties.identifier);
  const path = `config.channels.${rest.itemKey}`;
  const { get, set } = PForm.useContext();
  const item = PForm.useFieldValue<WriteChannel>(path);
  const { port, type, cmdChannel, stateChannel } = item;
  return (
    <List.Item {...rest} full="x" justify="between">
      <Flex.Box pack x align="center">
        <PForm.Field<string>
          path={`${path}.port`}
          showLabel={false}
          showHelpText={false}
          onChange={(value) => {
            if (port === value) return;
            set(path, { ...item, port: value });
          }}
        >
          {({ value, onChange, preview }) => (
            <PortPicker<WriteChannel>
              value={value}
              onChange={onChange}
              preview={preview}
              model={model}
              type={convertWriteChannelTypeToPortType(type)}
              channel={item.key}
              name={(c) => commandName(c, identifier)}
              className={CSS.BE("labjack-write", "port-select")}
            >
              <PForm.Field<WriteChannelType>
                key="type"
                path={`${path}.type`}
                showLabel={false}
                showHelpText={false}
                gap="large"
                onChange={(value) => {
                  if (type === value) return;
                  const portType = convertWriteChannelTypeToPortType(value);
                  const others = get<WriteChannel[]>("config.channels").value.filter(
                    ({ key }) => key !== item.key,
                  );
                  const port = (
                    getOpenPort(others, model, [portType]) ?? PORTS[model][portType][0]
                  ).key;
                  set(path, { ...item, type: value, port });
                }}
                empty
              >
                {selectWriteChannelType}
              </PForm.Field>
            </PortPicker>
          )}
        </PForm.Field>
      </Flex.Box>
      <Flex.Box x align="center" justify="evenly">
        <Task.WriteChannelNames
          cmdChannel={cmdChannel}
          itemKey={item.key}
          stateChannel={stateChannel}
          cmdNamePath={`${path}.cmdChannelName`}
          stateNamePath={`${path}.stateChannelName`}
          cmdDefaultName={commandName(item, identifier)}
          stateDefaultName={stateName(item, identifier)}
        />
        <Task.EnabledCheckbox path={`${path}.disabled`} />
      </Flex.Box>
    </List.Item>
  );
};

const selectWriteChannelType = Component.renderProp(SelectWriteChannelType);

const createFirstChannel = (): WriteChannel => ({
  ...createWriteChannel("digital"),
  key: id.create(),
});

/** Creates a channel on the port, of the type the port takes. */
const createChannelOn = (port: AOPort | DOPort): WriteChannel => ({
  ...createWriteChannel(convertPortTypeToWriteChannelType(port.type)),
  key: id.create(),
  port: port.key,
});

/**
 * Creates a channel on the lowest free port of the last channel's type, then of the
 * other type. Null when every port is taken.
 */
const createNextChannel = (
  channels: WriteChannel[],
  model: Model,
): WriteChannel | null => {
  const last = channels.at(-1);
  const preferred =
    last == null ? DO_PORT_TYPE : convertWriteChannelTypeToPortType(last.type);
  const backup = preferred === DO_PORT_TYPE ? AO_PORT_TYPE : DO_PORT_TYPE;
  const port = getOpenPort(channels, model, [preferred, backup]);
  if (port == null) return null;
  if (last == null) return createChannelOn(port);
  return {
    ...deep.copy(last),
    ...Task.WRITE_CHANNEL_OVERRIDE,
    type: convertPortTypeToWriteChannelType(port.type),
    key: id.create(),
    port: port.key,
  };
};

interface ChannelListProps {
  device: Device;
}

const ChannelList = ({ device }: ChannelListProps) => {
  const { model } = device;
  const identifier = channel.escapeInvalidName(device.properties.identifier);
  Task.useConfigCheck(uniqueTerminalsZ(model));
  const createChannel = (channels: WriteChannel[]) =>
    createNextChannel(channels, model);
  const listItem = ({ key, ...p }: Task.ChannelListItemProps) => (
    <ChannelListItem key={key} {...p} device={device} />
  );
  const ports = (p: Task.Views.PortsProps) => (
    <PortsView<WriteChannel, AOPort | DOPort>
      {...p}
      model={model}
      ports={[...PORTS[model][AO_PORT_TYPE], ...PORTS[model][DO_PORT_TYPE]]}
      create={(_, port) => createChannelOn(port)}
      name={(c) => commandName(c, identifier)}
    />
  );
  const resolve = (c: WriteChannel) => {
    const pair =
      device.properties[convertWriteChannelTypeToPortType(c.type)].channels[c.port] ??
      PlatformDevice.ZERO_COMMAND_STATE_PAIR;
    return { cmdChannel: pair.command, stateChannel: pair.state };
  };
  return (
    <Task.Views.List<WriteChannel>
      createChannel={createChannel}
      listItem={listItem}
      contextMenuItems={Task.writeChannelContextMenuItems}
      resolve={resolve}
      ports={ports}
    />
  );
};

const Form: FC = PlatformDevice.wrapTaskForm({
  use,
  useConfigure,
  Content: ChannelList,
});

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createFirstChannel()];
  return { name: "LabJack write task", type: WRITE_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  PlatformDevice.checkConfigured(dev);
  let modified = false;
  let shouldCreateStateIndex = primitive.isZero(dev.properties.writeStateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(dev.properties.writeStateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw errors.fromUnknown(e);
    }
  const identifier = channel.escapeInvalidName(dev.properties.identifier);
  try {
    if (shouldCreateStateIndex) {
      modified = true;
      const stateIndex = await client.channels.create({
        name: `${identifier}_write_state_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.writeStateIndex = stateIndex.key;
      dev.properties.DO.channels = {};
      dev.properties.AO.channels = {};
    }
    const commandChannelsToCreate: WriteChannel[] = [];
    const stateChannelsToCreate: WriteChannel[] = [];
    // Channels on one port share its data channels, named by an enabled one if any.
    const seen = new Set<string>();
    for (const channel of enabledFirst(config.channels)) {
      const key = channel.port;
      const type = convertWriteChannelTypeToPortType(channel.type);
      if (seen.has(`${type}/${key}`)) continue;
      seen.add(`${type}/${key}`);
      const existingPair = dev.properties[type].channels[key];
      if (existingPair == null) {
        commandChannelsToCreate.push(channel);
        stateChannelsToCreate.push(channel);
      } else {
        const { state, command } = existingPair;
        try {
          await client.channels.retrieve(state);
        } catch (e) {
          if (NotFoundError.matches(e)) stateChannelsToCreate.push(channel);
          else throw errors.fromUnknown(e);
        }
        try {
          await client.channels.retrieve(command);
        } catch (e) {
          if (NotFoundError.matches(e)) commandChannelsToCreate.push(channel);
          else throw errors.fromUnknown(e);
        }
      }
    }
    if (stateChannelsToCreate.length > 0) {
      modified = true;
      const stateChannels = await client.channels.create(
        stateChannelsToCreate.map((c) => ({
          name: stateName(c, identifier),
          index: dev.properties.writeStateIndex,
          dataType: c.type === "analog" ? "float32" : "uint8",
        })),
      );
      stateChannels.forEach((c, i) => {
        const statesToCreateC = stateChannelsToCreate[i];
        const port = statesToCreateC.port;
        if (
          !(
            port in
            dev.properties[convertWriteChannelTypeToPortType(statesToCreateC.type)]
              .channels
          )
        )
          dev.properties[
            convertWriteChannelTypeToPortType(statesToCreateC.type)
          ].channels[port] = {
            state: c.key,
            command: 0,
          };
        else
          dev.properties[
            convertWriteChannelTypeToPortType(statesToCreateC.type)
          ].channels[port].state = c.key;
      });
    }
    if (commandChannelsToCreate.length > 0) {
      modified = true;
      const commandIndexes = await client.channels.create(
        commandChannelsToCreate.map((c) => ({
          name: `${commandName(c, identifier)}_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );
      const commandChannels = await client.channels.create(
        commandChannelsToCreate.map((c, i) => ({
          name: commandName(c, identifier),
          index: commandIndexes[i].key,
          dataType: c.type === "analog" ? "float32" : "uint8",
        })),
      );
      commandChannels.forEach((c, i) => {
        const cmdToCreate = commandChannelsToCreate[i];
        const port = cmdToCreate.port;
        if (
          !(
            port in
            dev.properties[convertWriteChannelTypeToPortType(cmdToCreate.type)].channels
          )
        )
          dev.properties[convertWriteChannelTypeToPortType(cmdToCreate.type)].channels[
            port
          ] = {
            state: 0,
            command: c.key,
          };
        else
          dev.properties[convertWriteChannelTypeToPortType(cmdToCreate.type)].channels[
            port
          ].command = c.key;
      });
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
  }
  config.channels = config.channels.map((c) => {
    const pair =
      dev.properties[convertWriteChannelTypeToPortType(c.type)].channels[c.port];
    return { ...c, cmdChannel: pair.command, stateChannel: pair.state };
  });
  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  deviceConfigZ: async (client, config) =>
    uniqueTerminalsZ(
      (await client.devices.retrieve({ key: config.device, schemas: SCHEMAS })).model,
    ),
  type: "labjack_write",
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "LabJack write task",
  icon: <Icon.Logo.LabJack />,
  useOnSelect: useCreateWrite,
});
