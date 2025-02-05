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
import { id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { DigitalListItem } from "@/hardware/ni/task/DigitalListItem";
import {
  type DIChannel,
  DIGITAL_READ_TYPE,
  type DigitalReadConfig,
  digitalReadConfigZ,
  type DigitalReadDetails,
  type DigitalReadType,
  ZERO_DI_CHANNEL,
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

const Properties = (): ReactElement => (
  <>
    <Device.Select />
    <Common.Task.Fields.SampleRate />
    <Common.Task.Fields.StreamRate />
    <Common.Task.Fields.DataSaving />
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<DIChannel> {}

const ChannelListItem = ({
  entry,
  entry: { channel },
  ...props
}: ChannelListItemProps): ReactElement => (
  <DigitalListItem {...props} entry={entry}>
    <Common.Task.ChannelName channel={channel} defaultName="No Channel" />
  </DigitalListItem>
);

const generateChannel = (channels: DIChannel[]): DIChannel => {
  const line = Math.max(0, ...channels.map((v) => v.line)) + 1;
  return { ...ZERO_DI_CHANNEL, key: id.id(), line };
};

const TaskForm: FC<
  Common.Task.FormProps<DigitalReadConfig, DigitalReadDetails, DigitalReadType>
> = ({ isSnapshot }) => (
  <Common.Task.Layouts.List<DIChannel>
    isSnapshot={isSnapshot}
    generateChannel={generateChannel}
  >
    {(p) => <ChannelListItem {...p} />}
  </Common.Task.Layouts.List>
);

const zeroPayload: Common.Task.ZeroPayloadFunction<
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

const onConfigure = async (
  client: Synnax,
  config: DigitalReadConfig,
): Promise<DigitalReadConfig> => {
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
  const toCreate: DIChannel[] = [];
  for (const channel of config.channels) {
    const key = `${channel.port}l${channel.line}`;
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
      const key = `${toCreate[i].port}l${toCreate[i].line}`;
      dev.properties.digitalInput.channels[key] = c.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels.forEach((c) => {
    const key = `${c.port}l${c.line}`;
    c.channel = dev.properties.digitalInput.channels[key];
  });
  return config;
};

export const DigitalReadTask = Common.Task.wrapForm(<Properties />, TaskForm, {
  configSchema: digitalReadConfigZ,
  type: DIGITAL_READ_TYPE,
  zeroPayload,
  onConfigure,
});
