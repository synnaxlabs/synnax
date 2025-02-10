// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError, QueryError, type rack } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form as PForm } from "@synnaxlabs/pluto";
import { id, primitiveIsZero, unique } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Layouts } from "@/hardware/common/task/layouts";
import { type UseTareProps } from "@/hardware/common/task/useTare";
import { Device } from "@/hardware/ni/device";
import { AIChannelForm } from "@/hardware/ni/task/AIChannelForm";
import { generateAIChannel } from "@/hardware/ni/task/generateChannel";
import { SelectAIChannelTypeField } from "@/hardware/ni/task/SelectAIChannelTypeField";
import {
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ANALOG_READ_TYPE,
  type AnalogReadConfig,
  analogReadConfigZ,
  type AnalogReadStateDetails,
  type AnalogReadType,
  ZERO_AI_CHANNEL,
  ZERO_ANALOG_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Layout } from "@/layout";

export const ANALOG_READ_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  type: ANALOG_READ_TYPE,
  name: ZERO_ANALOG_READ_PAYLOAD.name,
  icon: "Logo.NI",
  key: ANALOG_READ_TYPE,
};

export const ANALOG_READ_SELECTABLE: Layout.Selectable = {
  key: ANALOG_READ_TYPE,
  title: "NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  create: async ({ layoutKey }) => ({ ...ANALOG_READ_LAYOUT, key: layoutKey }),
};

const Properties = () => (
  <>
    <Common.Task.Fields.SampleRate />
    <Align.Space direction="x" grow>
      <Common.Task.Fields.StreamRate />
      <Common.Task.Fields.DataSaving />
    </Align.Space>
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<AIChannel> {
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
    entry: { channel, enabled, type, port },
  } = rest;
  const hasTareButton = channel !== 0 && !isSnapshot;
  const canTare = enabled && isRunning;
  return (
    <Layouts.ListAndDetailsChannelItem
      {...rest}
      port={port}
      canTare={canTare}
      onTare={onTare}
      isSnapshot={isSnapshot}
      path={path}
      hasTareButton={hasTareButton}
      channel={channel}
      name={AI_CHANNEL_TYPE_NAMES[type]}
      portMaxChars={2}
    />
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<AIChannelType>(`${path}.type`);
  return (
    <>
      <SelectAIChannelTypeField path={path} inputProps={{ allowNone: false }} />
      <AIChannelForm type={type} prefix={path} />
    </>
  );
};

const Form: FC<
  Common.Task.FormProps<AnalogReadConfig, AnalogReadStateDetails, AnalogReadType>
> = ({ task, isRunning, isSnapshot, configured }) => {
  const [tare, allowTare, handleTare] = Common.Task.useTare<AIChannel>({
    task,
    isRunning,
    configured,
  } as UseTareProps<AIChannel>);
  return (
    <Common.Task.Layouts.ListAndDetails<AIChannel>
      ListItem={(p) => <ChannelListItem {...p} onTare={tare} isRunning={isRunning} />}
      Details={ChannelDetails}
      generateChannel={generateAIChannel}
      isSnapshot={isSnapshot}
      initialChannels={task.config.channels}
      onTare={handleTare}
      allowTare={allowTare}
    />
  );
};

const getInitialPayload: Common.Task.GetInitialPayload<
  AnalogReadConfig,
  AnalogReadStateDetails,
  AnalogReadType
> = (deviceKey) => ({
  ...ZERO_ANALOG_READ_PAYLOAD,
  config: {
    ...ZERO_ANALOG_READ_PAYLOAD.config,
    channels:
      deviceKey == null
        ? ZERO_ANALOG_READ_PAYLOAD.config.channels
        : [{ ...ZERO_AI_CHANNEL, device: deviceKey, key: id.id() }],
  },
});

const onConfigure: Common.Task.OnConfigure<AnalogReadConfig> = async (
  client,
  config,
) => {
  const devices = unique.unique(config.channels.map((c) => c.device));
  // TODO: check that multiple devices are not being configured at once.
  let rackKey: rack.Key | undefined;
  for (const devKey of devices) {
    const dev = await client.hardware.devices.retrieve<Device.Properties>(devKey);
    dev.properties = Device.enrich(dev.model, dev.properties);
    if (rackKey != null && dev.rack !== rackKey)
      throw new Error("All devices must be on the same rack");
    rackKey = dev.rack;
    let modified = false;
    let shouldCreateIndex = primitiveIsZero(dev.properties.analogInput.index);
    if (!shouldCreateIndex)
      try {
        await client.channels.retrieve(dev.properties.analogInput.index);
      } catch (e) {
        if (NotFoundError.matches(e)) shouldCreateIndex = true;
        else throw e;
      }
    if (shouldCreateIndex) {
      modified = true;
      const aiIndex = await client.channels.create({
        name: `${dev.properties.identifier}_ai_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.analogInput.index = aiIndex.key;
      dev.properties.analogInput.channels = {};
    }
    const toCreate: AIChannel[] = [];
    for (const channel of config.channels) {
      if (channel.device !== dev.key) continue;
      // check if the channel is in properties
      const exKey = dev.properties.analogInput.channels[channel.port.toString()];
      if (primitiveIsZero(exKey)) toCreate.push(channel);
      else
        try {
          await client.channels.retrieve(exKey.toString());
        } catch (e) {
          if (QueryError.matches(e)) toCreate.push(channel);
          else throw e;
        }
    }
    if (toCreate.length > 0) {
      modified = true;
      const channels = await client.channels.create(
        toCreate.map((c) => ({
          name: `${dev.properties.identifier}_ai_${c.port}`,
          dataType: "float32",
          index: dev.properties.analogInput.index,
        })),
      );
      channels.forEach(
        (c, i) =>
          (dev.properties.analogInput.channels[toCreate[i].port.toString()] = c.key),
      );
    }
    if (modified) await client.hardware.devices.create(dev);
    config.channels.forEach((c) => {
      if (c.device !== dev.key) return;
      c.channel = dev.properties.analogInput.channels[c.port.toString()];
    });
  }
  if (rackKey == null) throw new Error("No devices selected in task configuration");
  return [config, rackKey];
};

export const AnalogRead = Common.Task.wrapForm(Properties, Form, {
  configSchema: analogReadConfigZ,
  type: ANALOG_READ_TYPE,
  getInitialPayload,
  onConfigure,
});
