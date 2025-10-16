// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError, QueryError, type rack } from "@synnaxlabs/client";
import { Component, Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { id, primitive } from "@synnaxlabs/x";
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
import { type Selector } from "@/selector";

export const COUNTER_READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: COUNTER_READ_TYPE,
  name: ZERO_COUNTER_READ_PAYLOAD.name,
  icon: "Logo.NI",
};

export const COUNTER_READ_SELECTABLE: Selector.Selectable = {
  key: COUNTER_READ_TYPE,
  title: "NI Counter Read Task",
  icon: <Icon.Logo.NI />,
  create: async ({ layoutKey }) => ({ ...COUNTER_READ_LAYOUT, key: layoutKey }),
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
      device: deviceKey ?? "",
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
  const deviceKey = config.device;
  if (primitive.isZero(deviceKey))
    throw new Error("No device selected in task configuration");

  const dev = await client.hardware.devices.retrieve<Device.Properties>({
    key: deviceKey,
  });
  Common.Device.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);
  const rackKey = dev.rack;
  let modified = false;

  // Initialize shared index for counter channels (use same index as analog input)
  let shouldCreateIndex = primitive.isZero(dev.properties.analogInput.index);
  if (!shouldCreateIndex)
    try {
      await client.channels.retrieve(dev.properties.analogInput.index);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw e;
    }
  if (shouldCreateIndex) {
    modified = true;
    const ciIndex = await client.channels.create({
      name: `${dev.properties.identifier}_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.analogInput.index = ciIndex.key;
    dev.properties.analogInput.channels = {};
    dev.properties.counterInput.index = ciIndex.key;
    dev.properties.counterInput.channels = {};
  } else if (primitive.isZero(dev.properties.counterInput.index)) {
    // If analog index exists but counter index doesn't, share it
    modified = true;
    dev.properties.counterInput.index = dev.properties.analogInput.index;
    dev.properties.counterInput.channels = {};
  }

  // Create counter channels
  const toCreate: CIChannel[] = [];
  for (const channel of config.channels) {
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
    modified = true;
    const channels = await client.channels.create(
      toCreate.map((c) => ({
        name: `${dev.properties.identifier}_ctr_${c.port}`,
        dataType: "float64",
        index: dev.properties.counterInput.index,
      })),
    );
    channels.forEach(
      (c, i) =>
        (dev.properties.counterInput.channels[toCreate[i].port.toString()] = c.key),
    );
  }

  if (modified) await client.hardware.devices.create(dev);

  // Map config channels to their Synnax channel keys
  config.channels.forEach((c) => {
    c.channel = dev.properties.counterInput.channels[c.port.toString()];
  });

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
