// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { primitiveIsZero } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { DigitalListItem } from "@/hardware/ni/task/DigitalListItem";
import { generateDigitalInputChannel } from "@/hardware/ni/task/generateChannel";
import { getDigitalChannelDeviceKey } from "@/hardware/ni/task/getDigitalChannelDeviceKey";
import {
  DIGITAL_READ_TYPE,
  type DigitalInputChannel,
  type DigitalReadConfig,
  digitalReadConfigZ,
  type DigitalReadDetails,
  type DigitalReadType,
  ZERO_DIGITAL_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Layout } from "@/layout";

export const DIGITAL_READ_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  type: DIGITAL_READ_TYPE,
  name: ZERO_DIGITAL_READ_PAYLOAD.name,
  icon: "Logo.NI",
  key: DIGITAL_READ_TYPE,
};

export const DIGITAL_READ_SELECTABLE: Layout.Selectable = {
  key: DIGITAL_READ_TYPE,
  title: "NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  create: (key) => ({ ...DIGITAL_READ_LAYOUT, key }),
};

const Properties = () => (
  <>
    <Device.Select />
    <Common.Task.Fields.SampleRate />
    <Common.Task.Fields.StreamRate />
    <Common.Task.Fields.DataSaving />
  </>
);

const ChannelListItem = (
  props: Common.Task.ChannelListItemProps<DigitalInputChannel>,
) => (
  <DigitalListItem {...props}>
    <Common.Task.ChannelName channel={props.entry.channel} defaultName="No Channel" />
  </DigitalListItem>
);

const Form: FC<
  Common.Task.FormProps<DigitalReadConfig, DigitalReadDetails, DigitalReadType>
> = ({ ...props }) => (
  <Common.Task.Layouts.List<DigitalInputChannel>
    {...props}
    generateChannel={generateDigitalInputChannel}
    ListItem={ChannelListItem}
  />
);

const getInitialPayload: Common.Task.GetInitialPayload<
  DigitalReadConfig,
  DigitalReadDetails,
  DigitalReadType
> = (deviceKey) => ({
  ...ZERO_DIGITAL_READ_PAYLOAD,
  config: {
    ...ZERO_DIGITAL_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_DIGITAL_READ_PAYLOAD.config.device,
  },
});

const onConfigure = async (client: Synnax, config: DigitalReadConfig) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties>(config.device);
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateIndex = primitiveIsZero(dev.properties.digitalInput.index);
  if (!shouldCreateIndex)
    try {
      await client.channels.retrieve(dev.properties.digitalInput.index);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw e;
    }
  if (shouldCreateIndex) {
    modified = true;
    const aiIndex = await client.channels.create({
      name: `${dev.properties.identifier}_di_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.digitalInput.index = aiIndex.key;
    dev.properties.digitalInput.channels = {};
  }
  const toCreate: DigitalInputChannel[] = [];
  for (const channel of config.channels) {
    const key = getDigitalChannelDeviceKey(channel);
    // check if the channel is in properties
    const exKey = dev.properties.digitalInput.channels[key];
    if (primitiveIsZero(exKey)) toCreate.push(channel);
    else
      try {
        await client.channels.retrieve(exKey.toString());
      } catch (e) {
        if (NotFoundError.matches(e)) toCreate.push(channel);
        else throw e;
      }
  }
  if (toCreate.length > 0) {
    modified = true;
    const channels = await client.channels.create(
      toCreate.map((c) => ({
        name: `${dev.properties.identifier}_di_${c.port}_${c.line}`,
        dataType: "uint8",
        index: dev.properties.digitalInput.index,
      })),
    );
    channels.forEach((c, i) => {
      const key = getDigitalChannelDeviceKey(toCreate[i]);
      dev.properties.digitalInput.channels[key] = c.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels.forEach((c) => {
    const key = getDigitalChannelDeviceKey(c);
    c.channel = dev.properties.digitalInput.channels[key];
  });
  return config;
};

export const DigitalRead = Common.Task.wrapForm(() => <Properties />, Form, {
  configSchema: digitalReadConfigZ,
  type: DIGITAL_READ_TYPE,
  getInitialPayload,
  onConfigure,
});
