// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError } from "@synnaxlabs/client";
import {
  Component,
  Flex,
  Form as PForm,
  Icon,
  Input,
  Select,
  Telem,
  Text,
} from "@synnaxlabs/pluto";
import { DataType, deep, id } from "@synnaxlabs/x";
import { type FC } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { Device } from "@/hardware/visa/device";
import {
  channelName,
  type InputChannel,
  READ_SCHEMAS,
  READ_TYPE,
  type readConfigZ,
  readMapKey,
  type readStatusDataZ,
  type readTypeZ,
  RESPONSE_FORMAT_BINARY_BLOCK,
  RESPONSE_FORMAT_BOOLEAN,
  RESPONSE_FORMAT_FLOAT,
  RESPONSE_FORMAT_FLOAT_ARRAY,
  RESPONSE_FORMAT_INTEGER,
  RESPONSE_FORMAT_STRING,
  type ResponseFormat,
  ZERO_INPUT_CHANNEL,
  ZERO_READ_PAYLOAD,
} from "@/hardware/visa/task/types";
import { type Selector } from "@/selector";

export const READ_LAYOUT = {
  ...Common.Task.LAYOUT,
  type: READ_TYPE,
  name: ZERO_READ_PAYLOAD.name,
  icon: "Chip",
} as const satisfies Common.Task.Layout;

export const READ_SELECTABLE = {
  key: READ_TYPE,
  title: "VISA Read Task",
  icon: <Icon.Chip />,
  create: async ({ layoutKey }) => ({ ...READ_LAYOUT, key: layoutKey }),
} as const satisfies Selector.Selectable;

const Properties = () => (
  <>
    <Device.Select />
    <Flex.Box x grow>
      <Common.Task.Fields.SampleRate
        label="Sample Rate"
        tooltip="How often to query the device for new data"
      />
      <Common.Task.Fields.StreamRate />
      <Common.Task.Fields.DataSaving />
      <Common.Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

const RESPONSE_FORMAT_OPTIONS: Array<{ key: ResponseFormat; name: string }> = [
  { key: RESPONSE_FORMAT_FLOAT, name: "Float" },
  { key: RESPONSE_FORMAT_INTEGER, name: "Integer" },
  { key: RESPONSE_FORMAT_STRING, name: "String" },
  { key: RESPONSE_FORMAT_FLOAT_ARRAY, name: "Float Array" },
  { key: RESPONSE_FORMAT_BINARY_BLOCK, name: "Binary Block" },
  { key: RESPONSE_FORMAT_BOOLEAN, name: "Boolean" },
];

const needsDataType = (format: ResponseFormat): boolean =>
  format === RESPONSE_FORMAT_FLOAT ||
  format === RESPONSE_FORMAT_INTEGER ||
  format === RESPONSE_FORMAT_FLOAT_ARRAY ||
  format === RESPONSE_FORMAT_BINARY_BLOCK ||
  format === RESPONSE_FORMAT_BOOLEAN;

const needsDelimiter = (format: ResponseFormat): boolean =>
  format === RESPONSE_FORMAT_FLOAT_ARRAY;

const needsArrayLength = (format: ResponseFormat): boolean =>
  format === RESPONSE_FORMAT_FLOAT_ARRAY;

// Auto-detect format based on common SCPI patterns
const detectFormat = (command: string): ResponseFormat => {
  const cmd = command.toUpperCase();

  // Binary waveform commands
  if (cmd.includes("CURV") || cmd.includes("WAV:DATA") || cmd.includes(":DATA:VAL"))
    return RESPONSE_FORMAT_BINARY_BLOCK;

  // Array commands
  if (cmd.includes("TRAC:DATA") || cmd.includes("FETC:ARR") || cmd.includes(":ARR"))
    return RESPONSE_FORMAT_FLOAT_ARRAY;

  // Boolean/status commands
  if (cmd.includes(":STAT") || cmd.includes("OUTP:") || cmd.includes(":ENAB"))
    return RESPONSE_FORMAT_BOOLEAN;

  // String commands
  if (cmd.includes("SYST:ERR") || cmd.includes("*IDN") || cmd.includes(":NAME"))
    return RESPONSE_FORMAT_STRING;

  // Integer commands
  if (cmd.includes(":COUN") || cmd.includes(":NUMB"))
    return RESPONSE_FORMAT_INTEGER;

  // Default to float for measurement commands
  return RESPONSE_FORMAT_FLOAT;
};

const ChannelListItem = (props: Common.Task.ChannelListItemProps) => {
  const { itemKey } = props;
  const path = `config.channels.${itemKey}`;
  const { format, channel, scpiCommand } = PForm.useFieldValue<InputChannel>(path);
  const { get, set } = PForm.useContext();

  const handleScpiCommandChange = (value: string) => {
    set(`${path}.scpiCommand`, value);

    // Auto-detect format if command is not empty and format is default
    if (value.trim() && format === RESPONSE_FORMAT_FLOAT) {
      const detectedFormat = detectFormat(value);
      set(`${path}.format`, detectedFormat);

      // Set appropriate data type based on format
      if (detectedFormat === RESPONSE_FORMAT_BOOLEAN) {
        set(`${path}.dataType`, DataType.UINT8.toString());
      } else if (detectedFormat === RESPONSE_FORMAT_INTEGER) {
        set(`${path}.dataType`, DataType.INT64.toString());
      }
    }
  };

  return (
    <Select.ListItem
      {...props}
      style={{ width: "100%" }}
      justify="between"
      align="center"
      direction="y"
    >
      <Flex.Box x pack className={CSS.B("channel-item")} align="center" grow>
        <PForm.Field<string>
          path={`${path}.scpiCommand`}
          label="SCPI Query"
          showLabel={false}
          showHelpText={false}
          hideIfNull
        >
          {({ value, ...p }) => (
            <Flex.Box direction="y" style={{ minWidth: 250 }}>
              <Input.Text
                placeholder="MEAS:VOLT? or SYST:ERR? or TRAC:DATA?"
                value={value}
                onChange={handleScpiCommandChange}
                {...p}
              />
              {value && (
                <Text.Text level="small" style={{ marginTop: 2, opacity: 0.6 }}>
                  Query sent to instrument
                </Text.Text>
              )}
            </Flex.Box>
          )}
        </PForm.Field>
        <PForm.Field<ResponseFormat>
          path={`${path}.format`}
          label="Response Format"
          showLabel={false}
          showHelpText={false}
          hideIfNull
        >
          {({ value, onChange }) => (
            <Flex.Box direction="y">
              <Select.Single
                value={value}
                onChange={onChange}
                data={RESPONSE_FORMAT_OPTIONS}
                entryRenderKey="name"
              />
              <Text.Text level="small" style={{ marginTop: 2, opacity: 0.6 }}>
                How to parse the response
              </Text.Text>
            </Flex.Box>
          )}
        </PForm.Field>
        {needsDataType(format) && (
          <PForm.Field<string>
            path={`${path}.dataType`}
            label="Data Type"
            showLabel={false}
            showHelpText={false}
            hideIfNull
          >
            {({ value, onChange }) => (
              <Telem.SelectDataType
                value={value}
                onChange={onChange}
                hideVariableDensity
                location="bottom"
              />
            )}
          </PForm.Field>
        )}
        {needsDelimiter(format) && (
          <PForm.Field<string>
            path={`${path}.delimiter`}
            label="Delimiter"
            showLabel={false}
            showHelpText={false}
            hideIfNull
          >
            {(p) => (
              <Flex.Box direction="y" style={{ width: 80 }}>
                <Input.Text placeholder="," {...p} />
                <Text.Text level="small" style={{ marginTop: 2, opacity: 0.6 }}>
                  Array separator
                </Text.Text>
              </Flex.Box>
            )}
          </PForm.Field>
        )}
        {needsArrayLength(format) && (
          <Flex.Box direction="y">
            <PForm.NumericField
              inputProps={{ showDragHandle: false, placeholder: "0 = any" }}
              hideIfNull
              showLabel={false}
              showHelpText={false}
              path={`${path}.arrayLength`}
            />
            <Text.Text level="small" style={{ marginTop: 2, opacity: 0.6 }}>
              Expected array size
            </Text.Text>
          </Flex.Box>
        )}
      </Flex.Box>
      <Flex.Box x align="center" grow justify="end">
        <Common.Task.ChannelName
          channel={channel}
          id={Common.Task.getChannelNameID(itemKey)}
        />
        <Common.Task.EnableDisableButton path={`${path}.enabled`} />
      </Flex.Box>
    </Select.ListItem>
  );
};

const getOpenChannel = (channels: InputChannel[]): InputChannel => {
  if (channels.length === 0) return { ...ZERO_INPUT_CHANNEL, key: id.create() };
  const channelToCopy = channels[channels.length - 1];
  return {
    ...channelToCopy,
    key: id.create(),
  };
};

const listItem = Component.renderProp(ChannelListItem);

const Form: FC<
  Common.Task.FormProps<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ>
> = () => (
  <Common.Task.Layouts.List<InputChannel>
    createChannel={getOpenChannel}
    listItem={listItem}
  />
);

const getInitialValues: Common.Task.GetInitialValues<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = ({ deviceKey }) => ({
  ...ZERO_READ_PAYLOAD,
  config: {
    ...ZERO_READ_PAYLOAD.config,
    device: deviceKey ?? ZERO_READ_PAYLOAD.config.device,
  },
});

const onConfigure: Common.Task.OnConfigure<typeof readConfigZ> = async (
  client,
  config,
) => {
  const dev = await client.hardware.devices.retrieve<
    Device.Properties,
    Device.Make,
    Device.Model
  >({
    key: config.device,
  });

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
        name: channelName(dev.name, c.scpiCommand, c.format),
        dataType: c.dataType,
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
  schemas: READ_SCHEMAS,
  type: READ_TYPE,
  getInitialValues,
  onConfigure,
});