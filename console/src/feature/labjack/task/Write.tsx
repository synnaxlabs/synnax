// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { deep, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { use } from "@/feature/labjack/device/queries";
import { Select, useNoneSelected } from "@/feature/labjack/device/Select";
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
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => (
  <>
    <Select />
    <Flex.Box x>
      <Task.Fields.StateUpdateRate />
    </Flex.Box>
  </>
);

/** The name a channel shows by: its command channel's, or the one deploying gives. */
const commandName = (c: WriteChannel, identifier: string): string =>
  primitive.isNonZero(c.cmdChannelName)
    ? c.cmdChannelName
    : `${identifier}_${c.port}_cmd`;

/** The name a channel's state channel shows by. */
const stateName = (c: WriteChannel, identifier: string): string =>
  primitive.isNonZero(c.stateChannelName)
    ? c.stateChannelName
    : `${identifier}_${c.port}_state`;

const portOf = (c: WriteChannel): string =>
  `${convertWriteChannelTypeToPortType(c.type)}/${c.port}`;

interface ChannelListItemProps extends Task.ChannelListItemProps {
  device: Device;
}

const COLUMNS: List.Column[] = [
  { name: "Type", width: "24rem", field: true },
  { name: "Port", width: "22rem", field: true },
];

const ChannelListItem = ({ device, ...rest }: ChannelListItemProps) => {
  const { model } = device;
  const identifier = channel.escapeInvalidName(device.properties.identifier);
  const entry = path.config.channels[rest.itemKey];
  const { get, set } = PForm.useContext();
  const item = PForm.useFieldValue(entry);
  const skipped = Task.useSkipped(path.config.channels, rest.itemKey, portOf);
  const { port, type, cmdChannel, stateChannel } = item;
  return (
    <Task.Views.ListChannelItem
      {...rest}
      path={entry}
      channel={cmdChannel}
      defaultName={commandName(item, identifier)}
      stateChannel={stateChannel}
      stateDefaultName={stateName(item, identifier)}
      skipped={skipped}
    >
      <List.Cell>
        <PForm.Field<WriteChannelType>
          path={entry.type}
          showLabel={false}
          showHelpText={false}
          onChange={(value) => {
            if (type === value) return;
            const portType = convertWriteChannelTypeToPortType(value);
            const others = get(path.config.channels).value.filter(
              ({ key }) => key !== item.key,
            );
            const port = (
              getOpenPort(others, model, [portType]) ?? PORTS[model][portType][0]
            ).key;
            set(entry, { ...item, type: value, port });
          }}
        >
          {selectWriteChannelType}
        </PForm.Field>
      </List.Cell>
      <List.Cell>
        <PForm.Field
          path={entry.port}
          showLabel={false}
          showHelpText={false}
          onChange={(value) => {
            if (port === value) return;
            set(entry, { ...item, port: value });
          }}
        >
          {({ value, onChange, preview }) => (
            <PortPicker<WriteChannel>
              channelsPath={path.config.channels}
              value={value}
              onChange={onChange}
              preview={preview}
              model={model}
              type={convertWriteChannelTypeToPortType(type)}
              channel={item.key}
              name={(c) => commandName(c, identifier)}
            />
          )}
        </PForm.Field>
      </List.Cell>
    </Task.Views.ListChannelItem>
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
 * Creates a copy of the channel with the key, or of the last channel without one, on
 * the lowest free port of its type, then of the other type. Null when the channel to
 * copy is gone or every port is taken.
 */
const createNextChannel = (
  channels: WriteChannel[],
  model: Model,
  copy?: string,
): WriteChannel | null => {
  const template =
    copy == null ? channels.at(-1) : channels.find(({ key }) => key === copy);
  if (copy != null && template == null) return null;
  const preferred =
    template == null ? DO_PORT_TYPE : convertWriteChannelTypeToPortType(template.type);
  const backup = preferred === DO_PORT_TYPE ? AO_PORT_TYPE : DO_PORT_TYPE;
  const port = getOpenPort(channels, model, [preferred, backup]);
  if (port == null) return null;
  if (template == null) return createChannelOn(port);
  return {
    ...deep.copy(template),
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
  PForm.useCheck(path.config, uniqueTerminalsZ(model));
  const createChannel = (channels: WriteChannel[], copy?: string) =>
    createNextChannel(channels, model, copy);
  const listItem = ({ key, ...p }: Task.ChannelListItemProps) => (
    <ChannelListItem key={key} {...p} device={device} />
  );
  const map = (p: Task.Views.PortsProps<WriteChannel>) => (
    <PortsView<WriteChannel, AOPort | DOPort>
      {...p}
      model={model}
      ports={[...PORTS[model][AO_PORT_TYPE], ...PORTS[model][DO_PORT_TYPE]]}
      create={(_, port) => createChannelOn(port)}
      name={(c) => commandName(c, identifier)}
      compact
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
      path={path.config.channels}
      createChannel={createChannel}
      listItem={listItem}
      columns={COLUMNS}
      channelColumns={Task.Views.LIST_WRITE_CHANNEL_COLUMNS}
      checkable
      contextMenuItems={Task.writeChannelContextMenuItems}
      resolve={resolve}
      map={map}
    />
  );
};

const Form: FC = PlatformDevice.wrapTaskForm({
  use,
  useConfigure,
  useNoneSelected,
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

// Returns the index of a state channel the device maps that still exists, or 0 if there
// is none. A client that maps channels without setting writeStateIndex, such as Python,
// leaves the index only on the state channels.
const retrieveMappedStateIndex = async (
  client: Synnax,
  dev: Device,
): Promise<channel.Key> => {
  const pairs = [
    ...Object.values(dev.properties.AO.channels),
    ...Object.values(dev.properties.DO.channels),
  ];
  for (const { state } of pairs) {
    if (primitive.isZero(state)) continue;
    try {
      const { index } = await client.channels.retrieve(state);
      if (!primitive.isZero(index)) return index;
    } catch (e) {
      if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
    }
  }
  return 0;
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
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
      const mappedIndex = await retrieveMappedStateIndex(client, dev);
      if (!primitive.isZero(mappedIndex)) dev.properties.writeStateIndex = mappedIndex;
      else {
        const stateIndex = await client.channels.create({
          name: `${identifier}_write_state_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.writeStateIndex = stateIndex.key;
        dev.properties.DO.channels = {};
        dev.properties.AO.channels = {};
      }
    }
    const commandChannelsToCreate: WriteChannel[] = [];
    const stateChannelsToCreate: WriteChannel[] = [];
    for (const channel of Task.firstPerPort(config.channels, portOf)) {
      const key = channel.port;
      const type = convertWriteChannelTypeToPortType(channel.type);
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
    if (modified) await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
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
  type: "labjack_write",
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "LabJack write task",
  label: "Write",
  icon: <Icon.Logo.LabJack />,
  useOnSelect: useCreateWrite,
};
