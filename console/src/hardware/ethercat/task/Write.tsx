// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon, Telem } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { type Device } from "@/hardware/ethercat/device";
import { SelectSlave } from "@/hardware/ethercat/device/SelectSlave";
import { SelectChannelModeField } from "@/hardware/ethercat/task/SelectChannelModeField";
import { SelectPDOField } from "@/hardware/ethercat/task/SelectPDOField";
import {
  AUTOMATIC_TYPE,
  type ChannelMode,
  createWriteChannel,
  getChannelByMapKey,
  getPDOName,
  getPortLabel,
  resolvePDODataType,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  writeConfigZ,
  writeMapKey,
  type writeStatusDataZ,
  type writeTypeZ,
  ZERO_WRITE_CHANNELS,
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
  const ch = PForm.useFieldValue<WriteChannel>(path);
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...props}
      port={getPortLabel(ch)}
      path={path}
      channel={ch.cmdChannel}
      stateChannel={ch.stateChannel}
      hasTareButton={false}
      canTare={false}
      portMaxChars={20}
    />
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const channelMode = PForm.useFieldValue<ChannelMode>(`${path}.type`);
  return (
    <Flex.Box y gap="medium" style={{ padding: "1rem" }}>
      <SelectSlave path={`${path}.device`} />
      <SelectChannelModeField path={path} zeroChannels={ZERO_WRITE_CHANNELS} />
      {channelMode === AUTOMATIC_TYPE ? (
        <SelectPDOField path={path} pdoType="outputs" />
      ) : (
        <>
          <Flex.Box x gap="small">
            <PForm.NumericField
              path={`${path}.index`}
              label="Index (hex)"
              inputProps={{ showDragHandle: false }}
              grow
            />
            <PForm.NumericField
              path={`${path}.subindex`}
              label="Subindex"
              inputProps={{ showDragHandle: false }}
              grow
            />
          </Flex.Box>
          <Flex.Box x gap="small">
            <PForm.NumericField
              path={`${path}.bitLength`}
              label="Bit Length"
              inputProps={{ showDragHandle: false }}
              grow
            />
            <PForm.Field<string> path={`${path}.dataType`} label="Data Type" grow>
              {({ value, onChange }) => (
                <Telem.SelectDataType
                  value={value}
                  onChange={onChange}
                  hideVariableDensity
                />
              )}
            </PForm.Field>
          </Flex.Box>
        </>
      )}
    </Flex.Box>
  );
};

const channelDetails = Component.renderProp(ChannelDetails);

const listItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ>
> = () => (
  <Common.Task.Layouts.ListAndDetails<WriteChannel>
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

const checkOrCreateStateIndex = async (
  client: Parameters<Common.Task.OnConfigure<typeof writeConfigZ>>[0],
  slave: Device.SlaveDevice,
  networkSafeName: string,
): Promise<boolean> => {
  let shouldCreate = primitive.isZero(slave.properties.writeStateIndex);
  if (!shouldCreate)
    try {
      await client.channels.retrieve(slave.properties.writeStateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreate = true;
      else throw e;
    }

  if (shouldCreate) {
    const slaveSafeName = channel.escapeInvalidName(slave.properties.name);
    const idx = await client.channels.create({
      name: `${networkSafeName}_s${slave.properties.position}_${slaveSafeName}_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    slave.properties.writeStateIndex = idx.key;
    slave.properties.write.channels = {};
    return true;
  }
  return false;
};

const onConfigure: Common.Task.OnConfigure<typeof writeConfigZ> = async (
  client,
  config,
) => {
  const slaveKeys = [...new Set(config.channels.map((ch) => ch.device))];
  if (slaveKeys.length === 0) throw new Error("No channels configured");

  const slaves = await client.devices.retrieve<
    Device.SlaveProperties,
    Device.Make,
    Device.SlaveModel
  >({ keys: slaveKeys });

  const networks = [...new Set(slaves.map((s) => s.properties.network))];
  if (networks.length > 1)
    throw new Error(
      `All slaves must be on the same network. Found: ${networks.join(", ")}`,
    );
  if (networks.length === 0 || !networks[0])
    throw new Error("No valid network found for selected slaves");

  const network = networks[0];
  const networkSafeName = channel.escapeInvalidName(network);
  const rack = slaves[0].rack;

  const channelsBySlaveKey = new Map<string, WriteChannel[]>();
  for (const ch of config.channels) {
    const existing = channelsBySlaveKey.get(ch.device) ?? [];
    existing.push(ch);
    channelsBySlaveKey.set(ch.device, existing);
  }

  for (const slave of slaves) {
    const channels = channelsBySlaveKey.get(slave.key) ?? [];

    let modified = await checkOrCreateStateIndex(client, slave, networkSafeName);

    const toCreate: WriteChannel[] = [];
    for (const ch of channels) {
      const mapKey = writeMapKey(ch);
      const existing = getChannelByMapKey(slave.properties.write.channels, mapKey);
      if (existing === 0) {
        toCreate.push(ch);
        continue;
      }
      try {
        await client.channels.retrieve(existing);
      } catch (e) {
        if (NotFoundError.matches(e)) toCreate.push(ch);
        else throw e;
      }
    }

    if (toCreate.length > 0) {
      modified = true;
      const slaveSafeName = channel.escapeInvalidName(slave.properties.name);

      // Pre-compute derived values once per channel
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
            : `${networkSafeName}_s${slave.properties.position}_${slaveSafeName}_${pdoName}_cmd_time`,
          dataType: "timestamp",
          isIndex: true,
        })),
      );

      const cmdChannels = await client.channels.create(
        channelData.map(({ ch, pdoName, dataType }, i) => ({
          name: primitive.isNonZero(ch.cmdChannelName)
            ? ch.cmdChannelName
            : `${networkSafeName}_s${slave.properties.position}_${slaveSafeName}_${pdoName}_cmd`,
          dataType,
          index: cmdIndexes[i].key,
        })),
      );

      const stateChannels = await client.channels.create(
        channelData.map(({ ch, pdoName, dataType }) => ({
          name: primitive.isNonZero(ch.stateChannelName)
            ? ch.stateChannelName
            : `${networkSafeName}_s${slave.properties.position}_${slaveSafeName}_${pdoName}_state`,
          dataType,
          index: slave.properties.writeStateIndex,
        })),
      );

      toCreate.forEach((ch, i) => {
        const mapKey = writeMapKey(ch);
        slave.properties.write.channels[mapKey] = cmdChannels[i].key;
        slave.properties.write.channels[`${mapKey}_state`] = stateChannels[i].key;
      });
    }

    if (modified) await client.devices.create(slave);

    channels.forEach((ch) => {
      const mapKey = writeMapKey(ch);
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
