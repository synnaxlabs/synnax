// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { deep, errors, id, primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { use } from "@/feature/labjack/device/queries";
import { Select, useNoneSelected } from "@/feature/labjack/device/Select";
import {
  AI_PORT_TYPE,
  type AIPort,
  type Device,
  DI_PORT_TYPE,
  type DIPort,
  MAKE,
  type Model,
  negativeChannelOf,
  PORTS,
  SCHEMAS,
} from "@/feature/labjack/device/types";
import { thermocoupleRateZ, uniqueTerminalsZ } from "@/feature/labjack/task/checks";
import {
  convertPortTypeToReadChannelType,
  convertReadChannelTypeToPortType,
} from "@/feature/labjack/task/convertChannelTypeToPortType";
import { getOpenPort } from "@/feature/labjack/task/getOpenPort";
import { PortField } from "@/feature/labjack/task/PortField";
import { maxTerminalChars, terminalOf } from "@/feature/labjack/task/ports";
import { PortsView } from "@/feature/labjack/task/PortsView";
import { FORMS } from "@/feature/labjack/task/ReadChannelForms";
import {
  READ_CHANNEL_TYPE_ITEMS,
  READ_CHANNEL_TYPE_NAMES,
  SelectReadChannelTypeField,
} from "@/feature/labjack/task/SelectReadChannelTypeField";
import {
  createReadChannel,
  deployReadConfigZ,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadSchemas,
} from "@/feature/labjack/task/types";
import { useConfigure } from "@/feature/labjack/task/useConfigure";
import { Device as PlatformDevice } from "@/platform/device";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const root = deep.path<Task.Values<ReadSchemas>>();

const portOf = (c: ReadChannel): string =>
  `${convertReadChannelTypeToPortType(c.type)}/${c.port}`;

const Properties = () => (
  <>
    <Select />
    <Flex.Box x wrap>
      <Task.Fields.SampleRate />
      <Task.Fields.StreamRate />
    </Flex.Box>
  </>
);

/** The name a channel shows by: its own, or the one configuring the task gives it. */
const channelName = (
  { name, port }: Pick<ReadChannel, "name" | "port">,
  identifier: string,
): string => (primitive.isNonZero(name) ? name : `${identifier}_${port}`);

interface ChannelListItemProps extends Task.ChannelListItemProps {
  model: Model;
  /** The device's identifier, escaped for channel names. */
  identifier: string;
}

const ChannelListItem = ({ model, identifier, ...rest }: ChannelListItemProps) => {
  const entry = root.config.channels[rest.itemKey];
  const channel = PForm.useFieldValue(entry.channel);
  const name = PForm.useFieldValue(entry.name);
  const port = PForm.useFieldValue(entry.port);
  const skipped = Task.useSkipped(root.config.channels, rest.itemKey, portOf);
  return (
    <Task.Views.ListAndDetailsChannelItem
      {...rest}
      port={terminalOf(model, port)}
      path={entry}
      channel={channel}
      defaultName={channelName({ name, port }, identifier)}
      skipped={skipped}
      portMaxChars={maxTerminalChars(model)}
    />
  );
};

interface DetailsTitleProps extends Task.Views.DetailsProps<ReadChannel> {
  model: Model;
}

const DetailsTitle = ({ path, model }: DetailsTitleProps) => {
  const port = PForm.useFieldValue(path.port);
  const type = PForm.useFieldValue(path.type);
  return (
    <Task.Views.ItemLabel kind={READ_CHANNEL_TYPE_NAMES[type]}>
      Port {terminalOf(model, port)}
    </Task.Views.ItemLabel>
  );
};

interface ChannelDetailsProps extends Task.Views.DetailsProps<ReadChannel> {
  device: Device;
}

const ChannelDetails = ({ path, device }: ChannelDetailsProps) => {
  const { model } = device;
  const identifier = channel.escapeInvalidName(device.properties.identifier);
  const ch = PForm.useFieldValue(path);
  const TypeForm = FORMS[model][ch.type];
  return (
    <PForm.Sections>
      <PForm.Section title="Source">
        <SelectReadChannelTypeField
          path={path.type}
          inputProps={{ children: READ_CHANNEL_TYPE_ITEMS[model] }}
          onChange={(value, { get, set }) => {
            if (value == null) return;
            const prevType = get(path.type).value;
            if (prevType === value) return;
            const next = createReadChannel(value);
            const prevParent = get(path).value;
            const schema = READ_CHANNEL_SCHEMAS[value];
            const nextParent = deep.overrideValidItems(next, prevParent, schema);
            const prevPortType = convertReadChannelTypeToPortType(prevType);
            const nextPortType = convertReadChannelTypeToPortType(value);
            let nextPort = nextParent.port;
            if (prevPortType !== nextPortType) {
              const others = get(root.config.channels).value.filter(
                ({ key }) => key !== prevParent.key,
              );
              nextPort = (
                getOpenPort(others, model, [nextPortType]) ??
                PORTS[model][nextPortType][0]
              ).key;
            }
            set(path, {
              ...nextParent,
              type: next.type,
              port: nextPort,
            } as ReadChannel);
          }}
        />
        <PortField<ReadChannel>
          path={path}
          channelsPath={root.config.channels}
          model={model}
          type={convertReadChannelTypeToPortType(ch.type)}
          name={(c) => channelName(c, identifier)}
        />
      </PForm.Section>
      {TypeForm != null && (
        <PForm.Section title="Signal">
          <TypeForm deviceModel={model} path={path} />
        </PForm.Section>
      )}
      {"scale" in ch && (
        <PForm.Section title="Scale">
          <Task.ScaleForm path={deep.variant(path, "type", ch.type).scale} />
        </PForm.Section>
      )}
    </PForm.Sections>
  );
};

const createFirstChannel = (): ReadChannel => ({
  ...createReadChannel("analog"),
  key: id.create(),
});

/**
 * Creates a channel on the port. It copies the template when the template's type fits
 * the port, which keeps a thermocouple a thermocouple on an AI port.
 */
const createChannelOn = (
  port: AIPort | DIPort,
  template?: ReadChannel,
): ReadChannel => {
  const type =
    template != null && convertReadChannelTypeToPortType(template.type) === port.type
      ? template.type
      : convertPortTypeToReadChannelType(port.type);
  const zero = createReadChannel(type);
  const next: ReadChannel = {
    ...(template == null
      ? zero
      : deep.overrideValidItems(zero, template, READ_CHANNEL_SCHEMAS[type])),
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
    port: port.key,
  };
  // A differential copy reads against the pair of its own port.
  if ("negChan" in next && next.negChan != null)
    next.negChan = negativeChannelOf(port.key) ?? undefined;
  return next;
};

/**
 * Creates a copy of the channel with the key, or an analog channel without one, on the
 * lowest free port of its type, then of the other type. Null when the channel to copy
 * is gone or every port is taken.
 */
const createNextChannel = (
  channels: ReadChannel[],
  model: Model,
  copy?: string,
): ReadChannel | null => {
  const template = copy == null ? undefined : channels.find(({ key }) => key === copy);
  if (copy != null && template == null) return null;
  const preferred =
    template == null ? AI_PORT_TYPE : convertReadChannelTypeToPortType(template.type);
  const backup = preferred === DI_PORT_TYPE ? AI_PORT_TYPE : DI_PORT_TYPE;
  const port = getOpenPort(channels, model, [preferred, backup]);
  return port == null ? null : createChannelOn(port, template);
};

interface ChannelsFormProps {
  device: Device;
}

const ChannelsForm = ({ device }: ChannelsFormProps) => {
  const { model } = device;
  const identifier = channel.escapeInvalidName(device.properties.identifier);
  PForm.useCheck(root.config, uniqueTerminalsZ(model));
  PForm.useCheck(root.config, thermocoupleRateZ(model));
  const createChannel = (channels: ReadChannel[], copy?: string) =>
    createNextChannel(channels, model, copy);
  const listItem = ({ key, ...p }: Task.ChannelListItemProps) => (
    <ChannelListItem key={key} {...p} model={model} identifier={identifier} />
  );
  const details = (p: Task.Views.DetailsProps<ReadChannel>) => (
    <ChannelDetails {...p} device={device} />
  );
  const resolve = (c: ReadChannel) => ({
    channel:
      device.properties[convertReadChannelTypeToPortType(c.type)].channels[c.port] ?? 0,
  });
  const detailsTitle = (p: Task.Views.DetailsProps<ReadChannel>) => (
    <DetailsTitle {...p} model={model} />
  );
  const ports = (p: Task.Views.PortsProps<ReadChannel>) => (
    <PortsView<ReadChannel, AIPort | DIPort>
      {...p}
      model={model}
      ports={[...PORTS[model][AI_PORT_TYPE], ...PORTS[model][DI_PORT_TYPE]]}
      create={(channels, port) =>
        createChannelOn(
          port,
          channels.find((c) => convertReadChannelTypeToPortType(c.type) === port.type),
        )
      }
      name={(c) => channelName(c, identifier)}
    />
  );
  return (
    <Task.Views.ListAndDetails<ReadChannel>
      path={root.config.channels}
      listItem={listItem}
      details={details}
      detailsTitle={detailsTitle}
      createChannel={createChannel}
      tareable
      contextMenuItems={Task.readChannelContextMenuItem}
      resolve={resolve}
      ports={ports}
    />
  );
};

const Form: FC = PlatformDevice.wrapTaskForm({
  use,
  useConfigure,
  useNoneSelected,
  Content: ChannelsForm,
});

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createFirstChannel()];
  return { name: "LabJack read task", type: READ_TYPE, config: cfg };
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  const before = deep.copy(dev.properties);
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
    for (const c of Task.firstPerPort(config.channels, portOf)) {
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
          name: channelName(c, identifier),
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
    if (modified) await PlatformDevice.setProperties(client, dev, before, SCHEMAS);
  }
  config.channels.forEach(
    (c) =>
      (c.channel =
        dev.properties[convertReadChannelTypeToPortType(c.type)].channels[c.port]),
  );
  return [config, dev.rack];
};

/** @returns the channels that a plot of the read task shows. */
export const readPlotChannels: Task.PlotChannels = (config) =>
  Task.enabledChannels(READ_SCHEMAS.config.parse(config).channels);

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "labjack_read",
  onConfigure,
  plotChannels: readPlotChannels,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "LabJack read task",
  label: "Read",
  icon: <Icon.Logo.LabJack />,
  useOnSelect: useCreateRead,
};
