// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type ethercat, query } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Channel } from "@synnaxlabs/pluto";
import { deep, primitive, unique } from "@synnaxlabs/x";
import { type FC } from "react";

import {
  useResultSlave,
  useSlavesByKeys,
  useSlavesOnRack,
} from "@/feature/ethercat/device/queries";
import { SLAVE_SCHEMAS, type SlaveDevice } from "@/feature/ethercat/device/types";
import { AddSlaveEntriesItem } from "@/feature/ethercat/task/AddSlaveEntriesItem";
import {
  ReadChannelDetails,
  ReadDetailsTitle,
} from "@/feature/ethercat/task/ChannelDetails";
import { slavesZ } from "@/feature/ethercat/task/checks";
import {
  bindOrphanedChannels,
  checkOrCreateIndex,
  createSlaveChannels,
  fillStartupParameters,
  retrieveSlaves,
} from "@/feature/ethercat/task/configure";
import { PortLabel } from "@/feature/ethercat/task/PortLabel";
import {
  bindReadChannel,
  createEntryReadChannel,
  createReadChannel,
  type DataTypes,
  deployReadConfigZ,
  entryAt,
  type EntryReadChannel,
  hasEntry,
  isCalibrated,
  mappedChannels,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  readChannelName,
  readChannelOf,
  readDataType,
  readNameSuffix,
  type ReadSchemas,
} from "@/feature/ethercat/task/types";
import { Device } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<ReadSchemas>>();

const Properties = () => (
  <Flex.Box x grow wrap>
    <Task.Fields.SampleRate />
    <Task.Fields.StreamRate />
  </Flex.Box>
);

const toDataTypes = (channels: channel.Channel[]): DataTypes =>
  new Map(channels.map((c) => [c.key, c.dataType.toString()]));

const readChannelKeys = (slaves: SlaveDevice[]): number[] =>
  unique.unique(
    slaves.flatMap((s) => s.properties.read.channels.map(({ channel }) => channel)),
  );

/**
 * @returns the data types of the read channels that the slaves map, or undefined while
 * they load.
 */
const useDataTypes = (slaves: SlaveDevice[]): DataTypes | undefined => {
  const keys = readChannelKeys(slaves);
  const { data } = Channel.useResultMultiple(
    keys.length === 0 ? null : query.keysRequest(keys),
  );
  if (keys.length === 0) return new Map();
  return data == null ? undefined : toDataTypes(data);
};

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const entry = path.config.channels[itemKey];
  const { value: ch, status } = PForm.useFieldState(entry);
  const { data: slave } = useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  const dataTypes = useDataTypes(slave == null ? [] : [slave]);
  const named =
    slave != null &&
    dataTypes != null &&
    hasEntry(ch) &&
    entryAt(slave.properties.pdos.inputs, ch) != null;
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      nameDirection="y"
      port={<PortLabel ch={ch} entries={slave?.properties.pdos.inputs ?? []} />}
      status={status.variant === "error" ? "error" : undefined}
      path={entry}
      channel={ch.channel}
      defaultName={named ? readChannelName(slave, ch, dataTypes) : undefined}
      suffix={named ? readNameSuffix(slave, ch, dataTypes) : undefined}
      skipped={ch.disabled}
      hasTareButton={false}
      canTare={false}
      portMaxChars={20}
    />
  );
};

const channelDetails = Component.renderProp(ReadChannelDetails);

const detailsTitle = Component.renderProp(ReadDetailsTitle);

const listItem = Component.renderProp(ChannelListItem);

/** @returns a channel on the entry, calibrated when the Driver calibrates the entry. */
const createSlaveReadChannel = (
  { key, properties }: SlaveDevice,
  entry: ethercat.PDOEntry,
): EntryReadChannel => ({
  ...createEntryReadChannel(key, entry),
  calibrated: isCalibrated(properties, entry),
});

const ContextMenuItems = (props: Task.ContextMenuItemProps<ReadChannel>) => (
  <>
    <AddSlaveEntriesItem
      {...props}
      path={path.config.channels}
      pdoType="inputs"
      createChannel={createSlaveReadChannel}
    />
    {Task.readChannelContextMenuItem(props)}
  </>
);

const contextMenuItems = Component.renderProp(ContextMenuItems);

const Form: FC = () => {
  const slaves = useSlavesByKeys(Task.useChannelDeviceKeys(path.config.channels));
  const onRack = useSlavesOnRack(slaves?.[0]?.rack);
  PForm.useCheck(path.config, slavesZ("inputs", slaves ?? [], onRack ?? []));
  const dataTypes = useDataTypes(slaves ?? []);
  const resolve = (ch: ReadChannel) => {
    const slave = slaves?.find(({ key }) => key === ch.device);
    if (slave == null || dataTypes == null || !hasEntry(ch)) return null;
    return { channel: readChannelOf(slave, ch, dataTypes) };
  };
  return (
    <Task.Views.ListAndDetails<ReadChannel>
      path={path.config.channels}
      listItem={listItem}
      details={channelDetails}
      detailsTitle={detailsTitle}
      createChannel={createReadChannel}
      contextMenuItems={contextMenuItems}
      resolve={resolve}
    />
  );
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = async ({
  client,
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (config == null)
    cfg.channels =
      deviceKey == null
        ? [createReadChannel([])]
        : await createSlaveChannels(
            client,
            deviceKey,
            "inputs",
            createSlaveReadChannel,
            createReadChannel([]),
          );
  return { name: "EtherCAT read task", type: READ_TYPE, config: cfg };
};

const READ_INDEX_OPTIONS = {
  indexProperty: "readIndex" as const,
  channelsProperty: "read" as const,
  nameSuffix: "_time" as const,
};

/** @throws when two new channels of the task share a name. */
const checkNamesUnique = (names: string[]): void => {
  const repeated = names.find((name, i) => names.indexOf(name) !== i);
  if (repeated != null)
    throw new Error(
      `Two channels of the task use the name ${repeated}. Choose another name for ` +
        "one of them.",
    );
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const { slaves, rack, channelsBySlaveKey } = await retrieveSlaves<ReadChannel>(
    client,
    "inputs",
    config,
  );

  // The checks run before any create, so their failure leaves nothing behind.
  config.startupParameters = fillStartupParameters(
    slaves,
    config.channels,
    config.startupParameters,
  );
  const dataTypes = toDataTypes(
    await client.channels.retrieve(readChannelKeys(slaves)),
  );
  const plans = slaves.map((slave) => {
    const channels = channelsBySlaveKey.get(slave.key) ?? [];
    const missing = channels.filter((ch) => readChannelOf(slave, ch, dataTypes) === 0);
    return { slave, channels, missing };
  });
  checkNamesUnique(
    plans.flatMap(({ slave, missing }) =>
      missing.map((ch) => readChannelName(slave, ch, dataTypes)),
    ),
  );

  for (const plan of plans) {
    const { slave, channels } = plan;
    const before = deep.copy(slave.properties);
    const nameOf = (ch: EntryReadChannel) => readChannelName(slave, ch, dataTypes);
    let modified = await checkOrCreateIndex(client, slave, READ_INDEX_OPTIONS);
    // A new index clears the read channels of the slave, so every channel is missing.
    const missing = modified ? channels : plan.missing;
    if (missing.length > 0) modified = true;
    const toCreate = await bindOrphanedChannels({
      client,
      channels: missing,
      names: missing.map(nameOf),
      index: () => slave.properties.readIndex,
      mapped: mappedChannels(slave),
      bind: (ch, key) => bindReadChannel(slave, ch, key, dataTypes),
      getDataType: readDataType,
    });

    if (toCreate.length > 0) {
      const created = await client.channels.create(
        toCreate.map((ch) => ({
          name: nameOf(ch),
          dataType: readDataType(ch),
          index: slave.properties.readIndex,
        })),
      );

      created.forEach((c, i) => bindReadChannel(slave, toCreate[i], c.key, dataTypes));
    }

    if (modified) await Device.setProperties(client, slave, before, SLAVE_SCHEMAS);

    channels.forEach((ch) => {
      ch.channel = readChannelOf(slave, ch, dataTypes);
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
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "EtherCAT read task",
  label: "Read",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateRead,
};
