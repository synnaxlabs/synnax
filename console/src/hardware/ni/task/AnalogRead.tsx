// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  NotFoundError,
  QueryError,
  type Synnax,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form as PForm, List, Text } from "@synnaxlabs/pluto";
import { id, primitiveIsZero, unique } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { AnalogInputChannelForm } from "@/hardware/ni/task/AnalogInputChannelForm";
import { generateAnalogInputChannel } from "@/hardware/ni/task/generateChannel";
import { SelectAIChannelTypeField } from "@/hardware/ni/task/SelectAIChannelTypeField";
import {
  ANALOG_INPUT_CHANNEL_TYPE_NAMES,
  ANALOG_READ_TYPE,
  type AnalogInputChannel,
  type AnalogInputChannelType,
  type AnalogReadConfig,
  analogReadConfigZ,
  type AnalogReadStateDetails,
  type AnalogReadType,
  ZERO_ANALOG_INPUT_CHANNEL,
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

interface ChannelListItemProps
  extends Common.Task.ChannelListItemProps<AnalogInputChannel> {
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
    <List.ItemFrame {...rest} justify="spaceBetween" align="center">
      <Align.Space direction="x">
        <Text.Text level="p" shade={6}>
          {port}
        </Text.Text>
        <Text.Text level="p" shade={9}>
          {ANALOG_INPUT_CHANNEL_TYPE_NAMES[type]}
        </Text.Text>
      </Align.Space>
      <Align.Pack direction="x" align="center" size="small">
        {hasTareButton && (
          <Common.Task.TareButton disabled={!canTare} onTare={() => onTare(channel)} />
        )}
        <Common.Task.EnableDisableButton
          path={`${path}.enabled`}
          isSnapshot={isSnapshot}
        />
      </Align.Pack>
    </List.ItemFrame>
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<AnalogInputChannelType>(`${path}.type`);
  return (
    <>
      <SelectAIChannelTypeField path={path} inputProps={{ allowNone: false }} />
      <AnalogInputChannelForm type={type} prefix={path} />
    </>
  );
};

const Form: FC<
  Common.Task.FormProps<AnalogReadConfig, AnalogReadStateDetails, AnalogReadType>
> = ({ task, isRunning, isSnapshot }) => {
  const [tare, allowTare, handleTare] = Common.Task.useTare<AnalogInputChannel>({
    task,
    isRunning,
  });
  return (
    <Common.Task.Layouts.ListAndDetails<AnalogInputChannel>
      ListItem={(p) => <ChannelListItem {...p} onTare={tare} isRunning={isRunning} />}
      Details={ChannelDetails}
      generateChannel={generateAnalogInputChannel}
      isSnapshot={isSnapshot}
      initalChannels={task.config.channels}
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
        : [{ ...ZERO_ANALOG_INPUT_CHANNEL, device: deviceKey, key: id.id() }],
  },
});

const onConfigure = async (client: Synnax, config: AnalogReadConfig) => {
  const devices = unique.unique(config.channels.map((c) => c.device));
  for (const devKey of devices) {
    const dev = await client.hardware.devices.retrieve<Device.Properties>(devKey);
    dev.properties = Device.enrich(dev.model, dev.properties);
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
    const toCreate: AnalogInputChannel[] = [];
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
  return config;
};

export const AnalogRead = Common.Task.wrapForm(() => <Properties />, Form, {
  configSchema: analogReadConfigZ,
  type: ANALOG_READ_TYPE,
  getInitialPayload,
  onConfigure,
});
