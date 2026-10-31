// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Device } from "@/feature/ethercat/device";
import {
  WriteChannelDetails,
  WriteDetailsTitle,
} from "@/feature/ethercat/task/ChannelDetails";
import {
  checkOrCreateIndex,
  findChannelsToCreate,
  retrieveAndValidateSlaves,
} from "@/feature/ethercat/task/configure";
import {
  addressKey,
  createWriteChannel,
  deployWriteConfigZ,
  type EntryWriteChannel,
  getEntryChannelName,
  getPortLabel,
  hasEntry,
  stateKey,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteSchemas,
} from "@/feature/ethercat/task/types";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <Flex.Box x grow>
    <PForm.NumericField
      path="config.executionRate"
      label="Execution rate"
      inputProps={EXECUTION_RATE_INPUT_PROPS}
    />
    <Task.Fields.StateUpdateRate />
  </Flex.Box>
);

const EXECUTION_RATE_INPUT_PROPS = { endContent: "Hz" } as const;

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<WriteChannel>(path);
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      port={getPortLabel(ch)}
      status={ch.type === "unresolved" ? "error" : undefined}
      path={path}
      channel={ch.cmdChannel}
      stateChannel={ch.stateChannel}
      hasTareButton={false}
      canTare={false}
      nameDirection="y"
      portMaxChars={20}
    />
  );
};

const channelDetails = Component.renderProp(WriteChannelDetails);

const detailsTitle = Component.renderProp(WriteDetailsTitle);

const listItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const slaves = Device.useSlavesByKeys(Task.useChannelDeviceKeys());
  const resolve = (ch: WriteChannel) => {
    const slave = slaves?.find(({ key }) => key === ch.device);
    if (slave == null || !hasEntry(ch)) return null;
    const { channels } = slave.properties.write;
    return {
      cmdChannel: channels[addressKey(ch)] ?? 0,
      stateChannel: channels[stateKey(ch)] ?? 0,
    };
  };
  return (
    <Task.Views.ListAndDetails<WriteChannel>
      listItem={listItem}
      details={channelDetails}
      detailsTitle={detailsTitle}
      createChannel={createWriteChannel}
      contextMenuItems={Task.writeChannelContextMenuItems}
      resolve={resolve}
    />
  );
};

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({ config }) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (config == null) cfg.channels = [createWriteChannel([])];
  return { name: "EtherCAT write task", type: WRITE_TYPE, config: cfg };
};

const WRITE_INDEX_OPTIONS = {
  indexProperty: "writeStateIndex" as const,
  channelsProperty: "write" as const,
  nameSuffix: "_state_time" as const,
};

const onConfigure: Task.OnConfigure<WriteSchemas["config"]> = async (
  client,
  config,
) => {
  const { slaves, rack, channelsBySlaveKey } =
    await retrieveAndValidateSlaves<WriteChannel>(client, config.channels, "outputs");

  for (const slave of slaves) {
    const channels = channelsBySlaveKey.get(slave.key) ?? [];
    let modified = await checkOrCreateIndex(client, slave, WRITE_INDEX_OPTIONS);
    const existingChannels = slave.properties.write.channels;
    // The command and state of a pair are checked apart: deleting one must recreate
    // only that one, leaving the other bound where the device already maps it.
    const cmdToCreate = await findChannelsToCreate(client, channels, existingChannels);
    const stateToCreate = await findChannelsToCreate(
      client,
      channels,
      existingChannels,
      stateKey,
    );
    const { identifier, pdos } = slave.properties;
    const baseName = (ch: EntryWriteChannel) =>
      getEntryChannelName(identifier, pdos.outputs, ch);

    if (cmdToCreate.length > 0) {
      modified = true;
      const cmdIndexes = await client.channels.create(
        cmdToCreate.map((ch) => ({
          name: primitive.isNonZero(ch.cmdChannelName)
            ? `${ch.cmdChannelName}_cmd_time`
            : `${baseName(ch)}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );

      const cmdChannels = await client.channels.create(
        cmdToCreate.map((ch, i) => ({
          name: primitive.isNonZero(ch.cmdChannelName)
            ? ch.cmdChannelName
            : `${baseName(ch)}_cmd`,
          dataType: ch.dataType,
          index: cmdIndexes[i].key,
        })),
      );

      cmdToCreate.forEach((ch, i) => {
        existingChannels[addressKey(ch)] = cmdChannels[i].key;
      });
    }

    if (stateToCreate.length > 0) {
      modified = true;
      const stateChannels = await client.channels.create(
        stateToCreate.map((ch) => ({
          name: primitive.isNonZero(ch.stateChannelName)
            ? ch.stateChannelName
            : `${baseName(ch)}_state`,
          dataType: ch.dataType,
          index: slave.properties.writeStateIndex,
        })),
      );

      stateToCreate.forEach((ch, i) => {
        existingChannels[stateKey(ch)] = stateChannels[i].key;
      });
    }

    if (modified) await client.devices.create(slave);

    channels.forEach((ch) => {
      ch.cmdChannel = existingChannels[addressKey(ch)];
      ch.stateChannel = existingChannels[stateKey(ch)];
    });
  }

  return [config, rack];
};

export const Write = Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  deployConfigZ: deployWriteConfigZ,
  type: "ethercat_write",
  getInitialValues,
  onConfigure,
});

export const useCreateWrite = Task.createUseCreate({
  getInitialValues,
});

export const WriteSelectable = Selector.createSelectable({
  type: WRITE_TYPE,
  title: "EtherCAT write task",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateWrite,
});
