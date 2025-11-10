// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Flex, Form as PForm, Icon, Select, Text } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/ni/device";
import { COChannelForm } from "@/hardware/ni/task/COChannelForm";
import { createCOChannel } from "@/hardware/ni/task/createChannel";
import { SelectCOChannelTypeField } from "@/hardware/ni/task/SelectCOChannelTypeField";
import {
  CO_CHANNEL_TYPE_ICONS,
  type COChannel,
  type COChannelType,
  COUNTER_WRITE_SCHEMAS,
  COUNTER_WRITE_TYPE,
  counterWriteConfigZ,
  type counterWriteStatusDataZ,
  type counterWriteTypeZ,
  ZERO_COUNTER_WRITE_PAYLOAD,
} from "@/hardware/ni/task/types";
import { type Selector } from "@/selector";

export const COUNTER_WRITE_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: COUNTER_WRITE_TYPE,
  name: ZERO_COUNTER_WRITE_PAYLOAD.name,
  icon: "Logo.NI",
};

export const COUNTER_WRITE_SELECTABLE: Selector.Selectable = {
  key: COUNTER_WRITE_TYPE,
  title: "NI Counter Write Task",
  icon: <Icon.Logo.NI />,
  create: async ({ layoutKey }) => ({ ...COUNTER_WRITE_LAYOUT, key: layoutKey }),
};

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x>
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

// Custom channel name component for Counter Write tasks
// Counter Write channels are configuration-only (no actual Synnax channels created)
// Shows configuration status instead of channel names
const CounterWriteChannelName = ({
  configured,
  className,
}: {
  configured: boolean;
  className?: string;
}) => {
  if (configured) 
    return (
      <Text.Text level="small" className={className} color={7}>
        Configured
      </Text.Text>
    );
  
  return (
    <Text.Text level="small" className={className} status="warning">
      No Channel
    </Text.Text>
  );
};

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const item = PForm.useFieldValue<COChannel>(path);
  if (item == null) return null;
  const { port, type, configured = false } = item;
  const Icon = CO_CHANNEL_TYPE_ICONS[type];

  // CO Pulse Output channels don't have cmd/state channels - config only
  return (
    <Select.ListItem {...props}>
      <Flex.Box direction="x" gap="small" align="center">
        <Text.Text>{port}</Text.Text>
        <Icon />
        {/* Show single configuration status instead of cmd/state */}
        <CounterWriteChannelName configured={configured} />
      </Flex.Box>
      <Flex.Box pack direction="x" align="center" size="small">
        <Common.Task.EnableDisableButton path={`${itemKey}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const ChannelDetails = ({ path }: Common.Task.Layouts.DetailsProps) => {
  const type = PForm.useFieldValue<COChannelType>(`${path}.type`);
  return (
    <>
      <SelectCOChannelTypeField path={path} />
      <COChannelForm type={type} path={path} />
    </>
  );
};

const channelDetails = Component.renderProp(ChannelDetails);
const channelListItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<
    typeof counterWriteTypeZ,
    typeof counterWriteConfigZ,
    typeof counterWriteStatusDataZ
  >
> = () => (
  <Common.Task.Layouts.ListAndDetails
    listItem={channelListItem}
    details={channelDetails}
    createChannel={createCOChannel}
    // CO Pulse Output channels don't have cmd/state, so no write-specific menu items
    contextMenuItems={undefined}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof counterWriteTypeZ,
  typeof counterWriteConfigZ,
  typeof counterWriteStatusDataZ
> = ({ deviceKey, config }) => {
  const cfg =
    config != null
      ? counterWriteConfigZ.parse(config)
      : ZERO_COUNTER_WRITE_PAYLOAD.config;
  return {
    ...ZERO_COUNTER_WRITE_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Common.Task.OnConfigure<typeof counterWriteConfigZ> = async (
  client,
  config,
) => {
  // CO Pulse Output channels are configuration-only - no cmd/state channels needed
  // Just validate the device exists and is configured
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>({
    key: config.device,
  });
  Common.Device.checkConfigured(dev);
  dev.properties = Device.enrich(dev.model, dev.properties);

  config.channels = config.channels.map((c) => ({ ...c, configured: true }));

  // Return configuration as-is without creating any Synnax channels
  return [config, dev.rack];
};

export const CounterWrite = Common.Task.wrapForm({
  Properties,
  Form,
  schemas: COUNTER_WRITE_SCHEMAS,
  type: COUNTER_WRITE_TYPE,
  getInitialValues,
  onConfigure,
});
