// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/modbus/task/Read.css";

import { NotFoundError } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form as PForm, List, Select } from "@synnaxlabs/pluto";
import { DataType, deep, id } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/modbus/device";
import { SelectInputChannelTypeField } from "@/hardware/modbus/task/SelectInputChannelTypeField";
import {
  COIL_INPUT_TYPE,
  DISCRETE_INPUT_TYPE,
  HOLDING_REGISTER_INPUT_TYPE,
  INPUT_CHANNEL_SCHEMAS,
  type InputChannel,
  type InputChannelType,
  isVariableDensityInputChannel,
  READ_TYPE,
  type ReadConfig,
  readConfigZ,
  type ReadStateDetails,
  type ReadType,
  REGISTER_INPUT_TYPE,
  type TypedInput,
  ZERO_INPUT_CHANNELS,
  ZERO_READ_PAYLOAD,
} from "@/hardware/modbus/task/types";
import { type Selector } from "@/selector";

interface FormProps {
  path: string;
}

export const READ_LAYOUT: Common.Task.Layout = {
  ...Common.Task.LAYOUT,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.Modbus",
};

export const READ_SELECTABLE: Selector.Selectable = {
  key: READ_TYPE,
  title: "Modbus Read Task",
  icon: <Icon.Logo.Modbus />,
  create: async ({ layoutKey }) => ({ ...READ_LAYOUT, key: layoutKey }),
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

export const FORMS: Record<InputChannelType, FC<FormProps>> = {
  [COIL_INPUT_TYPE]: () => <></>,
  [DISCRETE_INPUT_TYPE]: () => <></>,
  [HOLDING_REGISTER_INPUT_TYPE]: () => <></>,
  [REGISTER_INPUT_TYPE]: () => <></>,
};

const ChannelListItem = ({
  path,
  isSnapshot,
  ...rest
}: Common.Task.ChannelListItemProps<InputChannel>) => {
  const { entry } = rest;
  return (
    <List.ItemFrame
      {...rest}
      style={{ width: "100%" }}
      justify="spaceBetween"
      align="center"
      direction="x"
    >
      <Align.Pack direction="x" align="center" className={CSS.B("channel-item")}>
        <SelectInputChannelTypeField
          path={path}
          onChange={(value, { get, set, path }) => {
            const prevType = get<InputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(ZERO_INPUT_CHANNELS[value]);
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<InputChannel>(parentPath).value;
            const schema = INPUT_CHANNEL_SCHEMAS[value];
            set(parentPath, {
              ...deep.overrideValidItems(next, prevParent, schema),
              type: value,
            });
          }}
        />
        <PForm.NumericField
          inputProps={{ showDragHandle: false }}
          hideIfNull
          showLabel={false}
          showHelpText={false}
          path={`${path}.address`}
        />
        {(entry.type === REGISTER_INPUT_TYPE ||
          entry.type === HOLDING_REGISTER_INPUT_TYPE) && (
          <PForm.Field<string>
            path={`${path}.dataType`}
            showLabel={false}
            showHelpText={false}
            hideIfNull
          >
            {({ value, onChange }) => (
              <Select.DataType
                value={value}
                onChange={(v) => onChange(v)}
                hideVariableDensity={true}
              />
            )}
          </PForm.Field>
        )}
      </Align.Pack>
      <Common.Task.EnableDisableButton
        path={`${path}.enabled`}
        isSnapshot={isSnapshot}
      />
    </List.ItemFrame>
  );
};

const getOpenChannel = (channels: InputChannel[]): InputChannel => {
  if (channels.length === 0)
    return {
      type: COIL_INPUT_TYPE,
      address: 0,
      channel: 0,
      key: id.create(),
      enabled: true,
    };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    key: id.create(),
    address: channelToCopy.address + 1,
  };
};

const ChannelList = ({ isSnapshot }: { isSnapshot: boolean }) => {
  const createChannel = useCallback(
    (channels: InputChannel[]) => getOpenChannel(channels),
    [],
  );
  const listItem = useCallback(
    ({ key, ...p }: Common.Task.ChannelListItemProps<InputChannel>) => (
      <ChannelListItem key={key} {...p} />
    ),
    [],
  );
  return (
    <Common.Task.Layouts.List<InputChannel>
      isSnapshot={isSnapshot}
      createChannel={createChannel}
      listItem={listItem}
    />
  );
};

const Form: FC<Common.Task.FormProps<ReadConfig, ReadStateDetails, ReadType>> = (
  props,
) => {
  const { isSnapshot } = props;
  return <ChannelList isSnapshot={isSnapshot} />;
};

const readMapKey = (channel: InputChannel) => {
  let s = `${channel.type}-${channel.address.toString()}`;
  if (isVariableDensityInputChannel(channel)) s += `-${channel.dataType}`;
  return s.replaceAll("_", "-");
};

const channelName = (device: Device.Device, channel: InputChannel) => {
  let s = `${device.name}_${channel.type}_${channel.address}`;
  if (isVariableDensityInputChannel(channel))
    s += `_${new DataType(channel.dataType).toString(true)}`;
  return s;
};

const getInitialPayload: Common.Task.GetInitialPayload<
  ReadConfig,
  ReadStateDetails,
  ReadType
> = ({ deviceKey }) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<ReadConfig> = async (client, config) => {
  const dev = await client.hardware.devices.retrieve<Device.Properties, Device.Make>(
    config.device,
  );
  let shouldCreateIndex = false;
  if (dev.properties.read.index)
    try {
      await client.channels.retrieve(dev.properties.read.index);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw e;
    }
  else shouldCreateIndex = true;
  let modified = false;
  if (shouldCreateIndex) {
    modified = true;
    const index = await client.channels.create({
      name: `${dev.name}_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.read.index = index.key;
  }

  const toCreate: InputChannel[] = [];
  for (const c of config.channels) {
    const key = readMapKey(c);
    const existing = dev.properties.read.channels[key];
    console.log(existing, key, dev.properties.read.channels);
    if (existing == null) toCreate.push(c);
    else
      try {
        await client.channels.retrieve(existing.toString());
      } catch (e) {
        if (NotFoundError.matches(e)) toCreate.push(c);
        else throw e;
      }
  }
  if (toCreate.length > 0) {
    modified = true;
    const channels = await client.channels.create(
      toCreate.map((c) => ({
        name: channelName(dev, c),
        dataType: (c as TypedInput).dataType ?? DataType.UINT8.toString(),
        index: dev.properties.read.index,
      })),
    );

    channels.forEach((c, i) => {
      const channel = toCreate[i];
      dev.properties.read.channels[readMapKey(channel)] = c.key;
    });
  }

  if (modified) await client.hardware.devices.create(dev);

  config.channels.forEach((c) => {
    c.channel = dev.properties.read.channels[readMapKey(c)];
  });

  return [config, dev.rack];
};

export const Read = Common.Task.wrapForm({
  Properties,
  Form,
  configSchema: readConfigZ,
  type: READ_TYPE,
  getInitialPayload,
  onConfigure,
});
