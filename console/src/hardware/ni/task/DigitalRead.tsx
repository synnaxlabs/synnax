// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component, Flex, Icon } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { createDIChannel } from "@/hardware/ni/task/createChannel";
import {
  DigitalChannelList,
  type DigitalNameComponentProps,
} from "@/hardware/ni/task/DigitalChannelList";
import { getDigitalChannelDeviceKey } from "@/hardware/ni/task/getDigitalChannelDeviceKey";
import {
  type DIChannel,
  DIGITAL_READ_SCHEMAS,
  DIGITAL_READ_TYPE,
  digitalReadConfigZ,
  type digitalReadStatusDataZ,
  type digitalReadTypeZ,
  ZERO_DIGITAL_READ_PAYLOAD,
} from "@/hardware/ni/task/types";
import { Selector } from "@/selector";

export const DIGITAL_READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  icon: "Logo.NI",
  name: ZERO_DIGITAL_READ_PAYLOAD.name,
  type: DIGITAL_READ_TYPE,
};

export const DigitalReadSelectable = Selector.createSimpleItem({
  title: "NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  layout: DIGITAL_READ_LAYOUT,
});

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x>
      <Common.Task.Fields.SampleRate />
      <Common.Task.Fields.StreamRate />
      <Common.Task.Fields.DataSaving />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

interface NameComponentProps extends DigitalNameComponentProps<DIChannel> {}

const NameComponent = ({ channel, itemKey, path }: NameComponentProps) => (
  <Common.Task.ChannelName
    channel={channel}
    id={Common.Task.getChannelNameID(itemKey)}
    level="p"
    namePath={`${path}.name`}
  />
);

const name = Component.renderProp(NameComponent);

const Form: FC<
  Common.Task.FormProps<
    typeof digitalReadTypeZ,
    typeof digitalReadConfigZ,
    typeof digitalReadStatusDataZ
  >
> = (props) => (
  <DigitalChannelList<DIChannel>
    {...props}
    createChannel={createDIChannel}
    name={name}
    contextMenuItems={Common.Task.readChannelContextMenuItem}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof digitalReadTypeZ,
  typeof digitalReadConfigZ,
  typeof digitalReadStatusDataZ
> = ({ deviceKey, config }) => {
  const cfg =
    config != null
      ? digitalReadConfigZ.parse(config)
      : ZERO_DIGITAL_READ_PAYLOAD.config;
  return {
    ...ZERO_DIGITAL_READ_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof digitalReadConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: { properties: Device.propertiesZ, make: Device.makeZ },
  });
  Common.Device.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);
  let modified = false;
  let shouldCreateIndex = primitive.isZero(dev.properties.digitalInput.index);
  if (!shouldCreateIndex)
    try {
      await client.channels.retrieve(dev.properties.digitalInput.index);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw e;
    }
  const identifier = channel.escapeInvalidName(dev.properties.identifier);
  try {
    if (shouldCreateIndex) {
      modified = true;
      const aiIndex = await client.channels.create({
        name: `${identifier}_di_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.digitalInput.index = aiIndex.key;
      dev.properties.digitalInput.channels = {};
    }
    const toCreate: DIChannel[] = [];
    for (const channel of config.channels) {
      const key = getDigitalChannelDeviceKey(channel);
      // check if the channel is in properties
      const exKey = dev.properties.digitalInput.channels[key];
      if (primitive.isZero(exKey)) toCreate.push(channel);
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
          name: primitive.isNonZero(c.name)
            ? c.name
            : `${identifier}_di_${c.port}_${c.line}`,
          dataType: "uint8",
          index: dev.properties.digitalInput.index,
        })),
      );
      channels.forEach((c, i) => {
        const key = getDigitalChannelDeviceKey(toCreate[i]);
        dev.properties.digitalInput.channels[key] = c.key;
      });
    }
  } finally {
    if (modified) await client.devices.create(dev);
  }
  config.channels.forEach((c) => {
    const key = getDigitalChannelDeviceKey(c);
    c.channel = dev.properties.digitalInput.channels[key];
  });
  return [config, dev.rack];
};

export const DigitalRead = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: DIGITAL_READ_SCHEMAS,
  getInitialValues,
  onConfigure,
  type: DIGITAL_READ_TYPE,
});
