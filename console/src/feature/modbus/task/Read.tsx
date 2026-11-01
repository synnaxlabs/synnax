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
import { Select } from "@synnaxlabs/lyra/select";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, deep, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import { Select as SelectDevice } from "@/feature/modbus/device/Select";
import { MAKE, SCHEMAS } from "@/feature/modbus/device/types";
import { AddressField } from "@/feature/modbus/task/AddressField";
import { OrderFields } from "@/feature/modbus/task/OrderFields";
import { SelectReadChannelTypeField } from "@/feature/modbus/task/SelectReadChannelTypeField";
import {
  BIT_BOUNDS,
  deployReadConfigZ,
  isStringChannel,
  isVariableDensityReadChannel,
  nextChannel,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  readChannelName,
  readMapKey,
  type ReadSchemas,
  STRING_LENGTH_BOUNDS,
  type TypedReadChannel,
  UNSUPPORTED_READ_DATA_TYPES,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<ReadSchemas>>();

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <Task.Fields.SampleRate />
      <Task.Fields.StreamRate />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const entry = path.config.channels[itemKey];
  const ch = PForm.useFieldValue(entry);
  const { channel } = ch;
  return (
    <Select.Item
      {...props}
      className={CSS.BE("modbus", "channel-item")}
      justify="between"
      align="center"
      x
    >
      <Flex.Box x pack className={CSS.B("channel-item")}>
        <SelectReadChannelTypeField
          path={entry.type}
          onChange={(value, { get, set }) => {
            const prevType = get(entry.type).value;
            if (prevType === value) return;
            const next = READ_CHANNEL_SCHEMAS[value].parse({ type: value });
            const prevParent = get(entry).value;
            const schema = READ_CHANNEL_SCHEMAS[value];
            set(entry, {
              ...deep.overrideValidItems(next, prevParent, schema),
              type: value,
            } as ReadChannel);
          }}
        />
        <AddressField path={entry} />
        {"bit" in ch && (
          <PForm.NumericField
            inputProps={{
              showDragHandle: false,
              bounds: BIT_BOUNDS,
              startContent: "Bit",
              "aria-label": "Bit",
            }}
            showLabel={false}
            showHelpText={false}
            path={deep.variant(entry, "type", ch.type).bit}
          />
        )}
        {isVariableDensityReadChannel(ch) && (
          <PForm.Field
            path={deep.variant(entry, "type", ch.type).dataType}
            showLabel={false}
            showHelpText={false}
          >
            {renderTelemSelectDataType}
          </PForm.Field>
        )}
        {isStringChannel(ch) && (
          <PForm.NumericField
            inputProps={{
              showDragHandle: false,
              bounds: STRING_LENGTH_BOUNDS,
              startContent: "String length",
              "aria-label": "String length",
            }}
            showLabel={false}
            showHelpText={false}
            path={deep.variant(entry, "type", ch.type).stringLength}
          />
        )}
      </Flex.Box>
      {isVariableDensityReadChannel(ch) && (
        <OrderFields
          path={deep.variant(entry, "type", ch.type)}
          dataType={ch.dataType}
        />
      )}
      <Flex.Box x align="center" grow justify="end">
        <Task.ChannelName
          channel={channel}
          namePath={entry.name}
          id={Task.getChannelNameID(itemKey)}
        />
        <Task.EnabledCheckbox path={entry.disabled} />
      </Flex.Box>
    </Select.Item>
  );
};

const renderTelemSelectDataType = Component.renderProp(
  (props: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType
      {...props}
      hideDataTypes={UNSUPPORTED_READ_DATA_TYPES}
      location="bottom"
    />
  ),
);

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

const listItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const dev = useFromConfig();
  const resolve = (c: ReadChannel) =>
    dev == null ? null : { channel: dev.properties.read.channels[readMapKey(c)] ?? 0 };
  return (
    <Task.Views.List<ReadChannel>
      path={path.config.channels}
      createChannel={getOpenChannel}
      contextMenuItems={Task.readChannelContextMenuItem}
      listItem={listItem}
      resolve={resolve}
    />
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
          dataType: (c as TypedReadChannel).dataType ?? DataType.UINT8.toString(),
          index: dev.properties.read.index,
        })),
      );

      channels.forEach((c, i) => {
        const channel = toCreate[i];
        dev.properties.read.channels[readMapKey(channel)] = c.key;
      });
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
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

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "Modbus read task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateRead,
});
