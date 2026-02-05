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
import { ReadChannelDetails } from "@/hardware/ethercat/task/ChannelDetails";
import {
  checkOrCreateIndex,
  findChannelsToCreate,
  retrieveAndValidateSlaves,
} from "@/hardware/ethercat/task/configure";
import {
  AUTOMATIC_TYPE,
  channelMapKey,
  createReadChannel,
  getChannelByMapKey,
  getPDOName,
  getPortLabel,
  type InputChannel,
  READ_SCHEMAS,
  READ_TYPE,
  readConfigZ,
  type readStatusDataZ,
  type readTypeZ,
  resolvePDODataType,
  ZERO_READ_PAYLOAD,
} from "@/hardware/ethercat/task/types";
import { Selector } from "@/selector";

export const READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.EtherCAT",
};

export const ReadSelectable = Selector.createSimpleItem({
  title: "EtherCAT Read Task",
  icon: <Icon.Logo.EtherCAT />,
  layout: READ_LAYOUT,
});

const Properties = () => (
  <Flex.Box x grow>
    <Common.Task.Fields.SampleRate />
    <Common.Task.Fields.StreamRate />
    <Common.Task.Fields.DataSaving />
    <Common.Task.Fields.AutoStart />
  </Flex.Box>
);

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<InputChannel>(path);
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...props}
      nameDirection="y"
      port={getPortLabel(ch)}
      path={path}
      channel={ch.channel}
      hasTareButton={false}
      canTare={false}
      portMaxChars={10}
    />
  );
};

const channelDetails = Component.renderProp(ReadChannelDetails);

const listItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ>
> = () => (
  <Common.Task.Layouts.ListAndDetails<InputChannel>
    listItem={listItem}
    details={channelDetails}
    createChannel={createReadChannel}
    contextMenuItems={Common.Task.readChannelContextMenuItem}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = ({ config }) => {
  if (config != null)
    return {
      ...ZERO_READ_PAYLOAD,
      config: readConfigZ.parse(config),
    };
  return { ...ZERO_READ_PAYLOAD };
};

const READ_INDEX_OPTIONS = {
  indexProperty: "readIndex" as const,
  channelsProperty: "read" as const,
  nameSuffix: "_time" as const,
};

const onConfigure: Common.Task.OnConfigure<typeof readConfigZ> = async (
  client,
  config,
) => {
  const { slaves, rack, channelsBySlaveKey } =
    await retrieveAndValidateSlaves<InputChannel>(client, config.channels);

  for (const slave of slaves) {
    const channels = channelsBySlaveKey.get(slave.key) ?? [];
    let modified = await checkOrCreateIndex(client, slave, READ_INDEX_OPTIONS);
    const toCreate = await findChannelsToCreate(
      client,
      channels,
      slave.properties.read.channels,
    );

    if (toCreate.length > 0) {
      modified = true;
      const identifier = channel.escapeInvalidName(slave.properties.identifier);
      const created = await client.channels.create(
        toCreate.map((ch) => ({
          name: primitive.isNonZero(ch.name)
            ? ch.name
            : `${identifier}_${getPDOName(ch)}`,
          dataType:
            ch.type === AUTOMATIC_TYPE
              ? resolvePDODataType(slave, ch.pdo, "inputs")
              : ch.dataType,
          index: slave.properties.readIndex,
        })),
      );

      created.forEach((c, i) => {
        slave.properties.read.channels[channelMapKey(toCreate[i])] = c.key;
      });
    }

    if (modified) await client.devices.create(slave);

    channels.forEach((ch) => {
      ch.channel = getChannelByMapKey(
        slave.properties.read.channels,
        channelMapKey(ch),
      );
    });
  }

  return [config, rack];
};

export const Read = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  type: READ_TYPE,
  getInitialValues,
  onConfigure,
});
