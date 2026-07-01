// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/modbus/task/Task.css";

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

import { ContextMenu } from "@/component/context-menu";
import { CSS } from "@/component/css";
import { Selector } from "@/component/selector";
import { Task } from "@/component/task";
import { Device } from "@/service/modbus/device";
import { SelectOutputChannelTypeField } from "@/service/modbus/task/SelectOutputChannelTypeField";
import {
  OUTPUT_CHANNEL_SCHEMAS,
  type OutputChannel,
  type OutputChannelType,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteSchemas,
  ZERO_OUTPUT_CHANNELS,
  ZERO_WRITE_PAYLOAD,
} from "@/service/modbus/task/types";
import { Task as ServiceTask } from "@/service/task";

export const WRITE_LAYOUT = {
  ...ServiceTask.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.Modbus",
} as const satisfies ServiceTask.Layout;

export const WriteSelectable = Selector.createSimpleItem({
  title: "Modbus Write Task",
  icon: <Icon.Logo.Modbus />,
  layout: WRITE_LAYOUT,
});

const Properties = () => (
  <>
    <Device.Select />
    <Task.Fields.AutoStart />
  </>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const { type, channel } = PForm.useFieldValue<OutputChannel>(path);
  return (
    <Select.ListItem {...props} justify="between" align="center" x full="x">
      <Flex.Box x pack className={CSS.B("channel-item")}>
        <SelectOutputChannelTypeField
          path={path}
          onChange={(value, { get, set, path }) => {
            const prevType = get<OutputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(ZERO_OUTPUT_CHANNELS[value]);
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<OutputChannel>(parentPath).value;
            const schema = OUTPUT_CHANNEL_SCHEMAS[value];
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
        {type === "holding_register_output" && (
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
    <Telem.SelectDataType {...props} hideVariableDensity />
  ),
);

const getOpenChannel = (channels: OutputChannel[]): OutputChannel => {
  if (channels.length === 0)
    return {
      type: "coil_output",
      address: 0,
      channel: 0,
      enabled: true,
      key: id.create(),
      name: "",
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

interface ContextMenuItemProps
  extends Task.ContextMenuItemProps<OutputChannel> {}

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

const Form: FC<Task.FormProps<WriteSchemas>> = () => (
  <Task.Layouts.List<OutputChannel>
    createChannel={getOpenChannel}
    listItem={listItem}
    contextMenuItems={contextMenuItems}
  />
);

const writeMapKey = (channel: OutputChannel) =>
  `${channel.type}-${channel.address.toString()}`.replace("_", "-");

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
}) => ({
  ...ZERO_WRITE_PAYLOAD,
  config: {
    ...ZERO_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  const commandsToCreate: OutputChannel[] = [];
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
          : `${safeName}_${c.type}_${c.address}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: primitive.isNonZero(c.name)
          ? c.name
          : `${safeName}_${c.type}_${c.address}_cmd`,
        dataType: c.type === "holding_register_output" ? c.dataType : "uint8",
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

export const Write = ServiceTask.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  type: "modbus_write",
  getInitialValues,
  onConfigure,
});
