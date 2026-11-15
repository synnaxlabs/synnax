// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { deep, errors, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { useFromConfig } from "@/feature/ni/device/queries";
import { Select } from "@/feature/ni/device/Select";
import { MAKE, SCHEMAS } from "@/feature/ni/device/types";
import { digitalHardwareZ } from "@/feature/ni/task/checks";
import { createNextDIChannel } from "@/feature/ni/task/createChannel";
import { getDigitalChannelDeviceKey } from "@/feature/ni/task/getDigitalChannelDeviceKey";
import { PORT_LINE_COLUMNS, PortLineCells } from "@/feature/ni/task/PortLine";
import { digitalInputName } from "@/feature/ni/task/ports";
import { LinesView } from "@/feature/ni/task/PortsView";
import {
  deployDigitalReadConfigZ,
  type DIChannel,
  DIGITAL_READ_SCHEMAS,
  DIGITAL_READ_TYPE,
  digitalReadConfigZ,
  type DigitalReadSchemas,
} from "@/feature/ni/task/types";
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<DigitalReadSchemas>>();

const Properties = () => (
  <>
    <Select />
    <Flex.Box x wrap>
      <Task.Fields.SampleRate />
      <Task.Fields.StreamRate />
    </Flex.Box>
  </>
);

const ChannelListItem = ({ itemKey, ...rest }: Task.ChannelListItemProps) => {
  const entry = root.config.channels[itemKey];
  const ch = PForm.useFieldValue(entry);
  const device = useFromConfig();
  const skipped = Task.useSkipped(
    root.config.channels,
    itemKey,
    getDigitalChannelDeviceKey,
  );
  return (
    <Task.Views.ListChannelItem
      {...rest}
      itemKey={itemKey}
      path={entry}
      channel={ch.channel}
      defaultName={device == null ? undefined : digitalInputName(ch, device)}
      skipped={skipped}
    >
      <PortLineCells path={entry} />
    </Task.Views.ListChannelItem>
  );
};

const listItem = Component.renderProp(ChannelListItem);

const Form: FC = () => {
  const dev = useFromConfig();
  PForm.useCheck(root.config, digitalHardwareZ(dev, "input"));
  const resolve = (ch: DIChannel) =>
    dev == null
      ? null
      : {
          channel:
            dev.properties.digitalInput.channels[getDigitalChannelDeviceKey(ch)] ?? 0,
        };
  const map =
    dev?.properties.hardware == null
      ? undefined
      : (props: Task.Views.PortsProps<DIChannel>) => (
          <LinesView<DIChannel>
            {...props}
            device={dev}
            direction="input"
            create={(channels, placement) =>
              createNextDIChannel(channels, placement, dev)
            }
            name={digitalInputName}
          />
        );
  return (
    <Task.Views.List<DIChannel>
      path={root.config.channels}
      createChannel={(channels) => createNextDIChannel(channels, {}, dev)}
      listItem={listItem}
      columns={PORT_LINE_COLUMNS}
      checkable
      contextMenuItems={Task.readChannelContextMenuItem}
      resolve={resolve}
      map={map}
    />
  );
};

const getInitialValues: Task.GetInitialValues<DigitalReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = digitalReadConfigZ.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createNextDIChannel([])];
  return { name: "NI digital read task", type: DIGITAL_READ_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<typeof digitalReadConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const before = deep.copy(dev.properties);
  PlatformDevice.checkConfigured(dev);
  let modified = false;
  let shouldCreateIndex = primitive.isZero(dev.properties.digitalInput.index);
  if (!shouldCreateIndex)
    try {
      await client.channels.retrieve(dev.properties.digitalInput.index);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw errors.fromUnknown(e);
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
    for (const channel of Task.firstPerPort(
      config.channels,
      getDigitalChannelDeviceKey,
    )) {
      const key = getDigitalChannelDeviceKey(channel);
      const exKey = dev.properties.digitalInput.channels[key];
      if (primitive.isZero(exKey)) toCreate.push(channel);
      else
        try {
          await client.channels.retrieve(exKey.toString());
        } catch (e) {
          if (NotFoundError.matches(e)) toCreate.push(channel);
          else throw errors.fromUnknown(e);
        }
    }
    if (toCreate.length > 0) {
      modified = true;
      const channels = await client.channels.create(
        toCreate.map((c) => ({
          name: digitalInputName(c, dev),
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
    if (modified) await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  config.channels.forEach((c) => {
    const key = getDigitalChannelDeviceKey(c);
    c.channel = dev.properties.digitalInput.channels[key];
  });
  return [config, dev.rack];
};

export const DigitalRead = Task.wrapForm({
  Properties,
  Form,
  schemas: DIGITAL_READ_SCHEMAS,
  deployConfigZ: deployDigitalReadConfigZ,
  onConfigure,
  type: "ni_digital_read",
});

export const useCreateDigitalRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const DigitalReadSelectable: Selector.Selectable = {
  type: DIGITAL_READ_TYPE,
  title: "NI digital read task",
  label: "Digital read",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateDigitalRead,
};
