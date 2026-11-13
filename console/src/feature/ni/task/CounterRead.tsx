// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError, QueryError, type rack } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { deep, errors, primitive, unique } from "@synnaxlabs/x";
import { type FC } from "react";

import { useByKeys } from "@/feature/ni/device/queries";
import { type Device, MAKE, SCHEMAS } from "@/feature/ni/device/types";
import { counterReadHardwareZ } from "@/feature/ni/task/checks";
import { CIChannelForm } from "@/feature/ni/task/CIChannelForm";
import { createNextCIChannel } from "@/feature/ni/task/createChannel";
import { DetailsTitle as Title } from "@/feature/ni/task/DetailsTitle";
import { COUNTER_INPUT, holdState } from "@/feature/ni/task/ports";
import { PortsView } from "@/feature/ni/task/PortsView";
import {
  CI_CHANNEL_TYPE_ICONS,
  CI_CHANNEL_TYPE_NAMES,
  type CIChannel,
  COUNTER_READ_SCHEMAS,
  COUNTER_READ_TYPE,
  counterReadConfigZ,
  type CounterReadSchemas,
  deployCounterReadConfigZ,
} from "@/feature/ni/task/types";
import { useChannelDevice } from "@/feature/ni/task/usePort";
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<CounterReadSchemas>>();

// Each device has its own ports.
const portOf = (c: CIChannel): string => `${c.device}/${c.port}`;

const Properties = () => (
  <>
    <Task.Fields.SampleRate />
    <Flex.Box x grow>
      <Task.Fields.StreamRate />
    </Flex.Box>
  </>
);

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const entry = root.config.channels[itemKey];
  const value = PForm.useFieldValue(entry);
  const device = useChannelDevice(entry);
  const { port, type, channel } = value;
  const skipped = Task.useSkipped(root.config.channels, itemKey, portOf);
  const Icon = CI_CHANNEL_TYPE_ICONS[type];
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      port={port}
      path={entry}
      channel={channel}
      defaultName={device == null ? undefined : COUNTER_INPUT.name(value, device)}
      skipped={skipped}
      icon={{ icon: <Icon />, name: CI_CHANNEL_TYPE_NAMES[type] }}
      portMaxChars={2}
    />
  );
};

const ChannelDetails = ({ path }: Task.Views.DetailsProps<CIChannel>) => {
  const type = PForm.useFieldValue(path.type);
  return <CIChannelForm type={type} path={path} channelsPath={root.config.channels} />;
};

const channelDetails = Component.renderProp(ChannelDetails);

const DetailsTitle = ({ path }: Task.Views.DetailsProps<CIChannel>) => {
  const value = PForm.useFieldValue(path);
  const channels = PForm.useFieldValue(root.config.channels);
  const device = useChannelDevice(path);
  const Icon = CI_CHANNEL_TYPE_ICONS[value.type];
  const own = channels.filter((c) => c.device === value.device);
  const skipped = Task.useSkipped(root.config.channels, value.key, portOf);
  return (
    <Title
      port={{
        label: String(value.port),
        state: holdState(COUNTER_INPUT, own, value),
        disabled: value.disabled,
      }}
      type={{ icon: <Icon />, name: CI_CHANNEL_TYPE_NAMES[value.type] }}
      name={{
        channel: value.channel,
        path: path.name,
        defaultName: device == null ? undefined : COUNTER_INPUT.name(value, device),
      }}
      skipped={skipped}
      model={device?.model}
    />
  );
};

const detailsTitle = Component.renderProp(DetailsTitle);

const listItem = Component.renderProp(ChannelListItem);

const deviceOf = (c: CIChannel) => c.device;

const hasPortList = (devices: Device[]): boolean =>
  devices.some(({ properties }) => properties.hardware != null);

const Form: FC = () => {
  const devices = useByKeys(Task.useChannelDeviceKeys(root.config.channels));
  PForm.useCheck(root.config, counterReadHardwareZ(devices ?? []));
  const resolve = (ch: CIChannel) => {
    const dev = devices?.find(({ key }) => key === ch.device);
    if (dev == null) return null;
    return { channel: dev.properties.counterInput.channels[ch.port.toString()] ?? 0 };
  };
  const ports =
    devices != null && hasPortList(devices)
      ? (props: Task.Views.PortsProps<CIChannel>) => (
          <PortsView<CIChannel>
            {...props}
            kind={COUNTER_INPUT}
            devices={devices}
            deviceOf={deviceOf}
            create={(channels, device, port) =>
              createNextCIChannel(channels, { device, port }, devices)
            }
            rack={devices[0].rack}
          />
        )
      : undefined;
  return (
    <Task.Views.ListAndDetails<CIChannel>
      path={root.config.channels}
      listItem={listItem}
      details={channelDetails}
      detailsTitle={detailsTitle}
      createChannel={(channels, copy) =>
        createNextCIChannel(channels, { copy }, devices)
      }
      tareable
      contextMenuItems={Task.readChannelContextMenuItem}
      resolve={resolve}
      ports={ports}
    />
  );
};

const getInitialValues: Task.GetInitialValues<CounterReadSchemas> = ({
  deviceKey,
  deviceKeys,
  config,
}) => {
  const cfg = counterReadConfigZ.parse(config ?? {});
  if (config == null) {
    const devices = deviceKeys ?? (deviceKey == null ? [""] : [deviceKey]);
    cfg.channels = [];
    for (const device of devices)
      cfg.channels.push(createNextCIChannel(cfg.channels, { device }));
  }
  return { name: "NI counter read task", type: COUNTER_READ_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<typeof counterReadConfigZ> = async (
  client,
  config,
) => {
  const devices = unique.unique(config.channels.map((c) => c.device));
  const allDevices = await client.devices.retrieve({
    key: devices,
    schemas: SCHEMAS,
  });
  PlatformDevice.checkFound(devices, allDevices);
  const racks = new Set(allDevices.map((d) => d.rack));
  if (racks.size > 1)
    throw new Error("Cannot create task with channels from multiple racks");
  const rackKey: rack.Key = allDevices[0].rack;

  for (const dev of allDevices) {
    PlatformDevice.checkConfigured(dev);
    const before = deep.copy(dev.properties);
    let devModified = false;

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

      const deviceChannels = config.channels.filter((c) => c.device === dev.key);
      const toCreate: CIChannel[] = [];
      for (const channel of Task.firstPerPort(deviceChannels, portOf)) {
        const key = channel.port.toString();
        const exKey = dev.properties.counterInput.channels[key];
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
            name: COUNTER_INPUT.name(c, dev),
            dataType: "float64",
            index: dev.properties.counterInput.index,
          })),
        );
        channels.forEach(
          (c, i) =>
            (dev.properties.counterInput.channels[toCreate[i].port.toString()] = c.key),
        );
      }
      deviceChannels.forEach((c) => {
        c.channel = dev.properties.counterInput.channels[c.port.toString()];
      });
    } finally {
      if (devModified) await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
    }
  }

  if (rackKey == null) throw new Error("No devices selected");
  return [config, rackKey];
};

/** @returns the channels that a plot of the read task shows. */
export const counterReadPlotChannels: Task.PlotChannels = (config) =>
  Task.enabledChannels(COUNTER_READ_SCHEMAS.config.parse(config).channels);

export const CounterRead = Task.wrapForm({
  Properties,
  Form,
  schemas: COUNTER_READ_SCHEMAS,
  deployConfigZ: deployCounterReadConfigZ,
  type: "ni_counter_read",
  onConfigure,
  plotChannels: counterReadPlotChannels,
});

export const useCreateCounterRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const CounterReadSelectable: Selector.Selectable = {
  type: COUNTER_READ_TYPE,
  title: "NI counter read task",
  label: "Counter read",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateCounterRead,
};
