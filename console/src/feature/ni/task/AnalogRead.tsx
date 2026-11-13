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
import { Text } from "@synnaxlabs/lyra/text";
import { deep, errors, primitive, strings, unique } from "@synnaxlabs/x";
import { type FC } from "react";

import { useByKeys } from "@/feature/ni/device/queries";
import { type Device, MAKE, SCHEMAS } from "@/feature/ni/device/types";
import { AIChannelForm, AIChannelSettings } from "@/feature/ni/task/AIChannelForm";
import { analogReadHardwareZ } from "@/feature/ni/task/checks";
import { createNextAIChannel } from "@/feature/ni/task/createChannel";
import { DetailsTitle as Title } from "@/feature/ni/task/DetailsTitle";
import { getAIChannelDeviceKey } from "@/feature/ni/task/getAIChannelDeviceKey";
import { ANALOG_INPUT, holdState } from "@/feature/ni/task/ports";
import { PortsView } from "@/feature/ni/task/PortsView";
import {
  AI_CHANNEL_TYPE_ICONS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  ANALOG_READ_SCHEMAS,
  ANALOG_READ_TYPE,
  analogReadConfigZ,
  type AnalogReadSchemas,
  channelPort,
  deployAnalogReadConfigZ,
} from "@/feature/ni/task/types";
import { useChannelDevice } from "@/feature/ni/task/usePort";
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<AnalogReadSchemas>>();

// Each device has its own ports.
const portOf = (c: AIChannel): string => `${c.device}/${getAIChannelDeviceKey(c)}`;

const Properties = () => (
  <>
    <Task.Fields.SampleRate />
    <Flex.Box x grow>
      <Task.Fields.StreamRate />
    </Flex.Box>
  </>
);

// What the port cell shows for a channel type that reads from no port. Two characters,
// matching the column's own width.
const NO_PORT = "NA";

const ChannelListItem = (props: Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const entry = root.config.channels[itemKey];
  const value = PForm.useFieldValue(entry);
  const device = useChannelDevice(entry);
  const { type, channel } = value;
  const skipped = Task.useSkipped(root.config.channels, itemKey, portOf);
  const Icon = AI_CHANNEL_TYPE_ICONS[type];
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...props}
      port={channelPort(value) ?? NO_PORT}
      path={entry}
      channel={channel}
      defaultName={device == null ? undefined : ANALOG_INPUT.name(value, device)}
      skipped={skipped}
      icon={{ icon: <Icon />, name: AI_CHANNEL_TYPE_NAMES[type] }}
      portMaxChars={2}
    />
  );
};

const ChannelDetails = ({ path }: Task.Views.DetailsProps<AIChannel>) => {
  const type = PForm.useFieldValue(path.type);
  return <AIChannelForm type={type} path={path} channelsPath={root.config.channels} />;
};

const listItem = Component.renderProp(ChannelListItem);

const channelDetails = Component.renderProp(ChannelDetails);

/** Edits the signal and the scale of the selected channels while they share a type. */
const MultiDetails = ({ keys }: Task.Views.MultiDetailsProps) => {
  const channels = PForm.useFieldValue(root.config.channels);
  const types = unique.unique(
    channels.filter((c) => keys.includes(c.key)).map((c) => c.type),
  );
  if (types.length > 1)
    return (
      <Text.Text level="small" color={9}>
        Select channels of one type to edit them together.
      </Text.Text>
    );
  const paths = keys.map((key) => root.config.channels[key]);
  return (
    <PForm.Fanout paths={paths}>
      <PForm.Sections>
        <AIChannelSettings type={types[0]} path={paths[0]} />
      </PForm.Sections>
    </PForm.Fanout>
  );
};

const multiDetails = Component.renderProp(MultiDetails);

const DetailsTitle = ({ path }: Task.Views.DetailsProps<AIChannel>) => {
  const value = PForm.useFieldValue(path);
  const channels = PForm.useFieldValue(root.config.channels);
  const device = useChannelDevice(path);
  const hardware = device?.properties.hardware;
  const Icon = AI_CHANNEL_TYPE_ICONS[value.type];
  const port = channelPort(value);
  const own = channels.filter((c) => c.device === value.device);
  const skipped = Task.useSkipped(root.config.channels, value.key, portOf);
  return (
    <Title
      port={
        port == null
          ? null
          : {
              label: String(port),
              state: holdState(ANALOG_INPUT, own, value, hardware),
              disabled: value.disabled,
            }
      }
      type={{ icon: <Icon />, name: AI_CHANNEL_TYPE_NAMES[value.type] }}
      name={{
        channel: value.channel,
        path: path.name,
        defaultName: device == null ? undefined : ANALOG_INPUT.name(value, device),
      }}
      skipped={skipped}
      model={device?.model}
    />
  );
};

const detailsTitle = Component.renderProp(DetailsTitle);

const deviceOf = (c: AIChannel) => c.device;

const hasPortList = (devices: Device[]): boolean =>
  devices.some(({ properties }) => properties.hardware != null);

const Form: FC = () => {
  const devices = useByKeys(Task.useChannelDeviceKeys(root.config.channels));
  PForm.useCheck(root.config, analogReadHardwareZ(devices ?? []));
  const resolve = (ch: AIChannel) => {
    const dev = devices?.find(({ key }) => key === ch.device);
    if (dev == null) return null;
    return {
      channel: dev.properties.analogInput.channels[getAIChannelDeviceKey(ch)] ?? 0,
    };
  };
  const ports =
    devices != null && hasPortList(devices)
      ? (props: Task.Views.PortsProps<AIChannel>) => (
          <PortsView<AIChannel>
            {...props}
            kind={ANALOG_INPUT}
            devices={devices}
            deviceOf={deviceOf}
            create={(channels, device, port) =>
              // A channel on a port never copies one with no port, such as the built-in
              // sensor.
              createNextAIChannel(
                channels.filter((c) => "port" in c),
                { device, port },
                devices,
              )
            }
            rack={devices[0].rack}
          />
        )
      : undefined;
  return (
    <Task.Views.ListAndDetails<AIChannel>
      path={root.config.channels}
      listItem={listItem}
      details={channelDetails}
      multiDetails={multiDetails}
      detailsTitle={detailsTitle}
      createChannel={(channels, copy) =>
        createNextAIChannel(channels, { copy }, devices)
      }
      tareable
      contextMenuItems={Task.readChannelContextMenuItem}
      resolve={resolve}
      ports={ports}
    />
  );
};

const getInitialValues: Task.GetInitialValues<AnalogReadSchemas> = ({
  deviceKey,
  deviceKeys,
  config,
}) => {
  const cfg = analogReadConfigZ.parse(config ?? {});
  if (config == null) {
    const devices = deviceKeys ?? (deviceKey == null ? [""] : [deviceKey]);
    cfg.channels = [];
    for (const device of devices)
      cfg.channels.push(createNextAIChannel(cfg.channels, { device }));
  }
  return { name: "NI analog read task", type: ANALOG_READ_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<typeof analogReadConfigZ> = async (
  client,
  config,
) => {
  const devices = unique.unique(config.channels.map((c) => c.device));
  let rackKey: rack.Key | undefined;
  const allDevices = await client.devices.retrieve({
    key: devices,
    schemas: SCHEMAS,
  });
  PlatformDevice.checkFound(devices, allDevices);
  const racks = new Set(allDevices.map((d) => d.rack));
  if (racks.size > 1) {
    const first = allDevices[0];
    const mismatched = allDevices.filter((d) => d.rack !== first.rack);
    throw new Error(
      `All devices must be on the same rack: ${first.name} and ${strings.naturalLanguageJoin(mismatched.map((d) => d.name))} are on different racks`,
    );
  }
  for (const dev of allDevices) {
    PlatformDevice.checkConfigured(dev);
    const before = deep.copy(dev.properties);
    rackKey = dev.rack;
    let modified = false;
    let shouldCreateIndex = primitive.isZero(dev.properties.analogInput.index);
    if (!shouldCreateIndex)
      try {
        await client.channels.retrieve(dev.properties.analogInput.index);
      } catch (e) {
        if (NotFoundError.matches(e)) shouldCreateIndex = true;
        else throw errors.fromUnknown(e);
      }
    const identifier = channel.escapeInvalidName(dev.properties.identifier);
    try {
      if (shouldCreateIndex) {
        modified = true;
        const aiIndex = await client.channels.create({
          name: `${identifier}_ai_time`,
          dataType: "timestamp",
          isIndex: true,
        });
        dev.properties.analogInput.index = aiIndex.key;
        dev.properties.analogInput.channels = {};
      }
      const toCreate: AIChannel[] = [];
      for (const channel of Task.firstPerPort(config.channels, portOf)) {
        if (channel.device !== dev.key) continue;
        const key = getAIChannelDeviceKey(channel);
        const exKey = dev.properties.analogInput.channels[key];
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
        modified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: ANALOG_INPUT.name(c, dev),
            dataType: "float32",
            index: dev.properties.analogInput.index,
          })),
        );
        channels.forEach((c, i) => {
          const key = getAIChannelDeviceKey(toCreate[i]);
          dev.properties.analogInput.channels[key] = c.key;
        });
      }
    } finally {
      if (modified) await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
    }
    config.channels.forEach((c) => {
      if (c.device !== dev.key) return;
      c.channel = dev.properties.analogInput.channels[getAIChannelDeviceKey(c)];
    });
  }
  if (rackKey == null) throw new Error("No devices selected");
  return [config, rackKey];
};

/** @returns the channels that a plot of the read task shows. */
export const analogReadPlotChannels: Task.PlotChannels = (config) =>
  Task.enabledChannels(ANALOG_READ_SCHEMAS.config.parse(config).channels);

export const AnalogRead = Task.wrapForm({
  Properties,
  Form,
  schemas: ANALOG_READ_SCHEMAS,
  deployConfigZ: deployAnalogReadConfigZ,
  type: "ni_analog_read",
  onConfigure,
  plotChannels: analogReadPlotChannels,
});

export const useCreateAnalogRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const AnalogReadSelectable: Selector.Selectable = {
  type: ANALOG_READ_TYPE,
  title: "NI analog read task",
  label: "Analog read",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogRead,
};
