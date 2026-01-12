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
import { id, primitive, strings, unique } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { AIChannelForm } from "@/hardware/ni/task/AIChannelForm";
import { createAIChannel } from "@/hardware/ni/task/createChannel";
import { SelectAIChannelTypeField } from "@/hardware/ni/task/SelectAIChannelTypeField";
import {
  AI_CHANNEL_TYPE_ICONS,
  AI_CHANNEL_TYPE_NAMES,
  type AIChannel,
  type AIChannelType,
  ANALOG_READ_SCHEMAS,
  ANALOG_READ_TYPE,
  analogReadConfigZ,
  type analogReadStatusDataZ,
  type analogReadTypeZ,
  ZERO_AI_CHANNEL,
  ZERO_ANALOG_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Selector } from "@/selector";

export const ANALOG_READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: ANALOG_READ_TYPE,
  name: ZERO_ANALOG_READ_PAYLOAD.name,
  icon: "Logo.NI",
};

export const ANALOG_READ_SELECTABLE: Selector.Selectable = {
  key: ANALOG_READ_TYPE,
  title: "NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  create: async ({ layoutKey }) => ({ ...ANALOG_READ_LAYOUT, key: layoutKey }),
};

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
  const { port, type, channel, enabled } = PForm.useFieldValue<AIChannel>(path);
  const isSnapshot = Common.Task.useIsSnapshot();
  const isRunning = Common.Task.useIsRunning();
  const hasTareButton = channel !== 0 && !isSnapshot;
  const canTare = enabled && isRunning;
  const Icon = AI_CHANNEL_TYPE_ICONS[type];
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...rest}
      port={port}
      canTare={canTare}
      onTare={onTare}
      path={path}
      hasTareButton={hasTareButton}
      channel={channel}
      icon={{ icon: <Icon />, name: AI_CHANNEL_TYPE_NAMES[type] }}
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

const channelDetails = Component.renderProp(ChannelDetails);

const Form: FC<
  Common.Task.FormProps<
    typeof analogReadTypeZ,
    typeof analogReadConfigZ,
    typeof analogReadStatusDataZ
  >
> = () => {
  const [tare, allowTare, handleTare] = Common.Task.useTare<AIChannel>();
  const listItem = useCallback(
    ({ key, ...rest }: Common.Task.ChannelListItemProps) => (
      <ChannelListItem key={key} {...rest} onTare={tare} />
    ),
    [tare],
  );
  return (
    <Common.Task.Layouts.ListAndDetails<AIChannel>
      listItem={listItem}
      details={channelDetails}
      createChannel={createAIChannel}
      onTare={handleTare}
      allowTare={allowTare}
      contextMenuItems={Common.Task.readChannelContextMenuItem}
    />
  );
};

const getInitialValues: Common.Task.GetInitialValues<
  typeof analogReadTypeZ,
  typeof analogReadConfigZ,
  typeof analogReadStatusDataZ
> = ({ deviceKey, config }) => {
  if (config != null)
    return {
      ...ZERO_ANALOG_READ_PAYLOAD,
      config: analogReadConfigZ.parse(config),
    };
  return {
    ...ZERO_ANALOG_READ_PAYLOAD,
    config: {
      ...ZERO_ANALOG_READ_PAYLOAD.config,
      channels:
        deviceKey == null
          ? ZERO_ANALOG_READ_PAYLOAD.config.channels
          : [{ ...ZERO_AI_CHANNEL, device: deviceKey, key: id.create() }],
    },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof analogReadConfigZ> = async (
  client,
  config,
) => {
  const devices = unique.unique(config.channels.map((c) => c.device));
  let rackKey: rack.Key | undefined;
  const allDevices = await client.devices.retrieve<Device.Properties>({
    keys: devices,
  });
  const racks = new Set(allDevices.map((d) => d.rack));
  if (racks.size > 1) {
    const first = allDevices[0];
    const mismatched = allDevices.filter((d) => d.rack !== first.rack);
    throw new Error(
      `All devices must be on the same driver: ${first.name} and ${strings.naturalLanguageJoin(mismatched.map((d) => d.name))} are on different racks`,
    );
  }
  for (const dev of allDevices) {
    Common.Device.checkConfigured(dev);
    dev.properties = Device.enrich(dev.model, dev.properties);
    rackKey = dev.rack;
    let modified = false;
    let shouldCreateIndex = primitive.isZero(dev.properties.analogInput.index);
    if (!shouldCreateIndex)
      try {
        await client.channels.retrieve(dev.properties.analogInput.index);
      } catch (e) {
        if (NotFoundError.matches(e)) shouldCreateIndex = true;
        else throw e;
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
      for (const channel of config.channels) {
        if (channel.device !== dev.key) continue;
        // check if the channel is in properties
        const exKey = dev.properties.analogInput.channels[channel.port.toString()];
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
        modified = true;
        const channels = await client.channels.create(
          toCreate.map((c) => ({
            name: primitive.isNonZero(c.name) ? c.name : `${identifier}_ai_${c.port}`,
            dataType: "float32",
            index: dev.properties.analogInput.index,
          })),
        );
        channels.forEach(
          (c, i) =>
            (dev.properties.analogInput.channels[toCreate[i].port.toString()] = c.key),
        );
      }
    } finally {
      if (modified) await client.devices.create(dev);
    }
    config.channels.forEach((c) => {
      if (c.device !== dev.key) return;
      c.channel = dev.properties.analogInput.channels[c.port.toString()];
    });
  }
  if (rackKey == null) throw new Error("No devices selected in task configuration");
  return [config, rackKey];
};

export const AnalogRead = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: ANALOG_READ_SCHEMAS,
  type: ANALOG_READ_TYPE,
  getInitialValues,
  onConfigure,
});
