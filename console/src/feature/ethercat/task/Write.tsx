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
import { deep, primitive } from "@synnaxlabs/x";
import type { FC } from "react";

import {
  useResultSlave,
  useSlavesByKeys,
  useSlavesOnRack,
} from "@/feature/ethercat/device/queries";
import { needsSetup } from "@/feature/ethercat/device/SetupNote";
import { SLAVE_SCHEMAS } from "@/feature/ethercat/device/types";
import { AddSlaveEntriesItem } from "@/feature/ethercat/task/AddSlaveEntriesItem";
import {
  WriteChannelDetails,
  WriteDetailsTitle,
} from "@/feature/ethercat/task/ChannelDetails";
import { ChannelUsage } from "@/feature/ethercat/task/ChannelUsage";
import { slavesZ } from "@/feature/ethercat/task/checks";
import {
  bindOrphanedChannels,
  checkOrCreateIndex,
  findChannelsToCreate,
  retrieveByName,
  retrieveSlaves,
} from "@/feature/ethercat/task/configure";
import { createSlaveWriteChannel } from "@/feature/ethercat/task/create";
import { PortLabel } from "@/feature/ethercat/task/PortLabel";
import {
  bindWriteChannel,
  createWriteChannel,
  deployWriteConfigZ,
  entryAt,
  type EntryWriteChannel,
  getEntryChannelName,
  hasEntry,
  mappedChannels,
  WRITE_SCHEMAS,
  type WriteChannel,
  writeChannelNames,
  writeChannelOf,
  type WriteSchemas,
} from "@/feature/ethercat/task/types";
import { Device } from "@/platform/device";
import { Task } from "@/platform/task";

const path = deep.path<Task.Values<WriteSchemas>>();

const Properties = () => (
  <Flex.Box x grow wrap>
    <PForm.NumericField
      path={path.config.executionRate}
      label="Execution rate"
      inputProps={EXECUTION_RATE_INPUT_PROPS}
    />
    <Task.Fields.StateUpdateRate />
  </Flex.Box>
);

const EXECUTION_RATE_INPUT_PROPS = { endContent: "Hz" } as const;

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const entry = path.config.channels[itemKey];
  const ch = PForm.useFieldValue(entry);
  const { data: slave } = useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  const names =
    slave?.configured === true &&
    hasEntry(ch) &&
    entryAt(slave.properties.pdos.outputs, ch) != null
      ? writeChannelNames(slave, ch)
      : undefined;
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      port={<PortLabel ch={ch} entries={slave?.properties.pdos.outputs ?? []} />}
      path={entry}
      channel={ch.cmdChannel}
      defaultName={names?.cmd}
      stateChannel={ch.stateChannel}
      stateDefaultName={names?.state}
      skipped={ch.disabled}
      warned={!ch.disabled && slave != null && needsSetup(slave)}
      nameDirection="y"
      portMaxChars={20}
    />
  );
};

const channelDetails = Component.renderProp(WriteChannelDetails);

const detailsTitle = Component.renderProp(WriteDetailsTitle);

const listItem = Component.renderProp(ChannelListItem);

const ContextMenuItems = (props: Task.ContextMenuItemProps<WriteChannel>) => (
  <>
    <AddSlaveEntriesItem
      {...props}
      path={path.config.channels}
      pdoType="outputs"
      createChannel={createSlaveWriteChannel}
    />
    {Task.writeChannelContextMenuItems(props)}
  </>
);

const contextMenuItems = Component.renderProp(ContextMenuItems);

const Form: FC = () => {
  const slaves = useSlavesByKeys(Task.useChannelDeviceKeys(path.config.channels));
  const onRack = useSlavesOnRack(slaves?.[0]?.rack);
  PForm.useCheck(path.config, slavesZ("outputs", slaves ?? [], onRack ?? []));
  const resolve = (ch: WriteChannel) => {
    const slave = slaves?.find(({ key }) => key === ch.device);
    if (slave == null || !hasEntry(ch)) return null;
    return {
      cmdChannel: writeChannelOf(slave, ch, "command"),
      stateChannel: writeChannelOf(slave, ch, "state"),
    };
  };
  return (
    <Flex.Box y grow empty>
      <Task.Views.ListAndDetails<WriteChannel>
        path={path.config.channels}
        listItem={listItem}
        details={channelDetails}
        detailsTitle={detailsTitle}
        createChannel={createWriteChannel}
        contextMenuItems={contextMenuItems}
        resolve={resolve}
      />
      <ChannelUsage
        indexOptions={WRITE_INDEX_OPTIONS}
        pdoType="outputs"
        path={path.config.channels}
        slaves={slaves}
      />
    </Flex.Box>
  );
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
  const { slaves, rack, channelsBySlaveKey } = await retrieveSlaves<WriteChannel>(
    client,
    "outputs",
    config,
  );

  for (const slave of slaves) {
    const channels = channelsBySlaveKey.get(slave.key) ?? [];
    const before = deep.copy(slave.properties);
    let modified = await checkOrCreateIndex(client, slave, WRITE_INDEX_OPTIONS);
    // The command and state of a pair are checked apart: deleting one must recreate
    // only that one, leaving the other bound where the device already maps it.
    const cmdMissing = await findChannelsToCreate(client, channels, (ch) =>
      writeChannelOf(slave, ch, "command"),
    );
    const stateMissing = await findChannelsToCreate(client, channels, (ch) =>
      writeChannelOf(slave, ch, "state"),
    );
    if (cmdMissing.length > 0 || stateMissing.length > 0) modified = true;
    const { identifier, pdos } = slave.properties;
    const cmdIndexName = (ch: EntryWriteChannel) =>
      primitive.isNonZero(ch.cmdChannelName)
        ? `${ch.cmdChannelName}_cmd_time`
        : `${getEntryChannelName(identifier, pdos.outputs, ch)}_cmd_time`;
    const cmdIndexNames = cmdMissing.map(cmdIndexName);
    const orphanCmdIndexes = await retrieveByName(client, cmdIndexNames);
    const cmdToCreate = await bindOrphanedChannels({
      client,
      slave: slave.key,
      channels: cmdMissing,
      names: cmdMissing.map((ch) => writeChannelNames(slave, ch).cmd),
      // A custom name can be a live channel of another device. A default name holds
      // the slave identifier, so only it is safe to reuse.
      index: (i) =>
        primitive.isZero(cmdMissing[i].cmdChannelName)
          ? orphanCmdIndexes.get(cmdIndexNames[i])?.key
          : undefined,
      mapped: mappedChannels(slave),
      bind: (ch, key) => bindWriteChannel(slave, ch, "command", key),
    });
    const stateToCreate = await bindOrphanedChannels({
      client,
      channels: stateMissing,
      names: stateMissing.map((ch) => writeChannelNames(slave, ch).state),
      index: () => slave.properties.writeStateIndex,
      mapped: mappedChannels(slave),
      bind: (ch, key) => bindWriteChannel(slave, ch, "state", key),
    });

    if (cmdToCreate.length > 0) {
      const cmdIndexes = await client.channels.create(
        cmdToCreate.map((ch) => ({
          name: cmdIndexName(ch),
          dataType: "timestamp",
          isIndex: true,
        })),
      );

      const cmdChannels = await client.channels.create(
        cmdToCreate.map((ch, i) => ({
          name: writeChannelNames(slave, ch).cmd,
          dataType: ch.dataType,
          index: cmdIndexes[i].key,
        })),
      );

      cmdToCreate.forEach((ch, i) =>
        bindWriteChannel(slave, ch, "command", cmdChannels[i].key),
      );
    }

    if (stateToCreate.length > 0) {
      const stateChannels = await client.channels.create(
        stateToCreate.map((ch) => ({
          name: writeChannelNames(slave, ch).state,
          dataType: ch.dataType,
          index: slave.properties.writeStateIndex,
        })),
      );

      stateToCreate.forEach((ch, i) =>
        bindWriteChannel(slave, ch, "state", stateChannels[i].key),
      );
    }

    if (modified) await Device.setProperties(client, slave, before, SLAVE_SCHEMAS);

    channels.forEach((ch) => {
      ch.cmdChannel = writeChannelOf(slave, ch, "command");
      ch.stateChannel = writeChannelOf(slave, ch, "state");
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
  onConfigure,
});
