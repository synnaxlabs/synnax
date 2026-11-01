// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/modbus/task/Task.css";

import { channel, type modbus, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { DataType, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import { Select as SelectDevice } from "@/feature/modbus/device/Select";
import { MAKE, SCHEMAS } from "@/feature/modbus/device/types";
import { AddressField } from "@/feature/modbus/task/AddressField";
import { ByteOrderField } from "@/feature/modbus/task/ByteOrder";
import { DataTypeCell } from "@/feature/modbus/task/DataTypeCell";
import { RegisterField } from "@/feature/modbus/task/Register";
import {
  RegisterMap,
  RegisterMapProvider,
  useHoveredRef,
  useIsHovered,
  useOverlaps,
} from "@/feature/modbus/task/RegisterMap";
import {
  deployWriteConfigZ,
  nextChannel,
  UNSUPPORTED_WRITE_DATA_TYPES,
  WRITE_CHANNEL_SCHEMAS,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteChannelType,
  type WriteSchemas,
} from "@/feature/modbus/task/types";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => <SelectDevice />;

const COLUMNS: List.Column[] = [
  { name: "Type", width: "30rem", field: true },
  { name: "Address", width: "24rem", field: true },
  { name: "Data type", width: "32rem", field: true },
  { name: "Byte order", width: "27rem", field: true },
];

// Auto-generated channel names and device map keys keep the released type
// spellings, so channels created before the labels were renamed keep matching.
const NAME_TYPES: Record<WriteChannelType, string> = {
  coil: "coil_output",
  holding_register: "holding_register_output",
  holding_register_bit: "holding_register_bit_output",
};

const writeMapKey = (channel: WriteChannel) => {
  let s = `${NAME_TYPES[channel.type]}-${channel.address.toString()}`;
  if (channel.type === "holding_register") s += `-${channel.dataType}`;
  if (channel.type === "holding_register_bit") s += `-${channel.bit}`;
  return s.replaceAll("_", "-");
};

// Python and older Consoles saved a holding register's command channel under a key with
// no data type. The client returns the older Console's key in camel case.
const legacyWriteMapKeys = (address: number): string[] => [
  `holding-register-output-${address}`,
  `holding-registerOutput-${address}`,
];

/**
 * Moves the command channel of a holding register from a legacy key to its current key
 * when the channel's data type matches.
 * @returns true if it moved a channel.
 */
const adoptLegacyCommand = async (
  client: Synnax,
  channels: Record<string, number>,
  ch: modbus.HoldingRegisterWriteChannel,
): Promise<boolean> => {
  for (const legacy of legacyWriteMapKeys(ch.address)) {
    const key = channels[legacy];
    if (key == null) continue;
    let cmd: channel.Channel;
    try {
      cmd = await client.channels.retrieve(key);
    } catch (e) {
      if (NotFoundError.matches(e)) continue;
      throw errors.fromUnknown(e);
    }
    if (!cmd.dataType.equals(ch.dataType)) continue;
    delete channels[legacy];
    channels[writeMapKey(ch)] = key;
    return true;
  }
  return false;
};

const commandName = (deviceName: string, channel: WriteChannel) => {
  let s = `${deviceName}_${NAME_TYPES[channel.type]}_${channel.address}`;
  if (channel.type === "holding_register")
    s += `_${new DataType(channel.dataType).toString(true)}`;
  if (channel.type === "holding_register_bit") s += `_${channel.bit}`;
  return `${s}_cmd`;
};

const commandDataType = (channel: WriteChannel): string =>
  channel.type === "holding_register" ? channel.dataType : DataType.UINT8.toString();

const ChannelListItem = ({ itemKey, ...rest }: Task.ChannelListItemProps) => {
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<WriteChannel>(path);
  const dev = useFromConfig();
  const hovered = useIsHovered(itemKey);
  const hoveredRef = useHoveredRef(itemKey);
  const overlapping = useOverlaps(itemKey);
  return (
    <Task.Views.ListChannelItem
      {...rest}
      itemKey={itemKey}
      path={path}
      channel={ch.channel}
      defaultName={
        dev == null ? undefined : commandName(channel.escapeInvalidName(dev.name), ch)
      }
      ref={hoveredRef}
      hovered={hovered}
    >
      <List.Cell>
        <RegisterField<WriteChannel> path={path} schemas={WRITE_CHANNEL_SCHEMAS} />
      </List.Cell>
      <List.Cell>
        <AddressField path={path} variant="shadow" overlapping={overlapping} />
      </List.Cell>
      <List.Cell>
        <DataTypeCell
          path={path}
          channel={ch}
          hideDataTypes={UNSUPPORTED_WRITE_DATA_TYPES}
        />
      </List.Cell>
      <List.Cell>
        {ch.type === "holding_register" && (
          <ByteOrderField path={path} dataType={ch.dataType} />
        )}
      </List.Cell>
    </Task.Views.ListChannelItem>
  );
};

const getOpenChannel = (channels: WriteChannel[]): WriteChannel => {
  if (channels.length === 0)
    return {
      ...WRITE_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
      key: id.create(),
    };
  return { ...nextChannel(channels[channels.length - 1]), key: id.create(), name: "" };
};

const createMapChannel = (type: WriteChannelType, address: number): WriteChannel => ({
  ...WRITE_CHANNEL_SCHEMAS[type].parse({ type, address }),
  key: id.create(),
});

const listItem = Component.renderProp(ChannelListItem);

const HEADER = (
  <>
    <Header.Header>
      <Header.Title weight={500} color={10}>
        Channels
      </Header.Title>
    </Header.Header>
    <RegisterMap createChannel={createMapChannel} />
  </>
);

const Form: FC = () => {
  const dev = useFromConfig();
  const resolve = (c: WriteChannel) =>
    dev == null
      ? null
      : { channel: dev.properties.write.channels[writeMapKey(c)] ?? 0 };
  return (
    <RegisterMapProvider>
      <Task.Views.List<WriteChannel>
        createChannel={getOpenChannel}
        contextMenuItems={Task.readChannelContextMenuItem}
        listItem={listItem}
        resolve={resolve}
        header={HEADER}
        columns={COLUMNS}
        checkable
      />
    </RegisterMapProvider>
  );
};

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({ deviceKey }) => {
  const config = WRITE_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  config.channels = [getOpenChannel([])];
  return { name: "Modbus write task", type: WRITE_TYPE, config };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const { channels } = dev.properties.write;
  let changed = false;
  const commandsToCreate: WriteChannel[] = [];
  for (const channel of config.channels) {
    const key = writeMapKey(channel);
    if (channels[key] == null && channel.type === "holding_register")
      changed = (await adoptLegacyCommand(client, channels, channel)) || changed;
    const existing = channels[key];
    if (existing == null) {
      commandsToCreate.push(channel);
      continue;
    }
    try {
      await client.channels.retrieve(existing);
    } catch (e) {
      if (NotFoundError.matches(e)) commandsToCreate.push(channel);
      else throw errors.fromUnknown(e);
    }
  }

  const safeName = channel.escapeInvalidName(dev.name);
  if (commandsToCreate.length > 0) {
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: primitive.isNonZero(c.name)
          ? `${c.name}_time`
          : `${commandName(safeName, c)}_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: primitive.isNonZero(c.name) ? c.name : commandName(safeName, c),
        dataType: commandDataType(c),
        index: commandIndexes[i].key,
      })),
    );
    commands.forEach((c, i) => {
      const channel = commandsToCreate[i];
      channels[writeMapKey(channel)] = c.key;
    });
    changed = true;
  }
  if (changed) await client.devices.create(dev, SCHEMAS);

  config.channels = config.channels.map((c) => ({
    ...c,
    channel: channels[writeMapKey(c)],
  }));

  return [config, dev.rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: "modbus_write",
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "Modbus write task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateWrite,
});
