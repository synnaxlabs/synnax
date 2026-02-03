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
  createReadChannel,
  getChannelByMapKey,
  getPDOName,
  getPortLabel,
  type InputChannel,
  READ_SCHEMAS,
  READ_TYPE,
  readConfigZ,
  readMapKey,
  type readStatusDataZ,
  type readTypeZ,
  resolvePDODataType,
  ZERO_READ_CHANNELS,
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

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const channelMode = PForm.useFieldValue<ChannelMode>(`${path}.type`);
  return (
    <Flex.Box y gap="medium" style={{ padding: "1rem" }}>
      <SelectSlave path={`${path}.device`} />
      <SelectChannelModeField path={path} zeroChannels={ZERO_READ_CHANNELS} />
      {channelMode === AUTOMATIC_TYPE ? (
        <SelectPDOField path={path} pdoType="inputs" />
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

const checkOrCreateIndex = async (
  client: Parameters<Common.Task.OnConfigure<typeof readConfigZ>>[0],
  slave: Device.SlaveDevice,
  networkSafeName: string,
): Promise<boolean> => {
  let shouldCreate = primitive.isZero(slave.properties.readIndex);
  if (!shouldCreate)
    try {
      await client.channels.retrieve(slave.properties.readIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreate = true;
      else throw e;
    }

  if (shouldCreate) {
    const slaveSafeName = channel.escapeInvalidName(slave.properties.name);
    const idx = await client.channels.create({
      name: `${networkSafeName}_s${slave.properties.position}_${slaveSafeName}_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    slave.properties.readIndex = idx.key;
    slave.properties.read.channels = {};
    return true;
  }
  return false;
};

const onConfigure: Common.Task.OnConfigure<typeof readConfigZ> = async (
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

  const channelsBySlaveKey = new Map<string, InputChannel[]>();
  for (const ch of config.channels) {
    const existing = channelsBySlaveKey.get(ch.device) ?? [];
    existing.push(ch);
    channelsBySlaveKey.set(ch.device, existing);
  }

  for (const slave of slaves) {
    const channels = channelsBySlaveKey.get(slave.key) ?? [];

    let modified = await checkOrCreateIndex(client, slave, networkSafeName);

    const toCreate: InputChannel[] = [];
    for (const ch of channels) {
      const mapKey = readMapKey(ch);
      const existing = getChannelByMapKey(slave.properties.read.channels, mapKey);
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
      const created = await client.channels.create(
        toCreate.map((ch) => {
          const dataType =
            ch.type === AUTOMATIC_TYPE
              ? resolvePDODataType(slave, ch.pdo, "inputs")
              : ch.dataType;
          return {
            name: primitive.isNonZero(ch.name)
              ? ch.name
              : `${networkSafeName}_s${slave.properties.position}_${slaveSafeName}_${getPDOName(ch)}`,
            dataType,
            index: slave.properties.readIndex,
          };
        }),
      );

      created.forEach((c, i) => {
        slave.properties.read.channels[readMapKey(toCreate[i])] = c.key;
      });
    }

    if (modified) await client.devices.create(slave);

    channels.forEach((ch) => {
      ch.channel = getChannelByMapKey(slave.properties.read.channels, readMapKey(ch));
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
