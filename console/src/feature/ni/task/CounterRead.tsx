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
import { errors, id, primitive, unique } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { enrich } from "@/feature/ni/device/enrich";
import * as Device from "@/feature/ni/device/types";
import { CIChannelForm } from "@/feature/ni/task/CIChannelForm";
import { createCIChannel } from "@/feature/ni/task/createChannel";
import { SelectCIChannelTypeField } from "@/feature/ni/task/SelectCIChannelTypeField";
import {
  CI_CHANNEL_TYPE_ICONS,
  CI_CHANNEL_TYPE_NAMES,
  type CIChannel,
  type CIChannelType,
  COUNTER_READ_SCHEMAS,
  COUNTER_READ_TYPE,
  counterReadConfigZ,
  type CounterReadSchemas,
  ZERO_CI_CHANNEL,
  ZERO_COUNTER_READ_PAYLOAD,
} from "@/feature/ni/task/types";
import { Device as PlatformDevice } from "@/platform/device";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export const CounterReadSelectable = Selector.createSelectable({
  type: COUNTER_READ_TYPE,
  title: "NI Counter Read Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: Task.createOpenTab(COUNTER_READ_TYPE),
});

const Properties = () => (
  <>
    <Task.Fields.SampleRate />
    <Flex.Box x grow>
      <Task.Fields.StreamRate />
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

interface ChannelListItemProps extends Task.ChannelListItemProps {
  onTare: (channelKey: channel.Key) => void;
}

const ChannelListItem = ({ onTare, ...rest }: ChannelListItemProps) => {
  const path = `config.channels.${rest.itemKey}`;
  const { port, type, channel, enabled } = PForm.useFieldValue<CIChannel>(path);
  const isSnapshot = Task.useIsSnapshot();
  const isRunning = Task.useIsRunning();
  const hasTareButton = channel !== 0 && !isSnapshot;
  const canTare = enabled && isRunning;
  const Icon = CI_CHANNEL_TYPE_ICONS[type];
  return (
    <Task.Views.ListAndDetailsChannelItem
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

const ChannelDetails = ({ path }: Task.Views.DetailsProps) => {
  const type = PForm.useFieldValue<CIChannelType>(`${path}.type`);
  return (
    <>
      <SelectCIChannelTypeField path={path} inputProps={{ allowNone: false }} />
      <CIChannelForm type={type} prefix={path} />
    </>
  );
};

const channelDetails = Component.renderProp(ChannelDetails);

const Form: FC<Task.FormProps<CounterReadSchemas>> = () => {
  const [tare, allowTare, handleTare] = Task.useTare<CIChannel>();
  const listItem = useCallback(
    ({ key, itemKey, ...rest }: Task.ChannelListItemProps) => (
      <ChannelListItem key={key} itemKey={itemKey} {...rest} onTare={tare} />
    ),
    [tare],
  );
  return (
    <Task.Views.ListAndDetails<CIChannel>
      listItem={listItem}
      details={channelDetails}
      createChannel={createCIChannel}
      onTare={handleTare}
      allowTare={allowTare}
      contextMenuItems={Task.readChannelContextMenuItem}
    />
  );
};

const getInitialValues: Task.GetInitialValues<CounterReadSchemas> = ({
  deviceKey,
  config,
}) => {
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

const onConfigure: Task.OnConfigure<typeof counterReadConfigZ> = async (
  client,
  config,
) => {
  const devices = unique.unique(config.channels.map((c) => c.device));
  if (devices.length === 0) throw new Error("No device selected in task configuration");

  const allDevices = await client.devices.retrieve({
    keys: devices,
    schemas: Device.SCHEMAS,
  });
  const racks = new Set(allDevices.map((d) => d.rack));
  if (racks.size > 1)
    throw new Error("Cannot create task with channels from multiple racks");
  const rackKey: rack.Key = allDevices[0].rack;

  for (const dev of allDevices) {
    PlatformDevice.checkConfigured(dev);
    dev.properties = enrich(dev.model, dev.properties);
    let devModified = false;

    // Initialize index for counter channels
    let shouldCreateIndex = primitive.isZero(dev.properties.counterInput.index);
    if (!shouldCreateIndex)
      try {
        await client.channels.retrieve(dev.properties.counterInput.index);
      } catch (e) {
        if (NotFoundError.matches(e)) shouldCreateIndex = true;
        else throw errors.fromUnknown(e);
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
            else throw errors.fromUnknown(e);
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
      if (devModified) await client.devices.create(dev, Device.SCHEMAS);
    }
  }

  if (rackKey == null) throw new Error("No devices selected in task configuration");
  return [config, rackKey];
};

export const CounterRead = Task.wrapForm({
  Properties,
  Form,
  Icon: Icon.Logo.NI,
  schemas: COUNTER_READ_SCHEMAS,
  type: "ni_counter_read",
  getInitialValues,
  onConfigure,
});
