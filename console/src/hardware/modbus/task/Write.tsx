// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/modbus/task/Task.css";

import { NotFoundError } from "@synnaxlabs/client";
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
import { caseconv, deep, id } from "@synnaxlabs/x";
import { type FC } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/modbus/device";
import { SelectOutputChannelTypeField } from "@/hardware/modbus/task/SelectOutputChannelTypeField";
import {
  HOLDING_REGISTER_OUTPUT_TYPE,
  OUTPUT_CHANNEL_SCHEMAS,
  type OutputChannel,
  type OutputChannelType,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type writeConfigZ,
  type writeStatusDataZ,
  type writeTypeZ,
  ZERO_OUTPUT_CHANNELS,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/modbus/task/types";
import { type Selector } from "@/selector";

export const WRITE_LAYOUT = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.Modbus",
} as const satisfies Common.Task.Layout;

export const WRITE_SELECTABLE = {
  key: WRITE_TYPE,
  title: "Modbus Write Task",
  icon: <Icon.Logo.Modbus />,
  create: async ({ layoutKey }) => ({ ...WRITE_LAYOUT, key: layoutKey }),
} as const satisfies Selector.Selectable;

const Properties = () => (
  <>
    <Device.Select />
    <Common.Task.Fields.AutoStart />
  </>
);

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
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
        {type === HOLDING_REGISTER_OUTPUT_TYPE && (
          <PForm.Field<string>
            path={`${path}.dataType`}
            showLabel={false}
            showHelpText={false}
            hideIfNull
          >
            {({ value, onChange }) => (
              <Telem.SelectDataType
                value={value}
                onChange={onChange}
                hideVariableDensity
              />
            )}
          </PForm.Field>
        )}
      </Flex.Box>
      <Flex.Box x align="center" grow justify="end">
        <Common.Task.ChannelName
          channel={channel}
          id={Common.Task.getChannelNameID(itemKey)}
        />
        <Common.Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const getOpenChannel = (channels: OutputChannel[]): OutputChannel => {
  if (channels.length === 0)
    return {
      type: "coil_output",
      address: 0,
      channel: 0,
      enabled: true,
      key: id.create(),
    };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    key: id.create(),
    address: channelToCopy.address + 1,
  };
};

const listItem = Component.renderProp(ChannelListItem);

interface ContextMenuItemProps
  extends Common.Task.ContextMenuItemProps<OutputChannel> {}

const ContextMenuItem: React.FC<ContextMenuItemProps> = ({ channels, keys }) => {
  if (keys.length !== 1) return null;
  const key = keys[0];
  const cmdChannel = channels.find((ch) => ch.key === key)?.channel;
  if (cmdChannel == null || cmdChannel == 0) return null;
  const handleRename = () => Text.edit(Common.Task.getChannelNameID(key));
  return (
    <>
      <Menu.Item itemKey="rename" onClick={handleRename}>
        <Icon.Rename />
        Rename
      </Menu.Item>
      <Menu.Divider />
    </>
  );
};

const contextMenuItems = Component.renderProp(ContextMenuItem);

const Form: FC<
  Common.Task.FormProps<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ>
> = () => (
  <Common.Task.Layouts.List<OutputChannel>
    createChannel={getOpenChannel}
    listItem={listItem}
    contextMenuItems={contextMenuItems}
  />
);

const writeMapKey = (channel: OutputChannel) =>
  `${channel.type}-${channel.address.toString()}`.replace("_", "-");

const getInitialValues: Common.Task.GetInitialValues<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> = ({ deviceKey }) => ({
  ...ZERO_WRITE_PAYLOAD,
  config: {
    ...ZERO_WRITE_PAYLOAD.config,
    device: deviceKey ?? ZERO_WRITE_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<typeof writeConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve<Device.Properties>({
    key: config.device,
  });
  const commandsToCreate: OutputChannel[] = [];
  for (const channel of config.channels) {
    const key = writeMapKey(channel);
    const existing =
      dev.properties.write.channels[key] ??
      dev.properties.write.channels[caseconv.snakeToCamel(key)];
    if (existing == null) {
      commandsToCreate.push(channel);
      continue;
    }
    try {
      await client.channels.retrieve(existing);
    } catch (e) {
      if (NotFoundError.matches(e)) commandsToCreate.push(channel);
      else throw e;
    }
  }

  if (commandsToCreate.length > 0) {
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.name}_${c.type}_${c.address}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );
    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.name}_${c.type}_${c.address}_cmd`,
        dataType: c.type === "holding_register_output" ? c.dataType : "uint8",
        index: commandIndexes[i].key,
      })),
    );
    commands.forEach((c, i) => {
      const channel = commandsToCreate[i];
      dev.properties.write.channels[writeMapKey(channel)] = c.key;
    });
    await client.devices.create(dev);
  }

  config.channels = config.channels.map((c) => ({
    ...c,
    channel:
      dev.properties.write.channels[writeMapKey(c)] ??
      dev.properties.write.channels[caseconv.snakeToCamel(writeMapKey(c))],
  }));

  return [config, dev.rack];
};

export const Write = Common.Task.wrapForm({
  initialStatusData: null,
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
