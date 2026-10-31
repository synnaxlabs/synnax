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
  type ReadChannelType,
  readMapKey,
  type ReadSchemas,
  STRING_LENGTH_BOUNDS,
  type TypedReadChannel,
  UNSUPPORTED_READ_DATA_TYPES,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

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
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<ReadChannel>(path);
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
          path={path}
          onChange={(value, { get, set, path }) => {
            const prevType = get<ReadChannelType>(path).value;
            if (prevType === value) return;
            const next = READ_CHANNEL_SCHEMAS[value].parse({ type: value });
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<ReadChannel>(parentPath).value;
            const schema = READ_CHANNEL_SCHEMAS[value];
            set(parentPath, {
              ...deep.overrideValidItems(next, prevParent, schema),
              type: value,
            });
          }}
        />
        <AddressField path={path} />
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
            path={`${path}.bit`}
          />
        )}
        {isVariableDensityReadChannel(ch) && (
          <PForm.Field<string>
            path={`${path}.dataType`}
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
            path={`${path}.stringLength`}
          />
        )}
      </Flex.Box>
      {isVariableDensityReadChannel(ch) && (
        <OrderFields path={path} dataType={ch.dataType} />
      )}
      <Flex.Box x align="center" grow justify="end">
        <Task.ChannelName
          channel={channel}
          namePath={`${path}.name`}
          id={Task.getChannelNameID(itemKey)}
        />
        <Task.EnabledCheckbox path={`${path}.disabled`} />
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
  getInitialValues,
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
