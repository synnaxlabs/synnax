// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Form as PForm, List, Text } from "@synnaxlabs/pluto";
import { deep, id, primitiveIsZero } from "@synnaxlabs/x";
import { type FC, type ReactElement, useCallback } from "react";

import { Common } from "@/hardware/common";
import { Device } from "@/hardware/labjack/device";
import { CustomScaleForm } from "@/hardware/labjack/task/CustomScaleForm";
import { getOpenPort } from "@/hardware/labjack/task/getOpenPort";
import { SelectInputChannelTypeField } from "@/hardware/labjack/task/SelectInputChannelTypeField";
import { ThermocoupleForm } from "@/hardware/labjack/task/ThermocoupleForm";
import {
  AI_CHANNEL_TYPE,
  DI_CHANNEL_TYPE,
  getPortTypeFromChannelType,
  type InputChannelType,
  inputChannelZ,
  type Read,
  READ_TYPE,
  type ReadChannel,
  type ReadConfig,
  readConfigZ,
  type ReadPayload,
  type ReadStateDetails,
  type ReadType,
  TC_CHANNEL_TYPE,
  thermocoupleChannelZ,
  ZERO_READ_CHANNEL,
  ZERO_READ_PAYLOAD,
  ZERO_THERMOCOUPLE_CHANNEL,
} from "@/hardware/labjack/task/types";
import { type Layout } from "@/layout";

export const READ_LAYOUT: Common.Task.LayoutBaseState = {
  ...Common.Task.LAYOUT,
  key: READ_TYPE,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Logo.LabJack",
};

export const READ_SELECTABLE: Layout.Selectable = {
  key: READ_TYPE,
  title: "LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  create: (key) => ({ ...READ_LAYOUT, key }),
};

const Properties = (): ReactElement => (
  <>
    <Device.Select />
    <Common.Task.Fields.SampleRate />
    <Common.Task.Fields.StreamRate />
    <Common.Task.Fields.DataSaving />
  </>
);

interface ChannelListItemProps extends Common.Task.ChannelListItemProps<ReadChannel> {
  onTare: (channelKey: channel.Key) => void;
  isRunning: boolean;
}

const ChannelListItem = ({
  path,
  isSnapshot,
  onTare,
  isRunning,
  ...rest
}: ChannelListItemProps): ReactElement => {
  const {
    entry: { channel, port, enabled, type },
  } = rest;
  const { set } = PForm.useContext();
  const hasTareButton = channel !== 0 && type === AI_CHANNEL_TYPE && !isSnapshot;
  const canTare = enabled && isRunning;
  return (
    <List.ItemFrame {...rest} justify="spaceBetween" align="center">
      <Align.Space direction="x" size="small">
        <Text.Text level="p" shade={6}>
          {port}
        </Text.Text>
        <Common.Task.ChannelName channel={channel} />
      </Align.Space>
      <Align.Pack direction="x" align="center" size="small">
        {hasTareButton && (
          <Common.Task.TareButton disabled={!canTare} onClick={() => onTare(channel)} />
        )}
        <Common.Task.EnableDisableButton
          value={enabled}
          onChange={(v) => set(`${path}.enabled`, v)}
          isSnapshot={isSnapshot}
        />
      </Align.Pack>
    </List.ItemFrame>
  );
};

interface ChannelDetailsProps extends Common.Task.Layouts.DetailsProps {
  device: Device.Device;
}

const ChannelDetails = ({ path, device }: ChannelDetailsProps): ReactElement => {
  const channel = PForm.useFieldValue<ReadChannel>(path);
  const model = device.model;
  return (
    <>
      <Align.Space direction="x">
        <SelectInputChannelTypeField
          path={path}
          onChange={(value, { get, path, set }) => {
            const prevType = get<InputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(
              value === TC_CHANNEL_TYPE ? ZERO_THERMOCOUPLE_CHANNEL : ZERO_READ_CHANNEL,
            );
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<ReadChannel>(parentPath).value;
            const schema =
              value === TC_CHANNEL_TYPE ? thermocoupleChannelZ : inputChannelZ;
            const port =
              Device.DEVICES[model].ports[getPortTypeFromChannelType(value)][0].key;
            set(parentPath, {
              ...deep.overrideValidItems(next, prevParent, schema),
              type: next.type,
            });
            // Need to explicitly set port to cause select port field to rerender
            set(`${parentPath}.port`, port);
          }}
        />
        <PForm.Field<string> path={`${path}.port`}>
          {(p) => (
            <Device.SelectPort
              {...p}
              model={model}
              portType={getPortTypeFromChannelType(channel.type)}
            />
          )}
        </PForm.Field>
      </Align.Space>
      <PForm.NumericField
        path={`${path}.range`}
        label="Max Voltage"
        inputProps={{ endContent: "V" }}
      />
      <ThermocoupleForm model={model} prefix={path} />
      <CustomScaleForm prefix={path} />
    </>
  );
};

const getOpenChannel = (
  channels: ReadChannel[],
  index: number,
  device: Device.Device,
): ReadChannel | null => {
  if (index === -1) return { ...deep.copy(ZERO_READ_CHANNEL), key: id.id() };
  const channelToCopy = channels[index];
  const preferredType = getPortTypeFromChannelType(channelToCopy.type);
  const backupType =
    preferredType === Device.DI_PORT_TYPE ? Device.AI_PORT_TYPE : Device.DI_PORT_TYPE;
  const port = getOpenPort(channels, device.model, [preferredType, backupType]);
  if (port == null) return null;
  const base = {
    key: id.id(),
    port: port.key,
    channel: device.properties[port.type].channels[port.key] ?? 0,
  };
  if (port.type === preferredType || channelToCopy.type !== TC_CHANNEL_TYPE)
    return { ...deep.copy(channelToCopy), ...base };
  // because the TC channel schema is more complicated, we need to overrideValidItems
  // when we use a digital input type based off of a TC channel
  return {
    ...deep.overrideValidItems(ZERO_READ_CHANNEL, channelToCopy, inputChannelZ),
    ...base,
  };
};

interface ChannelsFormProps {
  device: Device.Device;
  task: Read | ReadPayload;
  isRunning: boolean;
  isSnapshot: boolean;
}

const ChannelsForm = ({
  device,
  task,
  isRunning,
  isSnapshot,
}: ChannelsFormProps): ReactElement => {
  const [tare, allowTare, handleTare] = Common.Task.useTare<ReadChannel>({
    task,
    isRunning,
    isChannelTareable: ({ type }) => type === AI_CHANNEL_TYPE,
  });
  const generateChannel = useCallback(
    (channels: ReadChannel[], index: number) => getOpenChannel(channels, index, device),
    [device],
  );
  return (
    <Common.Task.Layouts.ListAndDetails
      listItem={(p) => <ChannelListItem {...p} onTare={tare} isRunning={isRunning} />}
      details={(p) => <ChannelDetails {...p} device={device} />}
      generateChannel={generateChannel}
      isSnapshot={isSnapshot}
      initalChannels={task.config.channels}
      onTare={handleTare}
      allowTare={allowTare}
    />
  );
};

const Form: FC<Common.Task.FormProps<ReadConfig, ReadStateDetails, ReadType>> = ({
  task,
  isRunning,
  isSnapshot,
}) => (
  <Common.Device.Provider<Device.Properties, Device.Make, Device.Model>
    canConfigure={!isSnapshot}
    configureLayout={Device.CONFIGURE_LAYOUT}
  >
    {({ device }) => (
      <ChannelsForm
        device={device}
        task={task}
        isRunning={isRunning}
        isSnapshot={isSnapshot}
      />
    )}
  </Common.Device.Provider>
);

const getInitialPayload: Common.Task.GetInitialPayload<
  ReadConfig,
  ReadStateDetails,
  ReadType
> = (deviceKey) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

const onConfigure = async (client: Synnax, config: ReadConfig): Promise<ReadConfig> => {
  const dev = await client.hardware.devices.retrieve<Device.Properties>(config.device);
  let shouldCreateIndex = false;
  if (dev.properties.readIndex)
    try {
      await client.channels.retrieve(dev.properties.readIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw e;
    }
  else shouldCreateIndex = true;
  let modified = false;
  if (shouldCreateIndex) {
    modified = true;
    const index = await client.channels.create({
      name: `${dev.properties.identifier}_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    dev.properties.readIndex = index.key;
  }
  const toCreate: ReadChannel[] = [];
  for (const c of config.channels) {
    const type = getPortTypeFromChannelType(c.type);
    const existing = dev.properties[type].channels[c.port];
    // check if the channel is in properties
    if (primitiveIsZero(existing)) toCreate.push(c);
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
        name: `${dev.properties.identifier}_${c.port}`,
        dataType: c.type === DI_CHANNEL_TYPE ? "uint8" : "float32",
        index: dev.properties.readIndex,
      })),
    );
    channels.forEach((c, i) => {
      const toCreateC = toCreate[i];
      const type = getPortTypeFromChannelType(toCreateC.type);
      dev.properties[type].channels[toCreateC.port] = c.key;
    });
  }
  if (modified) await client.hardware.devices.create(dev);
  config.channels.forEach(
    (c) =>
      (c.channel = dev.properties[getPortTypeFromChannelType(c.type)].channels[c.port]),
  );
  return config;
};

export const ReadTask = Common.Task.wrapForm(<Properties />, Form, {
  configSchema: readConfigZ,
  type: READ_TYPE,
  getInitialPayload,
  onConfigure,
});
