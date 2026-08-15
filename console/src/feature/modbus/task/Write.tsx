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
import {
  Component,
  Flex,
  Form as PForm,
  Icon,
  Menu,
  Select,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { deep, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Select as SelectDevice } from "@/feature/modbus/device/Select";
import * as Device from "@/feature/modbus/device/types";
import { SelectWriteChannelTypeField } from "@/feature/modbus/task/SelectWriteChannelTypeField";
import {
  deployWriteConfigZ,
  WRITE_CHANNEL_SCHEMAS,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteChannelType,
  type WriteSchemas,
} from "@/feature/modbus/task/types";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const { type, channel } = PForm.useFieldValue<WriteChannel>(path);
  return (
    <Select.ListItem {...props} justify="between" align="center" x full="x">
      <Flex.Box x pack className={CSS.B("channel-item")}>
        <SelectWriteChannelTypeField
          path={path}
          onChange={(value, { get, set, path }) => {
            const prevType = get<WriteChannelType>(path).value;
            if (prevType === value) return;
            const next = WRITE_CHANNEL_SCHEMAS[value].parse({ type: value });
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<WriteChannel>(parentPath).value;
            const schema = WRITE_CHANNEL_SCHEMAS[value];
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
        {type === "holding_register" && (
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
    <Telem.SelectDataType {...props} hideVariableDensity />
  ),
);

const getOpenChannel = (channels: WriteChannel[]): WriteChannel => {
  if (channels.length === 0)
    return {
      ...WRITE_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
      key: id.create(),
    };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    key: id.create(),
    name: "",
    address: channelToCopy.address + 1,
  };
};

const listItem = Component.renderProp(ChannelListItem);

interface ContextMenuItemProps extends Task.ContextMenuItemProps<WriteChannel> {}

const ContextMenuItem: React.FC<ContextMenuItemProps> = ({ channels, keys }) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const cmdChannel = channels.find((ch) => ch.key === key)?.channel;
  if (cmdChannel == null) return null;
  const handleRename = () => Text.edit(Task.getChannelNameID(key));
  return (
    <>
      <ContextMenu.RenameItem onClick={handleRename} />
      <Menu.Divider />
    </>
  );
};

const contextMenuItems = Component.renderProp(ContextMenuItem);

const Form: FC = () => (
  <Task.Views.List<WriteChannel>
    createChannel={getOpenChannel}
    listItem={listItem}
    contextMenuItems={contextMenuItems}
  />
);

// Auto-generated channel names and device map keys keep the released type
// spellings, so channels created before the labels were renamed keep matching.
const NAME_TYPES: Record<WriteChannelType, string> = {
  coil: "coil_output",
  holding_register: "holding_register_output",
};

const writeMapKey = (channel: WriteChannel) =>
  `${NAME_TYPES[channel.type]}-${channel.address.toString()}`.replace("_", "-");

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({ deviceKey }) => {
  const config = WRITE_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  return { name: "Modbus Write Task", type: WRITE_TYPE, config };
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const commandsToCreate: WriteChannel[] = [];
  for (const channel of config.channels) {
    const key = writeMapKey(channel);
    const existing = dev.properties.write.channels[key];
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
          : `${safeName}_${NAME_TYPES[c.type]}_${c.address}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: primitive.isNonZero(c.name)
          ? c.name
          : `${safeName}_${NAME_TYPES[c.type]}_${c.address}_cmd`,
        dataType: c.type === "holding_register" ? c.dataType : "uint8",
        index: commandIndexes[i].key,
      })),
    );
    commands.forEach((c, i) => {
      const channel = commandsToCreate[i];
      dev.properties.write.channels[writeMapKey(channel)] = c.key;
    });
    await client.devices.create(dev, Device.SCHEMAS);
  }

  config.channels = config.channels.map((c) => ({
    ...c,
    channel: dev.properties.write.channels[writeMapKey(c)],
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
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "Modbus Write Task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateWrite,
});
