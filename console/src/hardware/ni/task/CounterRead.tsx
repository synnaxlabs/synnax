// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, QueryError, type rack } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { id, primitive, unique } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { CIChannelForm } from "@/hardware/ni/task/CIChannelForm";
import { createCIChannel } from "@/hardware/ni/task/createChannel";
import { SelectCIChannelTypeField } from "@/hardware/ni/task/SelectCIChannelTypeField";
import {
  CI_CHANNEL_TYPE_ICONS,
  CI_CHANNEL_TYPE_NAMES,
  type CIChannel,
  type CIChannelType,
  COUNTER_READ_SCHEMAS,
  COUNTER_READ_TYPE,
  counterReadConfigZ,
  type counterReadStatusDataZ,
  type counterReadTypeZ,
  ZERO_CI_CHANNEL,
  ZERO_COUNTER_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { Selector } from "@/selector";

export const COUNTER_READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: COUNTER_READ_TYPE,
  name: ZERO_COUNTER_READ_PAYLOAD.name,
  icon: "Logo.NI",
};

export const CounterReadSelectable = Selector.createSimpleItem({
  title: "NI Counter Read Task",
  icon: <Icon.Logo.NI />,
  layout: COUNTER_READ_LAYOUT,
});

const Properties = () => (
  <>
    <Common.Task.Fields.SampleRate />
    <Flex.Box x grow>
      <Common.Task.Fields.StreamRate />
      <Common.Task.Fields.DataSaving />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps {
  onTare: (channelKey: channel.Key) => void;
}

const ChannelListItem = ({ onTare, ...rest }: ChannelListItemProps) => {
  const path = `config.channels.${rest.itemKey}`;
  const { port, type, channel, enabled } = PForm.useFieldValue<CIChannel>(path);
  const isSnapshot = Common.Task.useIsSnapshot();
  const isRunning = Common.Task.useIsRunning();
  const hasTareButton = channel !== 0 && !isSnapshot;
  const canTare = enabled && isRunning;
  const Icon = CI_CHANNEL_TYPE_ICONS[type];
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...rest}
      port={port}
      canTare={canTare}
      onTare={onTare}
      path={path}
      hasTareButton={hasTareButton}
      channel={channel}
      icon={{ icon: <Icon />, name: CI_CHANNEL_TYPE_NAMES[type] }}
      portMaxChars={2}
    />
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<CIChannelType>(`${path}.type`);
  return (
    <>
      <SelectCIChannelTypeField path={path} inputProps={{ allowNone: false }} />
      <CIChannelForm type={type} prefix={path} />
    </>
  );
};

const channelDetails = Component.renderProp(ChannelDetails);

const Form: FC<
  Common.Task.FormProps<
    typeof counterReadTypeZ,
    typeof counterReadConfigZ,
    typeof counterReadStatusDataZ
  >
> = () => {
  const [tare, allowTare, handleTare] = Common.Task.useTare<CIChannel>();
  const listItem = useCallback(
    ({ key, itemKey, ...rest }: Common.Task.ChannelListItemProps) => (
      <ChannelListItem key={key} itemKey={itemKey} {...rest} onTare={tare} />
    ),
    [tare],
  );
  return (
    <Common.Task.Layouts.ListAndDetails<CIChannel>
      listItem={listItem}
      details={channelDetails}
      createChannel={createCIChannel}
      onTare={handleTare}
      allowTare={allowTare}
      contextMenuItems={Common.Task.readChannelContextMenuItem}
    />
  );
};

const getInitialValues: Common.Task.GetInitialValues<
  typeof counterReadTypeZ,
  typeof counterReadConfigZ,
  typeof counterReadStatusDataZ
> = ({ deviceKey, config }) => {
  if (config != null)
    return {
      ...ZERO_COUNTER_READ_PAYLOAD,
      config: counterReadConfigZ.parse(config),
    };
  return {
    ...ZERO_COUNTER_READ_PAYLOAD,
    config: {
      ...ZERO_COUNTER_READ_PAYLOAD.config,
      channels:
        deviceKey == null
          ? ZERO_COUNTER_READ_PAYLOAD.config.channels
          : [{ ...ZERO_CI_CHANNEL, device: deviceKey, key: id.create() }],
    },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof counterReadConfigZ> = async (
  client,
  config,
) => {
  const devices = unique.unique(config.channels.map((c) => c.device));
  if (devices.length === 0) throw new Error("No device selected in task configuration");

  const allDevices = await client.devices.retrieve<Device.Properties>({
    keys: devices,
  });
  const racks = new Set(allDevices.map((d) => d.rack));
  if (racks.size > 1)
    throw new Error("Cannot create task with channels from multiple racks");
  const rackKey: rack.Key = allDevices[0].rack;

  for (const dev of allDevices) {
    Common.Device.checkConfigured(dev);
    dev.properties = Device.enrich(dev.model, dev.properties);
    let devModified = false;

    // Initialize index for counter channels
    let shouldCreateIndex = primitive.isZero(dev.properties.counterInput.index);
    if (!shouldCreateIndex)
      try {
        await client.channels.retrieve(dev.properties.counterInput.index);
      } catch (e) {
        if (NotFoundError.matches(e)) shouldCreateIndex = true;
        else throw e;
      }
    const identifier = channel.escapeInvalidName(dev.properties.identifier);
    try {
      if (shouldCreateIndex) {
        devModified = true;
        const ciIndex = await client.channels.create({
          name: `${identifier}_ctr_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.counterInput.index = ciIndex.key;
        dev.properties.counterInput.channels = {};
      }

      // Create counter channels for this device
      const deviceChannels = config.channels.filter((c) => c.device === dev.key);
      const toCreate: CIChannel[] = [];
      for (const channel of deviceChannels) {
        const exKey = dev.properties.counterInput.channels[channel.port.toString()];
        if (primitive.isZero(exKey)) toCreate.push(channel);
        else
          try {
            await client.channels.retrieve(exKey.toString());
          } catch (e) {
            if (QueryError.matches(e)) toCreate.push(channel);
            else throw e;
          }
      }

      if (toCreate.length > 0) {
        devModified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: primitive.isNonZero(c.name) ? c.name : `${identifier}_ctr_${c.port}`,
            dataType: "float64",
            index: dev.properties.counterInput.index,
          })),
        );
        channels.forEach(
          (c, i) =>
            (dev.properties.counterInput.channels[toCreate[i].port.toString()] = c.key),
        );
      }
      // Map config channels to their Synnax channel keys
      deviceChannels.forEach((c) => {
        c.channel = dev.properties.counterInput.channels[c.port.toString()];
      });
    } finally {
      if (devModified) await client.devices.create(dev);
    }
  }

  if (rackKey == null) throw new Error("No devices selected in task configuration");
  return [config, rackKey];
};

export const CounterRead = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: COUNTER_READ_SCHEMAS,
  type: COUNTER_READ_TYPE,
  getInitialValues,
  onConfigure,
});
