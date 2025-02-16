// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form as PForm } from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";
import { convertChannelTypeToPortType } from "@/hardware/labjack/task/convertChannelTypeToPortType";
import { getOpenPort } from "@/hardware/labjack/task/getOpenPort";
import { FORMS } from "@/hardware/labjack/task/OutputChannelForms";
import { SelectInputChannelTypeField } from "@/hardware/labjack/task/SelectInputChannelTypeField";
import {
  AI_CHANNEL_TYPE,
  DI_CHANNEL_TYPE,
  INPUT_CHANNEL_SCHEMAS,
  type InputChannel,
  type InputChannelType,
  READ_TYPE,
  type ReadConfig,
  readConfigZ,
  type ReadPayload,
  type ReadStateDetails,
  type ReadTask,
  type ReadType,
  ZERO_INPUT_CHANNEL,
  ZERO_INPUT_CHANNELS,
  ZERO_READ_PAYLOAD,
} from "@/hardware/labjack/task/types";
import { type Layout } from "@/layout";

export const READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const READ_SELECTABLE: Layout.Selectable = {
  key: READ_TYPE,
  title: "LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  create: async ({ layoutKey }) => ({ ...READ_LAYOUT, key: layoutKey }),
};

const Properties = () => (
  <>
    <Device.Select />
    <Align.Space direction="x" grow>
      <Common.Task.Fields.SampleRate />
      <Common.Task.Fields.StreamRate />
      <Common.Task.Fields.DataSaving />
    </Align.Space>
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<InputChannel> {
  onTare: (channelKey: channel.Key) => void;
  isRunning: boolean;
}

const ChannelListItem = ({
  path,
  isSnapshot,
  onTare,
  isRunning,
  ...rest
}: ChannelListItemProps) => {
  const {
    entry: { channel, port, enabled, type },
  } = rest;
  const hasTareButton = channel !== 0 && type === AI_CHANNEL_TYPE && !isSnapshot;
  const canTare = enabled && isRunning;
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...rest}
      port={port}
      canTare={canTare}
      onTare={onTare}
      isSnapshot={isSnapshot}
      path={path}
      hasTareButton={hasTareButton}
      channel={channel}
      portMaxChars={5}
    />
  );
};

interface ChannelDetailsProps extends Common.Task.Layouts.DetailsProps {
  device: Device.Device;
}

const ChannelDetails = ({ path, device }: ChannelDetailsProps) => {
  const channel = PForm.useFieldValue<InputChannel>(path);
  const model = device.model;
  const Form = FORMS[channel.type];
  return (
    <>
      <Align.Space direction="x">
        <SelectInputChannelTypeField
          path={path}
          grow
          onChange={(value, { get, path, set }) => {
            const prevType = get<InputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(ZERO_INPUT_CHANNELS[value]);
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<InputChannel>(parentPath).value;
            const schema = INPUT_CHANNEL_SCHEMAS[value];
            const port =
              Device.DEVICES[model].ports[convertChannelTypeToPortType(value)][0].key;
            set(parentPath, {
              ...deep.overrideValidItems(next, prevParent, schema),
              type: next.type,
            });
            // Need to explicitly set port to cause select port field to rerender
            set(`${parentPath}.port`, port);
          }}
        />
        <PForm.Field<string> path={`${path}.port`}>
          {(p) => (
            <Device.SelectPort
              {...p}
              model={model}
              portType={convertChannelTypeToPortType(channel.type)}
            />
          )}
        </PForm.Field>
      </Align.Space>
      <Form deviceModel={device.model} path={path} />
    </>
  );
};

const getOpenChannel = (
  channels: InputChannel[],
  index: number,
  device: Device.Device,
) => {
  if (index === -1) return { ...deep.copy(ZERO_INPUT_CHANNEL), key: id.id() };
  const channelToCopy = channels[index];
  // preferredPortType is AI or DI
  const preferredPortType = convertChannelTypeToPortType(channelToCopy.type);
  // backupPortType is the opposite of preferredPortType
  const backupPortType =
    preferredPortType === Device.DI_PORT_TYPE
      ? Device.AI_PORT_TYPE
      : Device.DI_PORT_TYPE;
  const port = getOpenPort(channels, device.model, [preferredPortType, backupPortType]);
  if (port == null) return null;
  // Now we need to determine what channel type we use the schema and zero channel for.
  // Note that if the copied channel was a TC channel, then we need to grab
  // channelToCopy.type instead of port.type as port.type cannot be TC.
  const channelTypeUsed =
    port.type === preferredPortType ? channelToCopy.type : backupPortType;
  return {
    ...deep.overrideValidItems(
      ZERO_INPUT_CHANNELS[channelTypeUsed],
      channelToCopy,
      INPUT_CHANNEL_SCHEMAS[channelTypeUsed],
    ),
    key: id.id(),
    port: port.key,
    channel: device.properties[port.type].channels[port.key] ?? 0,
  };
};

type ChannelsFormProps = {
  device: Device.Device;
  isRunning: boolean;
  isSnapshot: boolean;
  configured: boolean;
  task: ReadPayload | ReadTask;
};

const ChannelsForm = ({
  device,
  isSnapshot,
  isRunning,
  configured,
  task,
}: ChannelsFormProps) => {
  const [tare, allowTare, handleTare] = Common.Task.useTare<InputChannel>({
    isChannelTareable: ({ type }) => type === AI_CHANNEL_TYPE,
    isRunning,
    configured,
    task,
  } as Common.Task.UseTareProps<InputChannel>);
  const generateChannel = useCallback(
    (channels: InputChannel[], index: number) =>
      getOpenChannel(channels, index, device),
    [device],
  );
  return (
    <Common.Task.Layouts.ListAndDetails<InputChannel>
      ListItem={(p) => <ChannelListItem {...p} onTare={tare} isRunning={isRunning} />}
      Details={(p) => <ChannelDetails {...p} device={device} />}
      generateChannel={generateChannel}
      isSnapshot={isSnapshot}
      initialChannels={task.config.channels}
      onTare={handleTare}
      allowTare={allowTare}
    />
  );
};

const Form: FC<Common.Task.FormProps<ReadConfig, ReadStateDetails, ReadType>> = (
  props,
) => {
  const { isSnapshot } = props;
  return (
    <Common.Device.Provider<Device.Properties, Device.Make, Device.Model>
      canConfigure={!isSnapshot}
      configureLayout={Device.CONFIGURE_LAYOUT}
    >
      {({ device }) => <ChannelsForm device={device} {...props} />}
    </Common.Device.Provider>
  );
};

const getInitialPayload: Common.Task.GetInitialPayload<
  ReadConfig,
  ReadStateDetails,
  ReadType
> = (deviceKey) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<ReadConfig> = async (client, config) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties>(config.device);
  let shouldCreateIndex = false;
  if (dev.properties.readIndex)
    try {
      await client.channels.retrieve(dev.properties.readIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw e;
    }
  else shouldCreateIndex = true;
  let modified = false;
  if (shouldCreateIndex) {
    modified = true;
    const index = await client.channels.create({
      name: `${dev.properties.identifier}_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.readIndex = index.key;
  }
  const toCreate: InputChannel[] = [];
  for (const c of config.channels) {
    const type = convertChannelTypeToPortType(c.type);
    const existing = dev.properties[type].channels[c.port];
    // check if the channel is in properties
    if (primitiveIsZero(existing)) toCreate.push(c);
    else
      try {
        await client.channels.retrieve(existing.toString());
      } catch (e) {
        if (NotFoundError.matches(e)) toCreate.push(c);
        else throw e;
      }
  }
  if (toCreate.length > 0) {
    modified = true;
    const channels = await client.channels.create(
      toCreate.map((c) => ({
        name: `${dev.properties.identifier}_${c.port}`,
        dataType: c.type === DI_CHANNEL_TYPE ? "uint8" : "float32",
        index: dev.properties.readIndex,
      })),
    );
    channels.forEach((c, i) => {
      const toCreateC = toCreate[i];
      const type = convertChannelTypeToPortType(toCreateC.type);
      dev.properties[type].channels[toCreateC.port] = c.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels.forEach(
    (c) =>
      (c.channel =
        dev.properties[convertChannelTypeToPortType(c.type)].channels[c.port]),
  );
  return [config, dev.rack];
};

export const Read = Common.Task.wrapForm(Properties, Form, {
  configSchema: readConfigZ,
  type: READ_TYPE,
  getInitialPayload,
  onConfigure,
});
