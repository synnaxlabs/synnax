// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align } from "@synnaxlabs/pluto";
import { primitiveIsZero } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import {
  DigitalChannelList,
  type NameComponentProps,
} from "@/hardware/ni/task/DigitalChannelList";
import { generateDigitalInputChannel } from "@/hardware/ni/task/generateChannel";
import { getDigitalChannelDeviceKey } from "@/hardware/ni/task/getDigitalChannelDeviceKey";
import {
  DIGITAL_READ_TYPE,
  type DigitalInputChannel,
  type DigitalReadConfig,
  digitalReadConfigZ,
  type DigitalReadStateDetails,
  type DigitalReadType,
  ZERO_DIGITAL_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Layout } from "@/layout";

export const DIGITAL_READ_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  icon: "Logo.NI",
  key: DIGITAL_READ_TYPE,
  name: ZERO_DIGITAL_READ_PAYLOAD.name,
  type: DIGITAL_READ_TYPE,
};

export const DIGITAL_READ_SELECTABLE: Layout.Selectable = {
  create: async ({ layoutKey }) => ({ ...DIGITAL_READ_LAYOUT, key: layoutKey }),
  icon: <Icon.Logo.NI />,
  key: DIGITAL_READ_TYPE,
  title: "NI Digital Read Task",
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

const NameComponent = ({
  entry: { channel },
}: NameComponentProps<DigitalInputChannel>) => (
  <Common.Task.ChannelName channel={channel} />
);

const Form: FC<
  Common.Task.FormProps<DigitalReadConfig, DigitalReadStateDetails, DigitalReadType>
> = (props) => (
  <DigitalChannelList<DigitalInputChannel>
    {...props}
    generateChannel={generateDigitalInputChannel}
    NameComponent={(p) => <NameComponent {...p} />}
  />
);

const getInitialPayload: Common.Task.GetInitialPayload<
  DigitalReadConfig,
  DigitalReadStateDetails,
  DigitalReadType
> = (deviceKey) => ({
  ...ZERO_DIGITAL_READ_PAYLOAD,
  config: {
    ...ZERO_DIGITAL_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_DIGITAL_READ_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<DigitalReadConfig> = async (
  client,
  config,
) => {
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
  getInitialPayload,
  onConfigure,
  type: DIGITAL_READ_TYPE,
});
