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
import { Component, Flex, Form as PForm, Icon, Select, Telem } from "@synnaxlabs/pluto";
import { DataType, deep, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Select as SelectDevice } from "@/feature/modbus/device/Select";
import * as Device from "@/feature/modbus/device/types";
import { SelectReadChannelTypeField } from "@/feature/modbus/task/SelectReadChannelTypeField";
import {
  deployReadConfigZ,
  isVariableDensityReadChannel,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadChannelType,
  type ReadSchemas,
  type TypedReadChannel,
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
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const { type, channel } = PForm.useFieldValue<ReadChannel>(path);
  return (
    <Select.ListItem
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
        <PForm.NumericField
          inputProps={{ showDragHandle: false }}
          hideIfNull
          showLabel={false}
          showHelpText={false}
          path={`${path}.address`}
        />
        {(type === "input_register" || type === "holding_register") && (
          <PForm.Field<string>
            path={`${path}.dataType`}
            showLabel={false}
            showHelpText={false}
            hideIfNull
          >
            {renderTelemSelectDataType}
          </PForm.Field>
        )}
      </Flex.Box>
      <Flex.Box x align="center" grow justify="end">
        <Task.ChannelName
          channel={channel}
          namePath={`${path}.name`}
          id={Task.getChannelNameID(itemKey)}
        />
        <Task.EnableDisableButton path={`${path}.disabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const renderTelemSelectDataType = Component.renderProp(
  (props: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...props} hideVariableDensity location="bottom" />
  ),
);

const getOpenChannel = (channels: ReadChannel[]): ReadChannel => {
  if (channels.length === 0)
    return {
      ...READ_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
      key: id.create(),
    };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
    address: channelToCopy.address + 1,
  };
};

const listItem = Component.renderProp(ChannelListItem);

const Form: FC = () => (
  <Task.Views.List<ReadChannel>
    createChannel={getOpenChannel}
    contextMenuItems={Task.readChannelContextMenuItem}
    listItem={listItem}
  />
);

// Auto-generated channel names and device map keys keep the released type
// spellings, so channels created before the labels were renamed keep matching.
const NAME_TYPES: Record<ReadChannelType, string> = {
  coil: "coil_input",
  discrete_input: "discrete_input",
  holding_register: "holding_register_input",
  input_register: "register_input",
};

const readMapKey = (channel: ReadChannel) => {
  let s = `${NAME_TYPES[channel.type]}-${channel.address.toString()}`;
  if (isVariableDensityReadChannel(channel)) s += `-${channel.dataType}`;
  return s.replaceAll("_", "-");
};

const channelName = (deviceName: string, channel: ReadChannel) => {
  let s = `${deviceName}_${NAME_TYPES[channel.type]}_${channel.address}`;
  if (isVariableDensityReadChannel(channel))
    s += `_${new DataType(channel.dataType).toString(true)}`;
  return s;
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ deviceKey }) => {
  const config = READ_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  return { name: "Modbus Read Task", type: READ_TYPE, config };
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
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
          name: primitive.isNonZero(c.name) ? c.name : channelName(safeName, c),
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
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
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
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "Modbus Read Task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateRead,
});
