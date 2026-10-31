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
import { List } from "@synnaxlabs/lyra/list";
import { Telem } from "@synnaxlabs/pluto";
import { DataType, errors, id, primitive } from "@synnaxlabs/x";
import { type FC, type ReactElement } from "react";

import { useFromConfig } from "@/feature/modbus/device/queries";
import { Select as SelectDevice } from "@/feature/modbus/device/Select";
import { MAKE, SCHEMAS } from "@/feature/modbus/device/types";
import { AddressField } from "@/feature/modbus/task/AddressField";
import { ByteOrderField } from "@/feature/modbus/task/ByteOrder";
import { RegisterField, SHADOW_TRIGGER_PROPS } from "@/feature/modbus/task/Register";
import {
  RegisterMap,
  RegisterMapProvider,
  useHoveredRef,
  useIsHovered,
  useOverlaps,
} from "@/feature/modbus/task/RegisterMap";
import {
  BIT_BOUNDS,
  deployReadConfigZ,
  isStringChannel,
  isVariableDensityReadChannel,
  nextChannel,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  readChannelName,
  readMapKey,
  type ReadSchemas,
  STRING_LENGTH_BOUNDS,
  type TypedReadChannel,
  UNSUPPORTED_READ_DATA_TYPES,
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
    </Flex.Box>
  </>
);

const COLUMNS: List.Column[] = [
  { name: "Type", width: "30rem", field: true },
  { name: "Address", width: "24rem", field: true },
  { name: "Data type", width: "32rem", field: true },
  { name: "Byte order", width: "27rem", field: true },
];

const DataTypeField = ({ path }: { path: string }): ReactElement | null => {
  const field = PForm.useField<string>(`${path}.dataType`);
  if (field == null) return null;
  return (
    <Telem.SelectDataType
      value={field.value}
      onChange={field.onChange}
      preview={field.preview}
      hideDataTypes={UNSUPPORTED_READ_DATA_TYPES}
      location="bottom"
      triggerProps={SHADOW_TRIGGER_PROPS}
    />
  );
};

interface NumberFieldProps {
  path: string;
  label: string;
  bounds: { lower: number; upper: number };
  /** The shorter name in front of the number, when the cell already gives context. */
  prefix?: string;
}

// A whole number in a cell, named by a prefix in front of it.
const NumberField = ({
  path,
  label,
  bounds,
  prefix = label,
}: NumberFieldProps): ReactElement => (
  <PForm.NumericField
    path={path}
    showLabel={false}
    showHelpText={false}
    inputProps={{
      variant: "shadow",
      showDragHandle: false,
      bounds,
      startContent: prefix,
      "aria-label": label,
    }}
  />
);

// A register shows its data type, and a string also its length. A register bit has no
// data type, so the cell holds its bit.
const DataTypeCell = ({
  path,
  channel,
}: {
  path: string;
  channel: ReadChannel;
}): ReactElement | null => {
  if ("bit" in channel)
    return <NumberField path={`${path}.bit`} label="Bit" bounds={BIT_BOUNDS} />;
  if (!isVariableDensityReadChannel(channel)) return null;
  return (
    <Flex.Box x align="center" className={CSS.BE("modbus", "data-type")}>
      <DataTypeField path={path} />
      {isStringChannel(channel) && (
        <NumberField
          path={`${path}.stringLength`}
          label="String length"
          prefix="Length"
          bounds={STRING_LENGTH_BOUNDS}
        />
      )}
    </Flex.Box>
  );
};

const ChannelListItem = ({ itemKey, ...rest }: Task.ChannelListItemProps) => {
  const path = `config.channels.${itemKey}`;
  const ch = PForm.useFieldValue<ReadChannel>(path);
  const dev = useFromConfig();
  const hovered = useIsHovered(itemKey);
  const hoveredRef = useHoveredRef(itemKey);
  const overlapping = useOverlaps(itemKey);
  return (
    <Task.Views.ListChannelItem
      {...rest}
      itemKey={itemKey}
      path={path}
      channel={ch.channel}
      defaultName={
        dev == null
          ? undefined
          : readChannelName(channel.escapeInvalidName(dev.name), ch)
      }
      ref={hoveredRef}
      hovered={hovered}
    >
      <List.Cell>
        <RegisterField path={path} />
      </List.Cell>
      <List.Cell>
        <AddressField path={path} variant="shadow" overlapping={overlapping} />
      </List.Cell>
      <List.Cell>
        <DataTypeCell path={path} channel={ch} />
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
  return {
    ...nextChannel(channels[channels.length - 1]),
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
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
    <RegisterMapProvider>
      <Task.Views.List<ReadChannel>
        createChannel={getOpenChannel}
        contextMenuItems={Task.readChannelContextMenuItem}
        listItem={listItem}
        resolve={resolve}
        header={HEADER}
        columns={COLUMNS}
        checkable
      />
    </RegisterMapProvider>
  );
};

const getInitialValues: Task.GetInitialValues<ReadSchemas> = ({ deviceKey }) => {
  const config = READ_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  config.channels = [getOpenChannel([])];
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
          name: primitive.isNonZero(c.name) ? c.name : readChannelName(safeName, c),
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
  deviceMake: MAKE,
});

export const ReadSelectable = Selector.createSelectable({
  type: READ_TYPE,
  title: "Modbus read task",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateRead,
});
