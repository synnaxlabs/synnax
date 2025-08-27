// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import { Component, Device as PlutoDevice, Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { createDIChannel } from "@/hardware/ni/task/createChannel";
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
import { type Selector } from "@/selector";

export const DIGITAL_READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  icon: "Logo.NI",
  name: ZERO_DIGITAL_READ_PAYLOAD.name,
  type: DIGITAL_READ_TYPE,
};

export const DIGITAL_READ_SELECTABLE: Selector.Selectable = {
  create: async ({ layoutKey }) => ({ ...DIGITAL_READ_LAYOUT, key: layoutKey }),
  icon: <Icon.Logo.NI />,
  key: DIGITAL_READ_TYPE,
  title: "NI Digital Read Task",
};

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

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const item = PForm.useFieldValue<DIChannel>(path);
  if (item == null) return null;
  const { port, line, channel, customName } = item;
  
  // Get device from the config
  const deviceKey = PForm.useFieldValue<string>("config.device");
  const { data: device } = PlutoDevice.retrieve().useDirect({ 
    params: { key: deviceKey || "" },
  });
  
  const { set } = PForm.useContext();
  
  const handleCustomNameChange = useCallback(
    (newName: string) => {
      set(path, { ...item, customName: newName });
    },
    [item, path, set],
  );
  
  return (
    <Common.Task.Layouts.ListAndDetailsChannelItem
      {...props}
      port={`${port}/${line}`}
      portMaxChars={4}
      hasTareButton={false}
      channel={channel}
      canTare={false}
      path={path}
      previewDevice={device}
      previewChannelType="di"
      customName={customName}
      onCustomNameChange={handleCustomNameChange}
    />
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => (
    <>
      <PForm.NumericField
        path={`${path}.port`}
        label="Port"
        inputProps={{ showDragHandle: false }}
      />
      <PForm.NumericField
        path={`${path}.line`}
        label="Line"
        inputProps={{ showDragHandle: false }}
      />
    </>
  );

const channelDetails = Component.renderProp(ChannelDetails);
const channelListItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<
    typeof digitalReadTypeZ,
    typeof digitalReadConfigZ,
    typeof digitalReadStatusDataZ
  >
> = () => (
  <Common.Task.Layouts.ListAndDetails
    listItem={channelListItem}
    details={channelDetails}
    createChannel={createDIChannel}
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
  const dev = await client.hardware.devices.retrieve<Device.Properties>({
    key: config.device,
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
  if (shouldCreateIndex) {
    modified = true;
    const aiIndex = await client.channels.create({
      name: `${dev.properties.identifier}_time`,
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
        name: c.customName || `${dev.properties.identifier}_${c.port}_${c.line}`,
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
