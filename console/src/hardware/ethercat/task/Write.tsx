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
import { deep, primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ethercat/device";
import { SelectSlave } from "@/hardware/ethercat/device/SelectSlave";
import { SelectChannelModeField } from "@/hardware/ethercat/task/SelectChannelModeField";
import { SelectPDOField } from "@/hardware/ethercat/task/SelectPDOField";
import {
  AUTOMATIC_TYPE,
  type ChannelMode,
  createWriteChannel,
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
  <>
    <Device.Select />
    <Flex.Box x grow>
      <Common.Task.Fields.StateUpdateRate />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<WriteChannel>(path);

  const portLabel =
    ch.type === AUTOMATIC_TYPE
      ? ch.pdo || "No PDO"
      : `0x${ch.index.toString(16).padStart(4, "0")}:${ch.subindex}`;

  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...props}
      port={portLabel}
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
      <SelectChannelModeField
        path={path}
        onChange={(value, { get, set, path: fieldPath }) => {
          const prevType = get<ChannelMode>(fieldPath).value;
          if (prevType === value) return;
          const parentPath = fieldPath.slice(0, fieldPath.lastIndexOf("."));
          const prevParent = get<WriteChannel>(parentPath).value;
          const next = deep.copy(ZERO_WRITE_CHANNELS[value]);
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

const Form: FC<
  Common.Task.FormProps<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ>
> = () => {
  const listItem = useCallback(
    ({ key, ...rest }: Common.Task.ChannelListItemProps) => (
      <ChannelListItem key={key} {...rest} />
    ),
    [],
  );
  return (
    <Common.Task.Layouts.ListAndDetails<WriteChannel>
      listItem={listItem}
      details={channelDetails}
      createChannel={createWriteChannel}
      contextMenuItems={Common.Task.writeChannelContextMenuItems}
    />
  );
};

const getInitialValues: Common.Task.GetInitialValues<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> = ({ deviceKey, config }) => {
  if (config != null)
    return {
      ...ZERO_WRITE_PAYLOAD,
      config: writeConfigZ.parse(config),
    };
  return {
    ...ZERO_WRITE_PAYLOAD,
    config: {
      ...ZERO_WRITE_PAYLOAD.config,
      device: deviceKey ?? ZERO_WRITE_PAYLOAD.config.device,
    },
  };
};

const resolvePDODataType = (
  slave: Device.SlaveDevice | undefined,
  pdo: string,
): string => {
  if (slave == null) return "uint16";
  const pdoEntry = slave.properties.pdos.outputs.find((p) => p.name === pdo);
  return pdoEntry?.dataType ?? "uint16";
};

const onConfigure: Common.Task.OnConfigure<typeof writeConfigZ> = async (
  client,
  config,
) => {
  const network = await client.devices.retrieve<
    Device.NetworkProperties,
    Device.NetworkMake,
    Device.NetworkModel
  >({ key: config.device });

  Common.Device.checkConfigured(network);

  let shouldCreateStateIndex = primitive.isZero(network.properties.write.stateIndex);
  if (!shouldCreateStateIndex)
    try {
      await client.channels.retrieve(network.properties.write.stateIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateStateIndex = true;
      else throw e;
    }

  const safeName = channel.escapeInvalidName(network.name);
  let modified = false;

  try {
    if (shouldCreateStateIndex) {
      modified = true;
      const idx = await client.channels.create({
        name: `${safeName}_state_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      network.properties.write.stateIndex = idx.key;
      network.properties.write.channels = {};
    }

    const slaveCache = new Map<string, Device.SlaveDevice>();
    const toCreate: WriteChannel[] = [];

    for (const ch of config.channels) {
      const mapKey = writeMapKey(ch);
      const existing = network.properties.write.channels[mapKey];
      if (existing == null) {
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
      for (const ch of toCreate)
        if (primitive.isNonZero(ch.device) && !slaveCache.has(ch.device)) {
          const slave = await client.devices.retrieve<
            Device.SlaveProperties,
            Device.SlaveMake,
            Device.SlaveModel
          >({ key: ch.device });
          slaveCache.set(ch.device, slave);
        }

      const cmdIndexes = await client.channels.create(
        toCreate.map((ch) => {
          const slave = slaveCache.get(ch.device);
          const slaveName = slave?.properties?.name ?? "slave";
          const channelLabel =
            ch.type === AUTOMATIC_TYPE
              ? ch.pdo
              : `0x${ch.index.toString(16)}_${ch.subindex}`;
          return {
            name: primitive.isNonZero(ch.name)
              ? `${ch.name}_cmd_time`
              : `${safeName}_${slaveName}_${channelLabel}_cmd_time`,
            dataType: "timestamp",
            isIndex: true,
          };
        }),
      );

      const cmdChannels = await client.channels.create(
        toCreate.map((ch, i) => {
          const slave = slaveCache.get(ch.device);
          const dataType =
            ch.type === AUTOMATIC_TYPE
              ? resolvePDODataType(slave, ch.pdo)
              : ch.dataType;
          const slaveName = slave?.properties?.name ?? "slave";
          const channelLabel =
            ch.type === AUTOMATIC_TYPE
              ? ch.pdo
              : `0x${ch.index.toString(16)}_${ch.subindex}`;
          return {
            name: primitive.isNonZero(ch.name)
              ? `${ch.name}_cmd`
              : `${safeName}_${slaveName}_${channelLabel}_cmd`,
            dataType,
            index: cmdIndexes[i].key,
          };
        }),
      );

      const stateChannels = await client.channels.create(
        toCreate.map((ch) => {
          const slave = slaveCache.get(ch.device);
          const dataType =
            ch.type === AUTOMATIC_TYPE
              ? resolvePDODataType(slave, ch.pdo)
              : ch.dataType;
          const slaveName = slave?.properties?.name ?? "slave";
          const channelLabel =
            ch.type === AUTOMATIC_TYPE
              ? ch.pdo
              : `0x${ch.index.toString(16)}_${ch.subindex}`;
          return {
            name: primitive.isNonZero(ch.name)
              ? `${ch.name}_state`
              : `${safeName}_${slaveName}_${channelLabel}_state`,
            dataType,
            index: network.properties.write.stateIndex,
          };
        }),
      );

      toCreate.forEach((ch, i) => {
        const mapKey = writeMapKey(ch);
        network.properties.write.channels[mapKey] = cmdChannels[i].key;
        network.properties.write.channels[`${mapKey}_state`] = stateChannels[i].key;
      });
    }
  } finally {
    if (modified) await client.devices.create(network);
  }

  config.channels.forEach((ch) => {
    const mapKey = writeMapKey(ch);
    ch.cmdChannel = network.properties.write.channels[mapKey];
    ch.stateChannel = network.properties.write.channels[`${mapKey}_state`];
  });

  return [config, network.rack];
};

export const Write = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: WRITE_SCHEMAS,
  type: WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
