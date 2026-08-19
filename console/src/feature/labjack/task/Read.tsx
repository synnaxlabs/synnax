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

import { use } from "@/feature/labjack/device/queries";
import { Select } from "@/feature/labjack/device/Select";
import { SelectPort } from "@/feature/labjack/device/SelectPort";
import * as Device from "@/feature/labjack/device/types";
import { useConfigureModal } from "@/feature/labjack/device/useConfigureModal";
import {
  convertPortTypeToReadChannelType,
  convertReadChannelTypeToPortType,
} from "@/feature/labjack/task/convertChannelTypeToPortType";
import { getOpenPort } from "@/feature/labjack/task/getOpenPort";
import { FORMS } from "@/feature/labjack/task/ReadChannelForms";
import { SelectReadChannelTypeField } from "@/feature/labjack/task/SelectReadChannelTypeField";
import {
  createReadChannel,
  deployReadConfigZ,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadChannelType,
  type ReadSchemas,
} from "@/feature/labjack/task/types";
import { Device as PlatformDevice } from "@/platform/device";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

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
  type: ReadChannelType,
) => {
  const portType = convertReadChannelTypeToPortType(type);
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
  const disabled = PForm.useFieldValue<boolean>(`${path}.disabled`);
  const type = PForm.useFieldValue<ReadChannelType>(`${path}.type`);
  const isSnapshot = Task.useIsSnapshot();
  const isRunning = Task.useIsRunning();
  const hasTareButton = channel !== 0 && type === "analog" && !isSnapshot;
  const canTare = !disabled && isRunning;
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
  const channel = PForm.useFieldValue<ReadChannel>(path);
  const Form = FORMS[channel.type];
  return (
    <>
      <Flex.Box x>
        <SelectReadChannelTypeField
          path={path}
          grow
          onChange={(value, { get, path, set }) => {
            if (value == null) return;
            const prevType = get<ReadChannelType>(path).value;
            if (prevType === value) return;
            const next = createReadChannel(value);
            const parentPath = path.slice(0, path.lastIndexOf("."));
            const prevParent = get<ReadChannel>(parentPath).value;
            const schema = READ_CHANNEL_SCHEMAS[value];
            const nextParent = deep.overrideValidItems(next, prevParent, schema);
            const prevPortType = convertReadChannelTypeToPortType(prevType);
            const nextPortType = convertReadChannelTypeToPortType(value);
            let nextPort = nextParent.port;
            if (prevPortType !== nextPortType)
              nextPort = Device.PORTS[deviceModel][nextPortType][0].key;
            set(parentPath, { ...nextParent, type: next.type });
            // Need to explicitly set port to cause select port field to rerender
            set(`${parentPath}.port`, nextPort);
          }}
        />
        <PForm.Field<string> path={`${path}.port`}>
          {({ value, onChange, preview }) => (
            <SelectPort
              value={value}
              onChange={onChange}
              model={deviceModel}
              portType={convertReadChannelTypeToPortType(channel.type)}
              preview={preview}
            />
          )}
        </PForm.Field>
      </Flex.Box>
      <Form deviceModel={deviceModel} path={path} />
    </>
  );
};

const getOpenChannel = (
  channels: ReadChannel[],
  device: Device.Device,
  channelKeyToCopy?: string,
) => {
  if (channelKeyToCopy == null)
    return { ...createReadChannel("analog"), key: id.create() };
  const channelToCopy = channels.find(({ key }) => key === channelKeyToCopy);
  if (channelToCopy == null) return null;
  // preferredPortType is AI or DI
  const preferredPortType = convertReadChannelTypeToPortType(channelToCopy.type);
  const backupPortType =
    preferredPortType === Device.DI_PORT_TYPE
      ? Device.AI_PORT_TYPE
      : Device.DI_PORT_TYPE;
  const port = getOpenPort(channels, device.model, [preferredPortType, backupPortType]);
  if (port == null) return null;
  // Now we need to determine what channel type we use the schema and zero channel
  // for. Note that if the copied channel was a thermocouple channel, then we need to
  // grab channelToCopy.type instead of port.type as no port type maps back to it.
  const channelTypeUsed =
    port.type === preferredPortType
      ? channelToCopy.type
      : convertPortTypeToReadChannelType(backupPortType);
  return {
    ...deep.overrideValidItems(
      createReadChannel(channelTypeUsed),
      channelToCopy,
      READ_CHANNEL_SCHEMAS[channelTypeUsed],
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

const isChannelTareable = (channel: ReadChannel) => channel.type === "analog";

const ChannelsForm = ({ device }: ChannelsFormProps) => {
  const [tare, allowTare, handleTare] = Task.useTare({ isChannelTareable });
  const createChannel = useCallback(
    (channels: ReadChannel[], channelKeyToCopy?: string) =>
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
    <Task.Views.ListAndDetails<ReadChannel>
      listItem={listItem}
      details={details}
      createChannel={createChannel}
      onTare={handleTare}
      allowTare={allowTare}
      contextMenuItems={Task.readChannelContextMenuItem}
    />
  );
};

const Form: FC = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConfigureModal,
  Content: ChannelsForm,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "LabJack read task", type: READ_TYPE, config: cfg };
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
    const toCreate: ReadChannel[] = [];
    for (const c of config.channels) {
      const type = convertReadChannelTypeToPortType(c.type);
      const existing = dev.properties[type].channels[c.port];
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
          dataType: c.type === "digital" ? "uint8" : "float32",
          index: dev.properties.readIndex,
        })),
      );
      channels.forEach((c, i) => {
        const toCreateC = toCreate[i];
        const type = convertReadChannelTypeToPortType(toCreateC.type);
        dev.properties[type].channels[toCreateC.port] = c.key;
      });
    }
  } finally {
    if (modified) await client.devices.create(dev, Device.SCHEMAS);
  }
  config.channels.forEach(
    (c) =>
      (c.channel =
        dev.properties[convertReadChannelTypeToPortType(c.type)].channels[c.port]),
  );
  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "labjack_read",
  getInitialValues,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "LabJack read task",
  icon: <Icon.Logo.LabJack />,
  useOnSelect: useCreateRead,
});
