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

import { useSlavesByKeys } from "@/feature/ethercat/device/queries";
import {
  ReadChannelDetails,
  ReadDetailsTitle,
} from "@/feature/ethercat/task/ChannelDetails";
import {
  checkOrCreateIndex,
  findChannelsToCreate,
  retrieveAndValidateSlaves,
} from "@/feature/ethercat/task/configure";
import {
  addressKey,
  createReadChannel,
  deployReadConfigZ,
  getEntryChannelName,
  getPortLabel,
  hasEntry,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadSchemas,
} from "@/feature/ethercat/task/types";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <Flex.Box x grow>
    <Task.Fields.SampleRate />
    <Task.Fields.StreamRate />
  </Flex.Box>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<ReadChannel>(path);
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      nameDirection="y"
      port={getPortLabel(ch)}
      status={ch.type === "unresolved" ? "error" : undefined}
      path={path}
      channel={ch.channel}
      hasTareButton={false}
      canTare={false}
      portMaxChars={10}
    />
  );
};

const channelDetails = Component.renderProp(ReadChannelDetails);

const detailsTitle = Component.renderProp(ReadDetailsTitle);

const listItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const slaves = useSlavesByKeys(Task.useChannelDeviceKeys());
  const resolve = (ch: ReadChannel) => {
    const slave = slaves?.find(({ key }) => key === ch.device);
    if (slave == null || !hasEntry(ch)) return null;
    return { channel: slave.properties.read.channels[addressKey(ch)] ?? 0 };
  };
  return (
    <Task.Views.ListAndDetails<ReadChannel>
      listItem={listItem}
      details={channelDetails}
      detailsTitle={detailsTitle}
      createChannel={createReadChannel}
      contextMenuItems={Task.readChannelContextMenuItem}
      resolve={resolve}
    />
  );
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ config }) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (config == null) cfg.channels = [createReadChannel([])];
  return { name: "EtherCAT read task", type: READ_TYPE, config: cfg };
};

const READ_INDEX_OPTIONS = {
  indexProperty: "readIndex" as const,
  channelsProperty: "read" as const,
  nameSuffix: "_time" as const,
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const { slaves, rack, channelsBySlaveKey } =
    await retrieveAndValidateSlaves<ReadChannel>(client, config.channels, "inputs");

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
      const { identifier, pdos } = slave.properties;
      const created = await client.channels.create(
        toCreate.map((ch) => ({
          name: primitive.isNonZero(ch.name)
            ? ch.name
            : getEntryChannelName(identifier, pdos.inputs, ch),
          dataType: ch.dataType,
          index: slave.properties.readIndex,
        })),
      );

      created.forEach((c, i) => {
        slave.properties.read.channels[addressKey(toCreate[i])] = c.key;
      });
    }

    if (modified) await client.devices.create(slave);

    channels.forEach((ch) => {
      ch.channel = slave.properties.read.channels[addressKey(ch)];
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

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "EtherCAT read task",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateRead,
});
