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
import { errors, primitive, strings, unique } from "@synnaxlabs/x";
import { type FC } from "react";

import { useByKeys } from "@/feature/ni/device/queries";
import { type Device, MAKE, SCHEMAS } from "@/feature/ni/device/types";
import { AIChannelForm } from "@/feature/ni/task/AIChannelForm";
import { analogReadHardwareZ } from "@/feature/ni/task/checks";
import { createNextAIChannel } from "@/feature/ni/task/createChannel";
import { DetailsTitle as Title } from "@/feature/ni/task/DetailsTitle";
import { getAIChannelDeviceKey } from "@/feature/ni/task/getAIChannelDeviceKey";
import {
  ANALOG_INPUT,
  analogInputClaims,
  analogInputHolds,
  enabledFirst,
  holdState,
} from "@/feature/ni/task/ports";
import { PortsView } from "@/feature/ni/task/PortsView";
import {
  AI_CHANNEL_TYPE_ICONS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ANALOG_READ_SCHEMAS,
  ANALOG_READ_TYPE,
  analogReadConfigZ,
  type AnalogReadSchemas,
  channelPort,
  deployAnalogReadConfigZ,
} from "@/feature/ni/task/types";
import { useChannelDevice } from "@/feature/ni/task/usePort";
import { Device as PlatformDevice } from "@/platform/device";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <Task.Fields.SampleRate />
    <Flex.Box x grow>
      <Task.Fields.StreamRate />
    </Flex.Box>
  </>
);

interface ChannelListItemProps extends Task.ChannelListItemProps {
  onTare: (channelKey: channel.Key) => void;
}

// What the port cell shows for a channel type that reads from no port. Two characters,
// matching the column's own width.
const NO_PORT = "NA";

const ChannelListItem = ({ onTare, ...rest }: ChannelListItemProps) => {
  const path = `config.channels.${rest.itemKey}`;
  const value = PForm.useFieldValue<AIChannel>(path);
  const device = useChannelDevice(path);
  const { type, channel, disabled } = value;
  const isSnapshot = Task.useIsSnapshot();
  const isRunning = Task.useIsRunning();
  const hasTareButton = channel !== 0 && !isSnapshot;
  const canTare = !disabled && isRunning;
  const Icon = AI_CHANNEL_TYPE_ICONS[type];
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...rest}
      port={channelPort(value) ?? NO_PORT}
      canTare={canTare}
      onTare={onTare}
      path={path}
      hasTareButton={hasTareButton}
      channel={channel}
      defaultName={device == null ? undefined : ANALOG_INPUT.name(value, device)}
      icon={{ icon: <Icon />, name: AI_CHANNEL_TYPE_NAMES[type] }}
      portMaxChars={2}
    />
  );
};

const ChannelDetails = ({ path }: Task.Views.DetailsProps) => {
  const type = PForm.useFieldValue<AIChannelType>(`${path}.type`);
  return <AIChannelForm type={type} prefix={path} />;
};

const channelDetails = Component.renderProp(ChannelDetails);

const DetailsTitle = ({ path }: Task.Views.DetailsProps) => {
  const value = PForm.useFieldValue<AIChannel>(path);
  const channels = PForm.useFieldValue<AIChannel[]>("config.channels");
  const device = useChannelDevice(path);
  const hardware = device?.properties.hardware;
  const Icon = AI_CHANNEL_TYPE_ICONS[value.type];
  const port = channelPort(value);
  const claims = analogInputClaims(channels, value.device, hardware);
  return (
    <Title
      port={
        port == null
          ? null
          : {
              label: String(port),
              state: holdState(claims, analogInputHolds(value, hardware)),
              disabled: value.disabled,
            }
      }
      type={{ icon: <Icon />, name: AI_CHANNEL_TYPE_NAMES[value.type] }}
      channel={value.channel}
      namePath={`${path}.name`}
      defaultName={device == null ? undefined : ANALOG_INPUT.name(value, device)}
      model={device?.model}
    />
  );
};

const detailsTitle = Component.renderProp(DetailsTitle);

const deviceOf = (c: AIChannel) => c.device;

const hasPortList = (devices: Device[]): boolean =>
  devices.some(({ properties }) => properties.hardware != null);

const Form: FC = () => {
  const [tare, allowTare, handleTare] = Task.useTare<AIChannel>();
  const devices = useByKeys(Task.useChannelDeviceKeys());
  Task.useConfigCheck(devices == null ? null : analogReadHardwareZ(devices));
  const resolve = (ch: AIChannel) => {
    const dev = devices?.find(({ key }) => key === ch.device);
    if (dev == null) return null;
    return {
      channel: dev.properties.analogInput.channels[getAIChannelDeviceKey(ch)] ?? 0,
    };
  };
  const listItem = ({ key, ...rest }: Task.ChannelListItemProps) => (
    <ChannelListItem key={key} {...rest} onTare={tare} />
  );
  const ports =
    devices != null && hasPortList(devices)
      ? (props: Task.Views.PortsProps) => (
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
      listItem={listItem}
      details={channelDetails}
      detailsTitle={detailsTitle}
      createChannel={(channels, copy) =>
        createNextAIChannel(channels, { copy }, devices)
      }
      onTare={handleTare}
      allowTare={allowTare}
      contextMenuItems={Task.readChannelContextMenuItem}
      resolve={resolve}
      ports={ports}
    />
  );
};

const getInitialValues: Task.GetInitialValues<AnalogReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = analogReadConfigZ.parse(config ?? {});
  if (config == null) {
    const first = createNextAIChannel([]);
    if (deviceKey != null) first.device = deviceKey;
    cfg.channels = [first];
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
    keys: devices,
    schemas: SCHEMAS,
  });
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
      // Channels on one port share its data channel, named by an enabled one if any.
      const seen = new Set<string>();
      for (const channel of enabledFirst(config.channels)) {
        if (channel.device !== dev.key) continue;
        const key = getAIChannelDeviceKey(channel);
        if (seen.has(key)) continue;
        seen.add(key);
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
      if (modified) await client.devices.create(dev, SCHEMAS);
    }
    config.channels.forEach((c) => {
      if (c.device !== dev.key) return;
      c.channel = dev.properties.analogInput.channels[getAIChannelDeviceKey(c)];
    });
  }
  if (rackKey == null) throw new Error("No devices selected");
  return [config, rackKey];
};

export const AnalogRead = Task.wrapForm({
  Properties,
  Form,
  schemas: ANALOG_READ_SCHEMAS,
  deployConfigZ: deployAnalogReadConfigZ,
  deviceConfigZ: async (client, config) =>
    analogReadHardwareZ(
      await client.devices.retrieve({
        keys: unique.unique(config.channels.map(deviceOf)),
        schemas: SCHEMAS,
      }),
    ),
  type: "ni_analog_read",
  getInitialValues,
  onConfigure,
});

export const useCreateAnalogRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const AnalogReadSelectable = Selector.createSelectable({
  type: ANALOG_READ_TYPE,
  title: "NI analog read task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogRead,
});
