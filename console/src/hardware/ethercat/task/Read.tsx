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
import { caseconv, deep, primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ethercat/device";
import { SelectSlave } from "@/hardware/ethercat/device/SelectSlave";
import { SelectChannelModeField } from "@/hardware/ethercat/task/SelectChannelModeField";
import { SelectPDOField } from "@/hardware/ethercat/task/SelectPDOField";
import {
  AUTOMATIC_TYPE,
  type ChannelMode,
  createReadChannel,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  readConfigZ,
  readMapKey,
  type readStatusDataZ,
  type readTypeZ,
  ZERO_READ_CHANNELS,
  ZERO_READ_PAYLOAD,
} from "@/hardware/ethercat/task/types";
import { Selector } from "@/selector";

const getChannelByMapKey = (channels: Record<string, number>, mapKey: string): number =>
  channels[mapKey] ?? channels[caseconv.snakeToCamel(mapKey)] ?? 0;

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
  <>
    <Device.Select />
    <Flex.Box x grow>
      <Common.Task.Fields.SampleRate />
      <Common.Task.Fields.StreamRate />
      <Common.Task.Fields.DataSaving />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<ReadChannel>(path);

  const portLabel =
    ch.type === AUTOMATIC_TYPE
      ? ch.pdo || "No PDO"
      : `0x${ch.index.toString(16).padStart(4, "0")}:${ch.subindex}`;

  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...props}
      port={portLabel}
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
      <SelectChannelModeField
        path={path}
        onChange={(value, { get, set, path: fieldPath }) => {
          const prevType = get<ChannelMode>(fieldPath).value;
          if (prevType === value) return;
          const parentPath = fieldPath.slice(0, fieldPath.lastIndexOf("."));
          const prevParent = get<ReadChannel>(parentPath).value;
          const next = deep.copy(ZERO_READ_CHANNELS[value]);
          set(parentPath, {
            ...next,
            key: prevParent.key,
            device: prevParent.device,
            name: prevParent.name,
            enabled: prevParent.enabled,
            type: value,
          });
        }}
      />
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

const Form: FC<
  Common.Task.FormProps<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ>
> = () => {
  const listItem = useCallback(
    ({ key, ...rest }: Common.Task.ChannelListItemProps) => (
      <ChannelListItem key={key} {...rest} />
    ),
    [],
  );
  return (
    <Common.Task.Layouts.ListAndDetails<ReadChannel>
      listItem={listItem}
      details={channelDetails}
      createChannel={createReadChannel}
      contextMenuItems={Common.Task.readChannelContextMenuItem}
    />
  );
};

const getInitialValues: Common.Task.GetInitialValues<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = ({ deviceKey, config }) => {
  if (config != null)
    return {
      ...ZERO_READ_PAYLOAD,
      config: readConfigZ.parse(config),
    };
  return {
    ...ZERO_READ_PAYLOAD,
    config: {
      ...ZERO_READ_PAYLOAD.config,
      device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
    },
  };
};

const resolvePDODataType = (slave: Device.SlaveDevice, pdo: string): string => {
  const pdoEntry = slave.properties.pdos.inputs.find((p) => p.name === pdo);
  return pdoEntry?.dataType ?? "uint16";
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
  const network = await client.devices.retrieve<
    Device.NetworkProperties,
    Device.Make,
    Device.NetworkModel
  >({ key: config.device });

  Common.Device.checkConfigured(network);

  const maxSampleRate = network.properties.rate.valueOf();
  if (config.sampleRate > maxSampleRate)
    throw new Error(
      `Sample rate (${config.sampleRate} Hz) exceeds maximum allowed by network rate (${maxSampleRate} Hz)`,
    );

  const networkSafeName = channel.escapeInvalidName(network.name);

  const channelsBySlaveKey = new Map<string, ReadChannel[]>();
  for (const ch of config.channels) {
    const existing = channelsBySlaveKey.get(ch.device) ?? [];
    existing.push(ch);
    channelsBySlaveKey.set(ch.device, existing);
  }

  for (const [slaveKey, channels] of channelsBySlaveKey) {
    const slave = await client.devices.retrieve<
      Device.SlaveProperties,
      Device.Make,
      Device.SlaveModel
    >({ key: slaveKey });

    let modified = await checkOrCreateIndex(client, slave, networkSafeName);

    const toCreate: ReadChannel[] = [];
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
              ? resolvePDODataType(slave, ch.pdo)
              : ch.dataType;
          const pdoName = channel.escapeInvalidName(
            ch.type === AUTOMATIC_TYPE
              ? ch.pdo
              : `_0x${ch.index.toString(16)}_${ch.subindex}`,
          );
          return {
            name: primitive.isNonZero(ch.name)
              ? ch.name
              : `${networkSafeName}_s${slave.properties.position}_${slaveSafeName}_${pdoName}`,
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

  return [config, network.rack];
};

export const Read = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  type: READ_TYPE,
  getInitialValues,
  onConfigure,
});
