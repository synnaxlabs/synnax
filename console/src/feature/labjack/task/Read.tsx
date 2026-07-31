// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Flex, Form as PForm, Icon } from "@synnaxlabs/pluto";
import { deep, errors, id, primitive } from "@synnaxlabs/x";
import { type FC, useCallback } from "react";

import { Select } from "@/feature/labjack/device/Select";
import { SelectPort } from "@/feature/labjack/device/SelectPort";
import * as Device from "@/feature/labjack/device/types";
import { useConfigureModal } from "@/feature/labjack/device/useConfigureModal";
import { convertChannelTypeToPortType } from "@/feature/labjack/task/convertChannelTypeToPortType";
import { getOpenPort } from "@/feature/labjack/task/getOpenPort";
import { FORMS } from "@/feature/labjack/task/InputChannelForms";
import { SelectInputChannelTypeField } from "@/feature/labjack/task/SelectInputChannelTypeField";
import {
  INPUT_CHANNEL_SCHEMAS,
  type InputChannel,
  type InputChannelType,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadSchemas,
  ZERO_INPUT_CHANNEL,
  ZERO_INPUT_CHANNELS,
  ZERO_READ_PAYLOAD,
} from "@/feature/labjack/task/types";
import { Device as PlatformDevice } from "@/platform/device";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "LabJack Read Task",
  icon: <Icon.Logo.LabJack />,
  useOnSelect: Task.createOpenTab(READ_TYPE),
});

const Properties = () => (
  <>
    <Select />
    <Flex.Box x>
      <Task.Fields.SampleRate />
      <Task.Fields.StreamRate />
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const getRenderedPort = (
  port: string,
  deviceModel: Device.Model,
  type: InputChannelType,
) => {
  const portType = convertChannelTypeToPortType(type);
  const portInfo = Device.PORTS[deviceModel][portType].find(({ key }) => key === port);
  return portInfo == null ? port : (portInfo.alias ?? portInfo.key);
};

interface ChannelListItemProps extends Task.ChannelListItemProps {
  onTare: (channelKey: channel.Key) => void;
  deviceModel: Device.Model;
}

const ChannelListItem = ({ onTare, deviceModel, ...rest }: ChannelListItemProps) => {
  const path = `config.channels.${rest.itemKey}`;
  const channel = PForm.useFieldValue<channel.Key>(`${path}.channel`);
  const port = PForm.useFieldValue<string>(`${path}.port`);
  const enabled = PForm.useFieldValue<boolean>(`${path}.enabled`);
  const type = PForm.useFieldValue<InputChannelType>(`${path}.type`);
  const isSnapshot = Task.useIsSnapshot();
  const isRunning = Task.useIsRunning();
  const hasTareButton = channel !== 0 && type === "AI" && !isSnapshot;
  const canTare = enabled && isRunning;
  const renderedPort = getRenderedPort(port, deviceModel, type);
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...rest}
      port={renderedPort}
      canTare={canTare}
      onTare={onTare}
      path={path}
      hasTareButton={hasTareButton}
      channel={channel}
      portMaxChars={5}
    />
  );
};

interface ChannelDetailsProps extends Task.Views.DetailsProps {
  deviceModel: Device.Model;
}

const ChannelDetails = ({ path, deviceModel }: ChannelDetailsProps) => {
  const channel = PForm.useFieldValue<InputChannel>(path);
  const Form = FORMS[channel.type];
  return (
    <>
      <Flex.Box x>
        <SelectInputChannelTypeField
          path={path}
          grow
          onChange={(value, { get, path, set }) => {
            if (value == null) return;
            const prevType = get<InputChannelType>(path).value;
            if (prevType === value) return;
            const next = deep.copy(ZERO_INPUT_CHANNELS[value]);
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<InputChannel>(parentPath).value;
            const schema = INPUT_CHANNEL_SCHEMAS[value];
            const nextParent = deep.overrideValidItems(next, prevParent, schema);
            const prevPortType = convertChannelTypeToPortType(prevType);
            const nextPortType = convertChannelTypeToPortType(value);
            let nextPort = nextParent.port;
            if (prevPortType !== nextPortType)
              nextPort = Device.PORTS[deviceModel][nextPortType][0].key;
            set(parentPath, { ...nextParent, type: next.type });
            // Need to explicitly set port to cause select port field to rerender
            set(`${parentPath}.port`, nextPort);
          }}
        />
        <PForm.Field<string> path={`${path}.port`}>
          {({ value, onChange, variant }) => (
            <SelectPort
              value={value}
              onChange={onChange}
              model={deviceModel}
              portType={convertChannelTypeToPortType(channel.type)}
              triggerProps={{ variant }}
            />
          )}
        </PForm.Field>
      </Flex.Box>
      <Form deviceModel={deviceModel} path={path} />
    </>
  );
};

const getOpenChannel = (
  channels: InputChannel[],
  device: Device.Device,
  channelKeyToCopy?: string,
) => {
  if (channelKeyToCopy == null)
    return { ...deep.copy(ZERO_INPUT_CHANNEL), key: id.create() };
  const channelToCopy = channels.find(({ key }) => key === channelKeyToCopy);
  if (channelToCopy == null) return null;
  // preferredPortType is AI or DI
  const preferredPortType = convertChannelTypeToPortType(channelToCopy.type);
  // backupPortType is the opposite of preferredPortType
  const backupPortType =
    preferredPortType === Device.DI_PORT_TYPE
      ? Device.AI_PORT_TYPE
      : Device.DI_PORT_TYPE;
  const port = getOpenPort(channels, device.model, [preferredPortType, backupPortType]);
  if (port == null) return null;
  // Now we need to determine what channel type we use the schema and zero channel for.
  // Note that if the copied channel was a TC channel, then we need to grab
  // channelToCopy.type instead of port.type as port.type cannot be TC.
  const channelTypeUsed =
    port.type === preferredPortType ? channelToCopy.type : backupPortType;
  return {
    ...deep.overrideValidItems(
      ZERO_INPUT_CHANNELS[channelTypeUsed],
      channelToCopy,
      INPUT_CHANNEL_SCHEMAS[channelTypeUsed],
    ),
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
    port: port.key,
    channel: device.properties[port.type].channels[port.key] ?? 0,
  };
};

interface ChannelsFormProps {
  device: Device.Device;
}

const isChannelTareable = (channel: InputChannel) => channel.type === "AI";

const ChannelsForm = ({ device }: ChannelsFormProps) => {
  const [tare, allowTare, handleTare] = Task.useTare({ isChannelTareable });
  const createChannel = useCallback(
    (channels: InputChannel[], channelKeyToCopy?: string) =>
      getOpenChannel(channels, device, channelKeyToCopy),
    [device],
  );
  const listItem = useCallback(
    ({ key, ...p }: Task.ChannelListItemProps) => (
      <ChannelListItem {...p} onTare={tare} key={key} deviceModel={device.model} />
    ),
    [tare, device.model],
  );
  const details = useCallback(
    (p: Task.Views.DetailsProps) => (
      <ChannelDetails {...p} deviceModel={device.model} />
    ),
    [device.model],
  );
  return (
    <Task.Views.ListAndDetails<InputChannel>
      listItem={listItem}
      details={details}
      createChannel={createChannel}
      onTare={handleTare}
      allowTare={allowTare}
      contextMenuItems={Task.readChannelContextMenuItem}
    />
  );
};

const Form: FC<Task.FormProps<ReadSchemas>> = (props) => {
  const isSnapshot = Task.useIsSnapshot();
  const configure = useConfigureModal();
  return (
    <PlatformDevice.Provider
      canConfigure={!isSnapshot}
      onConfigure={(deviceKey) => configure({ deviceKey })}
      schemas={Device.SCHEMAS}
    >
      {({ device }) => <ChannelsForm device={device} {...props} />}
    </PlatformDevice.Provider>
  );
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg =
    config != null ? READ_SCHEMAS.config.parse(config) : ZERO_READ_PAYLOAD.config;
  return {
    ...ZERO_READ_PAYLOAD,
    config: { ...cfg, device: deviceKey ?? cfg.device },
  };
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: Device.SCHEMAS,
  });
  PlatformDevice.checkConfigured(dev);
  let shouldCreateIndex = false;
  if (dev.properties.readIndex)
    try {
      await client.channels.retrieve(dev.properties.readIndex);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw errors.fromUnknown(e);
    }
  else shouldCreateIndex = true;
  let modified = false;
  const identifier = channel.escapeInvalidName(dev.properties.identifier);
  try {
    if (shouldCreateIndex) {
      modified = true;
      const index = await client.channels.create({
        name: `${identifier}_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.readIndex = index.key;
    }
    const toCreate: InputChannel[] = [];
    for (const c of config.channels) {
      const type = convertChannelTypeToPortType(c.type);
      const existing = dev.properties[type].channels[c.port];
      // check if the channel is in properties
      if (primitive.isZero(existing)) toCreate.push(c);
      else
        try {
          await client.channels.retrieve(existing.toString());
        } catch (e) {
          if (NotFoundError.matches(e)) toCreate.push(c);
          else throw errors.fromUnknown(e);
        }
    }
    if (toCreate.length > 0) {
      modified = true;
      const channels = await client.channels.create(
        toCreate.map((c) => ({
          name: primitive.isNonZero(c.name) ? c.name : `${identifier}_${c.port}`,
          dataType: c.type === "DI" ? "uint8" : "float32",
          index: dev.properties.readIndex,
        })),
      );
      channels.forEach((c, i) => {
        const toCreateC = toCreate[i];
        const type = convertChannelTypeToPortType(toCreateC.type);
        dev.properties[type].channels[toCreateC.port] = c.key;
      });
    }
  } finally {
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }
  config.channels.forEach(
    (c) =>
      (c.channel =
        dev.properties[convertChannelTypeToPortType(c.type)].channels[c.port]),
  );
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  Icon: Icon.Logo.LabJack,
  schemas: READ_SCHEMAS,
  type: "labjack_read",
  getInitialValues,
  onConfigure,
});
