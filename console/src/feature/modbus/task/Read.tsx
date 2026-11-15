// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/modbus/task/Task.css";

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { DataType, deep, errors, id, primitive } from "@synnaxlabs/x";
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
  useOverlaps,
} from "@/feature/modbus/task/RegisterMap";
import {
  deployReadConfigZ,
  isStringReadChannel,
  isVariableDensityReadChannel,
  nextChannel,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  readChannelName,
  readDataType,
  readMapKey,
  type ReadSchemas,
  UNSUPPORTED_DATA_TYPES,
} from "@/feature/modbus/task/types";
import { Device } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<ReadSchemas>>();

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow wrap>
      <Task.Fields.SampleRate />
      <Task.Fields.StreamRate />
    </Flex.Box>
  </>
);

const COLUMNS: List.Column[] = [
  { name: "Type", width: "30rem", field: true },
  { name: "Address", width: "24rem", field: true },
  { name: "Data type", width: "26rem", field: true },
  {
    name: "Byte order",
    width: "var(--console-modbus-byte-order-width, 27rem)",
    field: true,
  },
];

const ChannelListItem = ({ itemKey, ...rest }: Task.ChannelListItemProps) => {
  const entry = path.config.channels[itemKey];
  const ch = PForm.useFieldValue(entry);
  const dev = useFromConfig();
  const overlapping = useOverlaps(path.config.channels, itemKey);
  return (
    <Task.Views.ListChannelItem
      {...rest}
      itemKey={itemKey}
      path={entry}
      channel={ch.channel}
      defaultName={
        dev == null
          ? undefined
          : readChannelName(channel.escapeInvalidName(dev.name), ch)
      }
    >
      <List.Cell>
        <RegisterField path={entry} schemas={READ_CHANNEL_SCHEMAS} />
      </List.Cell>
      <List.Cell>
        <AddressField path={entry} variant="shadow" overlapping={overlapping} />
      </List.Cell>
      <List.Cell>
        <DataTypeCell
          path={entry}
          channel={ch}
          hideDataTypes={UNSUPPORTED_DATA_TYPES}
        />
      </List.Cell>
      <List.Cell>
        {isVariableDensityReadChannel(ch) && (
          <ByteOrderField
            path={deep.variant(entry, "type", ch.type)}
            dataType={ch.dataType}
          />
        )}
        {isStringReadChannel(ch) && (
          <ByteOrderField
            path={deep.variant(entry, "type", ch.type)}
            dataType={DataType.STRING.toString()}
          />
        )}
      </List.Cell>
    </Task.Views.ListChannelItem>
  );
};

const getOpenChannel = (channels: ReadChannel[]): ReadChannel => {
  if (channels.length === 0)
    return {
      ...READ_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
      key: id.create(),
    };
  return {
    ...nextChannel(channels[channels.length - 1]),
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
  };
};

const createMapChannel = (type: ReadChannel["type"], address: number): ReadChannel => ({
  ...READ_CHANNEL_SCHEMAS[type].parse({ type, address }),
  ...Task.READ_CHANNEL_OVERRIDE,
  key: id.create(),
});

const listItem = Component.renderProp(ChannelListItem);

const map = ({ selected, onSelect }: Task.Views.PortsProps<ReadChannel>) => (
  <RegisterMap
    path={path.config.channels}
    createChannel={createMapChannel}
    selected={selected}
    onSelect={onSelect}
  />
);

const Form: FC = () => {
  const dev = useFromConfig();
  const resolve = (c: ReadChannel) =>
    dev == null ? null : { channel: dev.properties.read.channels[readMapKey(c)] ?? 0 };
  return (
    <RegisterMapProvider>
      <Task.Views.List
        path={path.config.channels}
        createChannel={getOpenChannel}
        contextMenuItems={Task.readChannelContextMenuItem}
        listItem={listItem}
        resolve={resolve}
        map={map}
        columns={COLUMNS}
        checkable
      />
    </RegisterMapProvider>
  );
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ deviceKey }) => {
  const config = READ_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  config.channels = [getOpenChannel([])];
  return { name: "Modbus read task", type: READ_TYPE, config };
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const before = deep.copy(dev.properties);
  let shouldCreateIndex = false;
  if (dev.properties.read.index)
    try {
      await client.channels.retrieve(dev.properties.read.index);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw errors.fromUnknown(e);
    }
  else shouldCreateIndex = true;
  let modified = false;
  const safeName = channel.escapeInvalidName(dev.name);
  try {
    if (shouldCreateIndex) {
      modified = true;
      const index = await client.channels.create({
        name: `${safeName}_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.read.index = index.key;
    }

    const toCreate: ReadChannel[] = [];
    for (const c of config.channels) {
      const key = readMapKey(c);
      const existing = dev.properties.read.channels[key];
      if (existing == null) toCreate.push(c);
      else
        try {
          await client.channels.retrieve(existing.toString());
        } catch (e) {
          if (NotFoundError.matches(e)) toCreate.push(c);
          else throw errors.fromUnknown(e);
        }
    }
    if (toCreate.length > 0) {
      modified = true;
      const channels = await client.channels.create(
        toCreate.map((c) => ({
          name: primitive.isNonZero(c.name) ? c.name : readChannelName(safeName, c),
          dataType: readDataType(c) ?? DataType.UINT8.toString(),
          index: dev.properties.read.index,
        })),
      );

      channels.forEach((c, i) => {
        const channel = toCreate[i];
        dev.properties.read.channels[readMapKey(channel)] = c.key;
      });
    }
  } finally {
    if (modified) await Device.setProperties(client, dev, before, SCHEMAS);
  }

  config.channels.forEach((c) => {
    c.channel = dev.properties.read.channels[readMapKey(c)];
  });

  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "modbus_read",
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "Modbus read task",
  label: "Read",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateRead,
};
