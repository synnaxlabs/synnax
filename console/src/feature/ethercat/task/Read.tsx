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

import { ReadChannelDetails } from "@/feature/ethercat/task/ChannelDetails";
import {
  checkOrCreateIndex,
  findChannelsToCreate,
  retrieveAndValidateSlaves,
} from "@/feature/ethercat/task/configure";
import {
  channelMapKey,
  createInputChannel,
  deployReadConfigZ,
  getChannelByMapKey,
  getPDOName,
  getPortLabel,
  type InputChannel,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadSchemas,
  resolvePDODataType,
  ZERO_READ_PAYLOAD,
} from "@/feature/ethercat/task/types";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <Flex.Box x grow>
    <Task.Fields.SampleRate />
    <Task.Fields.StreamRate />
    <Task.Fields.DataSaving />
    <Task.Fields.AutoStart />
  </Flex.Box>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<InputChannel>(path);
  return (
    <Task.Views.ListAndDetailsChannelItem
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

const Form: FC<Task.FormProps<ReadSchemas>> = () => (
  <Task.Views.ListAndDetails<InputChannel>
    listItem={listItem}
    details={channelDetails}
    createChannel={createInputChannel}
    contextMenuItems={Task.readChannelContextMenuItem}
  />
);

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ config }) => {
  if (config != null)
    return { ...ZERO_READ_PAYLOAD, config: READ_SCHEMAS.config.parse(config) };
  return { ...ZERO_READ_PAYLOAD };
};

const READ_INDEX_OPTIONS = {
  indexProperty: "readIndex" as const,
  channelsProperty: "read" as const,
  nameSuffix: "_time" as const,
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
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
            ch.type === "automatic"
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

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "ethercat_read",
  getInitialValues,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const readIngester = Task.createIngester({
  getInitialValues,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "EtherCAT Read Task",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateRead,
});
