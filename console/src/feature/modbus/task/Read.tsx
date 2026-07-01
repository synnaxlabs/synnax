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

import { Device } from "@/feature/modbus/device";
import { SelectInputChannelTypeField } from "@/feature/modbus/task/SelectInputChannelTypeField";
import {
  INPUT_CHANNEL_SCHEMAS,
  type InputChannel,
  type InputChannelType,
  isVariableDensityInputChannel,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadSchemas,
  type TypedInput,
  ZERO_INPUT_CHANNELS,
  ZERO_READ_PAYLOAD,
} from "@/feature/modbus/task/types";
import { Task as ServiceTask } from "@/feature/task";
import { CSS } from "@/primitive/css";
import { Selector } from "@/primitive/selector";
import { Task } from "@/primitive/task";

export const READ_LAYOUT = {
  ...ServiceTask.LAYOUT,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.Modbus",
} as const satisfies ServiceTask.Layout;

export const ReadSelectable = Selector.createSimpleItem({
  title: "Modbus Read Task",
  icon: <Icon.Logo.Modbus />,
  layout: READ_LAYOUT,
});

const Properties = () => (
  <>
    <Device.Select />
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
  const { type, channel } = PForm.useFieldValue<InputChannel>(path);
  return (
    <Select.ListItem
      {...props}
      style={{ width: "100%" }}
      justify="between"
      align="center"
      x
    >
      <Flex.Box x pack className={CSS.B("channel-item")}>
        <SelectInputChannelTypeField
          path={path}
          onChange={(value, { get, set, path }) => {
            const prevType = get<InputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(ZERO_INPUT_CHANNELS[value]);
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<InputChannel>(parentPath).value;
            const schema = INPUT_CHANNEL_SCHEMAS[value];
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
        {(type === "register_input" || type === "holding_register_input") && (
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
        <Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const renderTelemSelectDataType = Component.renderProp(
  (props: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...props} hideVariableDensity location="bottom" />
  ),
);

const getOpenChannel = (channels: InputChannel[]): InputChannel => {
  if (channels.length === 0)
    return {
      type: "coil_input",
      address: 0,
      channel: 0,
      key: id.create(),
      enabled: true,
      name: "",
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

const Form: FC<Task.FormProps<ReadSchemas>> = () => (
  <Task.Layouts.List<InputChannel>
    createChannel={getOpenChannel}
    contextMenuItems={Task.readChannelContextMenuItem}
    listItem={listItem}
  />
);

const readMapKey = (channel: InputChannel) => {
  let s = `${channel.type}-${channel.address.toString()}`;
  if (isVariableDensityInputChannel(channel)) s += `-${channel.dataType}`;
  return s.replaceAll("_", "-");
};

const channelName = (deviceName: string, channel: InputChannel) => {
  let s = `${deviceName}_${channel.type}_${channel.address}`;
  if (isVariableDensityInputChannel(channel))
    s += `_${new DataType(channel.dataType).toString(true)}`;
  return s;
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ deviceKey }) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

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

    const toCreate: InputChannel[] = [];
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
          dataType: (c as TypedInput).dataType ?? DataType.UINT8.toString(),
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

export const Read = ServiceTask.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  type: "modbus_read",
  getInitialValues,
  onConfigure,
});
