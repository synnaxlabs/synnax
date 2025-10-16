// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon, Input, Select, Text } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { type FC } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/visa/device";
import {
  type OutputChannel,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type writeConfigZ,
  writeMapKey,
  type writeStatusDataZ,
  type writeTypeZ,
  ZERO_OUTPUT_CHANNEL,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/visa/task/types";
import { type Selector } from "@/selector";

export const WRITE_LAYOUT = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Chip",
} as const satisfies Common.Task.Layout;

export const WRITE_SELECTABLE = {
  key: WRITE_TYPE,
  title: "VISA Write Task",
  icon: <Icon.Chip />,
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
  const { channel, commandTemplate } = PForm.useFieldValue<OutputChannel>(path);

  // Preview the command with an example value
  const commandPreview = commandTemplate?.replace("{value}", "1.234") || "";
  const hasPlaceholder = commandTemplate?.includes("{value}");

  return (
    <Select.ListItem
      {...props}
      justify="between"
      align="center"
      direction="y"
      full="x"
    >
      <Flex.Box x pack className={CSS.B("channel-item")} align="center" grow>
        <PForm.Field<string>
          path={`${path}.scpiCommand`}
          showLabel={false}
          showHelpText={false}
          hideIfNull
        >
          {(p) => (
            <Input.Text
              placeholder="SOUR:VOLT? (query for feedback)"
              style={{ minWidth: 180 }}
              {...p}
            />
          )}
        </PForm.Field>
        <PForm.Field<string>
          path={`${path}.commandTemplate`}
          showLabel={false}
          showHelpText={false}
          hideIfNull
        >
          {(p) => (
            <Flex.Box direction="y" style={{ minWidth: 220 }}>
              <Input.Text
                placeholder="SOUR:VOLT {value}"
                {...p}
              />
              {commandTemplate && (
                <Text.Text
                  level="small"
                  style={{
                    marginTop: 2,
                    opacity: hasPlaceholder ? 0.6 : 1,
                    color: hasPlaceholder ? undefined : "var(--pluto-warning-z)",
                  }}
                >
                  {hasPlaceholder ? (
                    `Preview: ${commandPreview}`
                  ) : (
                    "⚠️ Missing {value} placeholder"
                  )}
                </Text.Text>
              )}
            </Flex.Box>
          )}
        </PForm.Field>
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
  if (channels.length === 0) return { ...ZERO_OUTPUT_CHANNEL, key: id.create() };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    key: id.create(),
  };
};

const listItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ>
> = () => (
  <Common.Task.Layouts.List<OutputChannel>
    createChannel={getOpenChannel}
    listItem={listItem}
  />
);

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
  const dev = await client.hardware.devices.retrieve<
    Device.Properties,
    Device.Make,
    Device.Model
  >({
    key: config.device,
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
      else throw e;
    }
  }

  if (commandsToCreate.length > 0) {
    const commandIndexes = await client.channels.create(
      commandsToCreate.map((c) => ({
        name: `${dev.name}_${c.scpiCommand.replace(/[^a-zA-Z0-9]/g, "_")}_cmd_time`,
        dataType: "timestamp",
        isIndex: true,
      })),
    );

    const commands = await client.channels.create(
      commandsToCreate.map((c, i) => ({
        name: `${dev.name}_${c.scpiCommand.replace(/[^a-zA-Z0-9]/g, "_")}_cmd`,
        dataType: "float64",
        index: commandIndexes[i].key,
      })),
    );

    commands.forEach((c, i) => {
      const channel = commandsToCreate[i];
      dev.properties.write.channels[writeMapKey(channel)] = c.key;
    });

    await client.hardware.devices.create(dev);
  }

  config.channels = config.channels.map((c) => ({
    ...c,
    channel: dev.properties.write.channels[writeMapKey(c)],
  }));

  return [config, dev.rack];
};

export const Write = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
