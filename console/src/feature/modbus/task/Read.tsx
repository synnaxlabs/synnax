// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/modbus/task/Task.css";

import { channel, NotFoundError } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, errors, id, primitive } from "@synnaxlabs/x";
import { type CSSProperties, type FC, type ReactElement } from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import { Select as SelectDevice } from "@/feature/modbus/device/Select";
import { SCHEMAS } from "@/feature/modbus/device/types";
import { ByteOrderField } from "@/feature/modbus/task/ByteOrder";
import { RegisterField, SHADOW_TRIGGER_PROPS } from "@/feature/modbus/task/Register";
import {
  HoveredProvider,
  RegisterMap,
  useColor,
  useHoveredRef,
  useIsHovered,
  useOverlaps,
} from "@/feature/modbus/task/RegisterMap";
import {
  addressSpan,
  deployReadConfigZ,
  isVariableDensityReadChannel,
  nextAddress,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadChannelType,
  type ReadSchemas,
  type TypedReadChannel,
} from "@/feature/modbus/task/types";
import { CSS } from "@/platform/css";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = () => (
  <>
    <SelectDevice />
    <Flex.Box x grow>
      <Task.Fields.SampleRate />
      <Task.Fields.StreamRate />
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const COLUMNS: List.Column[] = [
  { name: "Type", width: "30rem", field: true },
  { name: "Address", width: "10rem", field: true },
  { name: "Data type", width: "18rem", field: true },
  { name: "Byte order", width: "27rem", field: true },
];

interface AddressFieldProps {
  path: string;
  channel: ReadChannel;
}

// A channel that spans several addresses shows its last one after the first. An
// address that another channel also reads shows in red.
const AddressField = ({ path, channel }: AddressFieldProps): ReactElement | null => {
  const field = PForm.useField<number>(`${path}.address`);
  const overlapping = useOverlaps(channel.key);
  if (field == null) return null;
  const [lo, hi] = addressSpan(channel);
  return (
    <div
      className={CSS.cls(
        CSS.BE("modbus", "address"),
        overlapping && CSS.BEM("modbus", "address", "overlap"),
      )}
      style={{ "--console-modbus-digits": String(lo).length } as CSSProperties}
    >
      <Input.Numeric
        value={field.value}
        onChange={field.onChange}
        preview={field.preview}
        variant="shadow"
        showDragHandle={false}
        aria-label="Address"
        bounds={ADDRESS_BOUNDS}
      />
      {hi > lo && <span className={CSS.BE("modbus", "address-end")}>{`–${hi}`}</span>}
    </div>
  );
};

const ADDRESS_BOUNDS = { lower: 0, upper: 65535 };

const DataTypeField = ({ path }: { path: string }): ReactElement | null => {
  const field = PForm.useField<string>(`${path}.dataType`);
  if (field == null) return null;
  return (
    <Telem.SelectDataType
      value={field.value}
      onChange={field.onChange}
      preview={field.preview}
      hideVariableDensity
      location="bottom"
      triggerProps={SHADOW_TRIGGER_PROPS}
    />
  );
};

const ChannelListItem = ({ itemKey, ...rest }: Task.ChannelListItemProps) => {
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<ReadChannel>(path);
  const dev = useFromConfig();
  const hovered = useIsHovered(itemKey);
  const hoveredRef = useHoveredRef(itemKey);
  const color = useColor(itemKey);
  return (
    <Task.Views.ListChannelItem
      {...rest}
      itemKey={itemKey}
      path={path}
      channel={ch.channel}
      defaultName={
        dev == null ? undefined : channelName(channel.escapeInvalidName(dev.name), ch)
      }
      ref={hoveredRef}
      hovered={hovered}
      style={{ "--console-modbus-color": color } as CSSProperties}
    >
      <List.Cell>
        <RegisterField path={path} />
      </List.Cell>
      <List.Cell>
        <AddressField path={path} channel={ch} />
      </List.Cell>
      <List.Cell>
        {isVariableDensityReadChannel(ch) && <DataTypeField path={path} />}
      </List.Cell>
      <List.Cell>
        {isVariableDensityReadChannel(ch) && (
          <ByteOrderField path={path} dataType={ch.dataType} />
        )}
      </List.Cell>
    </Task.Views.ListChannelItem>
  );
};

const getOpenChannel = (channels: ReadChannel[]): ReadChannel => {
  if (channels.length === 0)
    return {
      ...READ_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
      key: id.create(),
    };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
    address: nextAddress(channelToCopy),
  };
};

const listItem = Component.renderProp(ChannelListItem);

const HEADER = (
  <>
    <Header.Header>
      <Header.Title weight={500} color={10}>
        Channels
      </Header.Title>
    </Header.Header>
    <RegisterMap />
  </>
);

const Form: FC = () => {
  const dev = useFromConfig();
  const resolve = (c: ReadChannel) =>
    dev == null ? null : { channel: dev.properties.read.channels[readMapKey(c)] ?? 0 };
  return (
    <HoveredProvider>
      <Task.Views.List<ReadChannel>
        createChannel={getOpenChannel}
        contextMenuItems={Task.readChannelContextMenuItem}
        listItem={listItem}
        resolve={resolve}
        header={HEADER}
        columns={COLUMNS}
        checkable
      />
    </HoveredProvider>
  );
};

// Auto-generated channel names and device map keys keep the released type
// spellings, so channels created before the labels were renamed keep matching.
const NAME_TYPES: Record<ReadChannelType, string> = {
  coil: "coil_input",
  discrete_input: "discrete_input",
  holding_register: "holding_register_input",
  input_register: "register_input",
};

const readMapKey = (channel: ReadChannel) => {
  let s = `${NAME_TYPES[channel.type]}-${channel.address.toString()}`;
  if (isVariableDensityReadChannel(channel)) s += `-${channel.dataType}`;
  return s.replaceAll("_", "-");
};

const channelName = (deviceName: string, channel: ReadChannel) => {
  let s = `${deviceName}_${NAME_TYPES[channel.type]}_${channel.address}`;
  if (isVariableDensityReadChannel(channel))
    s += `_${new DataType(channel.dataType).toString(true)}`;
  return s;
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ deviceKey }) => {
  const config = READ_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  return { name: "Modbus read task", type: READ_TYPE, config };
};

const onConfigure: Task.OnConfigure<ReadSchemas["config"]> = async (client, config) => {
  const dev = await client.devices.retrieve({
    key: config.device,
    schemas: SCHEMAS,
  });
  let shouldCreateIndex = false;
  if (dev.properties.read.index)
    try {
      await client.channels.retrieve(dev.properties.read.index);
    } catch (e) {
      if (NotFoundError.matches(e)) shouldCreateIndex = true;
      else throw errors.fromUnknown(e);
    }
  else shouldCreateIndex = true;
  let modified = false;
  const safeName = channel.escapeInvalidName(dev.name);
  try {
    if (shouldCreateIndex) {
      modified = true;
      const index = await client.channels.create({
        name: `${safeName}_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      dev.properties.read.index = index.key;
    }

    const toCreate: ReadChannel[] = [];
    for (const c of config.channels) {
      const key = readMapKey(c);
      const existing = dev.properties.read.channels[key];
      if (existing == null) toCreate.push(c);
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
          name: primitive.isNonZero(c.name) ? c.name : channelName(safeName, c),
          dataType: (c as TypedReadChannel).dataType ?? DataType.UINT8.toString(),
          index: dev.properties.read.index,
        })),
      );

      channels.forEach((c, i) => {
        const channel = toCreate[i];
        dev.properties.read.channels[readMapKey(channel)] = c.key;
      });
    }
  } finally {
    if (modified) await client.devices.create(dev, SCHEMAS);
  }

  config.channels.forEach((c) => {
    c.channel = dev.properties.read.channels[readMapKey(c)];
  });

  return [config, dev.rack];
};

export const Read = Task.wrapForm({
  Properties,
  Form,
  schemas: READ_SCHEMAS,
  deployConfigZ: deployReadConfigZ,
  type: "modbus_read",
  getInitialValues,
  onConfigure,
});

export const useCreateRead = Task.createUseCreate({
  getInitialValues,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "Modbus read task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateRead,
});
