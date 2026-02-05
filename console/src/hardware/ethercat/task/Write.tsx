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
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { WriteChannelDetails } from "@/hardware/ethercat/task/ChannelDetails";
import {
  checkOrCreateIndex,
  findChannelsToCreate,
  retrieveAndValidateSlaves,
} from "@/hardware/ethercat/task/configure";
import {
  AUTOMATIC_TYPE,
  channelMapKey,
  createWriteChannel,
  getChannelByMapKey,
  getPDOName,
  getPortLabel,
  type OutputChannel,
  resolvePDODataType,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  writeConfigZ,
  type writeStatusDataZ,
  type writeTypeZ,
  ZERO_WRITE_PAYLOAD,
} from "@/hardware/ethercat/task/types";
import { Selector } from "@/selector";

export const WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: WRITE_TYPE,
  name: ZERO_WRITE_PAYLOAD.name,
  icon: "Logo.EtherCAT",
};

export const WriteSelectable = Selector.createSimpleItem({
  title: "EtherCAT Write Task",
  icon: <Icon.Logo.EtherCAT />,
  layout: WRITE_LAYOUT,
});

const Properties = () => (
  <Flex.Box x grow>
    <PForm.NumericField
      path="config.executionRate"
      label="Execution Rate"
      inputProps={{ endContent: "Hz" }}
    />
    <Common.Task.Fields.StateUpdateRate />
    <Common.Task.Fields.AutoStart />
  </Flex.Box>
);

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<OutputChannel>(path);
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
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

const Form: FC<
  Common.Task.FormProps<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ>
> = () => (
  <Common.Task.Layouts.ListAndDetails<OutputChannel>
    listItem={listItem}
    details={channelDetails}
    createChannel={createWriteChannel}
    contextMenuItems={Common.Task.writeChannelContextMenuItems}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> = ({ config }) => {
  if (config != null)
    return {
      ...ZERO_WRITE_PAYLOAD,
      config: writeConfigZ.parse(config),
    };
  return { ...ZERO_WRITE_PAYLOAD };
};

const WRITE_INDEX_OPTIONS = {
  indexProperty: "writeStateIndex" as const,
  channelsProperty: "write" as const,
  nameSuffix: "_state_time" as const,
};

const onConfigure: Common.Task.OnConfigure<typeof writeConfigZ> = async (
  client,
  config,
) => {
  const { slaves, rack, channelsBySlaveKey } =
    await retrieveAndValidateSlaves<OutputChannel>(client, config.channels);

  for (const slave of slaves) {
    const channels = channelsBySlaveKey.get(slave.key) ?? [];
    let modified = await checkOrCreateIndex(client, slave, WRITE_INDEX_OPTIONS);
    const toCreate = await findChannelsToCreate(
      client,
      channels,
      slave.properties.write.channels,
    );

    if (toCreate.length > 0) {
      modified = true;
      const identifier = channel.escapeInvalidName(slave.properties.identifier);

      const channelData = toCreate.map((ch) => ({
        ch,
        pdoName: getPDOName(ch),
        dataType:
          ch.type === AUTOMATIC_TYPE
            ? resolvePDODataType(slave, ch.pdo, "outputs")
            : ch.dataType,
      }));

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

      const stateChannels = await client.channels.create(
        channelData.map(({ ch, pdoName, dataType }) => ({
          name: primitive.isNonZero(ch.stateChannelName)
            ? ch.stateChannelName
            : `${identifier}_${pdoName}_state`,
          dataType,
          index: slave.properties.writeStateIndex,
        })),
      );

      toCreate.forEach((ch, i) => {
        const mapKey = channelMapKey(ch);
        slave.properties.write.channels[mapKey] = cmdChannels[i].key;
        slave.properties.write.channels[`${mapKey}_state`] = stateChannels[i].key;
      });
    }

    if (modified) await client.devices.create(slave);

    channels.forEach((ch) => {
      const mapKey = channelMapKey(ch);
      ch.cmdChannel = getChannelByMapKey(slave.properties.write.channels, mapKey);
      ch.stateChannel = getChannelByMapKey(
        slave.properties.write.channels,
        `${mapKey}_state`,
      );
    });
  }

  return [config, rack];
};

export const Write = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
