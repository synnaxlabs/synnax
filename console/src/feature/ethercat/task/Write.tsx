// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { useSlavesByKeys } from "@/feature/ethercat/device/queries";
import { WriteChannelDetails } from "@/feature/ethercat/task/ChannelDetails";
import {
  checkOrCreateIndex,
  findChannelsToCreate,
  retrieveAndValidateSlaves,
} from "@/feature/ethercat/task/configure";
import {
  channelMapKey,
  createWriteChannel,
  deployWriteConfigZ,
  getChannelByMapKey,
  getPDOName,
  getPortLabel,
  resolvePDODataType,
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
    <Task.Fields.DataSaving />
    <Task.Fields.AutoStart />
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

const listItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const slaves = useSlavesByKeys(Task.useChannelDeviceKeys());
  const resolve = useCallback(
    (ch: WriteChannel) => {
      const slave = slaves?.find(({ key }) => key === ch.device);
      if (slave == null) return null;
      const { channels } = slave.properties.write;
      const mapKey = channelMapKey(ch);
      return {
        cmdChannel: getChannelByMapKey(channels, mapKey),
        stateChannel: getChannelByMapKey(channels, `${mapKey}${STATE_KEY_SUFFIX}`),
      };
    },
    [slaves],
  );
  return (
    <Task.Views.ListAndDetails<WriteChannel>
      listItem={listItem}
      details={channelDetails}
      createChannel={createWriteChannel}
      contextMenuItems={Task.writeChannelContextMenuItems}
      resolve={resolve}
    />
  );
};

const getInitialValues: Task.GetInitialValues<WriteSchemas> = ({ config }) => ({
  name: "EtherCAT write task",
  type: WRITE_TYPE,
  config: WRITE_SCHEMAS.config.parse(config ?? {}),
});

// The state half of a pair lives beside its command under this suffix.
const STATE_KEY_SUFFIX = "_state";

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
    await retrieveAndValidateSlaves<WriteChannel>(client, config.channels);

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
      STATE_KEY_SUFFIX,
    );
    const identifier = channel.escapeInvalidName(slave.properties.identifier);
    const describe = (ch: WriteChannel) => ({
      ch,
      pdoName: getPDOName(ch),
      dataType:
        ch.type === "automatic"
          ? resolvePDODataType(slave, ch.pdo, "outputs")
          : ch.dataType,
    });

    if (cmdToCreate.length > 0) {
      modified = true;
      const channelData = cmdToCreate.map(describe);

      const cmdIndexes = await client.channels.create(
        channelData.map(({ ch, pdoName }) => ({
          name: primitive.isNonZero(ch.cmdChannelName)
            ? `${ch.cmdChannelName}_cmd_time`
            : `${identifier}_${pdoName}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );

      const cmdChannels = await client.channels.create(
        channelData.map(({ ch, pdoName, dataType }, i) => ({
          name: primitive.isNonZero(ch.cmdChannelName)
            ? ch.cmdChannelName
            : `${identifier}_${pdoName}_cmd`,
          dataType,
          index: cmdIndexes[i].key,
        })),
      );

      cmdToCreate.forEach((ch, i) => {
        existingChannels[channelMapKey(ch)] = cmdChannels[i].key;
      });
    }

    if (stateToCreate.length > 0) {
      modified = true;
      const stateChannels = await client.channels.create(
        stateToCreate.map(describe).map(({ ch, pdoName, dataType }) => ({
          name: primitive.isNonZero(ch.stateChannelName)
            ? ch.stateChannelName
            : `${identifier}_${pdoName}_state`,
          dataType,
          index: slave.properties.writeStateIndex,
        })),
      );

      stateToCreate.forEach((ch, i) => {
        existingChannels[`${channelMapKey(ch)}${STATE_KEY_SUFFIX}`] =
          stateChannels[i].key;
      });
    }

    if (modified) await client.devices.create(slave);

    channels.forEach((ch) => {
      const mapKey = channelMapKey(ch);
      ch.cmdChannel = getChannelByMapKey(slave.properties.write.channels, mapKey);
      ch.stateChannel = getChannelByMapKey(
        slave.properties.write.channels,
        `${mapKey}${STATE_KEY_SUFFIX}`,
      );
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
