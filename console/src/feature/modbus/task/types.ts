// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { modbus, type task } from "@synnaxlabs/client";
import { DataType, record } from "@synnaxlabs/x";
import { z } from "zod";

import { Task } from "@/platform/task";

export const PREFIX = "modbus";

export type ReadChannel = modbus.ReadChannel;
export type ReadChannelType = modbus.ReadChannelType;
export type TypedReadChannel =
  modbus.HoldingRegisterReadChannel | modbus.InputRegisterReadChannel;

export const READ_CHANNEL_SCHEMAS = modbus.READ_CHANNEL_SCHEMAS;

const VARIABLE_DENSITY_READ_CHANNEL_TYPES = new Set<ReadChannelType>([
  "holding_register",
  "input_register",
]);

export const isVariableDensityReadChannel = (
  channel: ReadChannel,
): channel is TypedReadChannel => VARIABLE_DENSITY_READ_CHANNEL_TYPES.has(channel.type);

export const isStringChannel = (channel: ReadChannel): channel is TypedReadChannel =>
  isVariableDensityReadChannel(channel) &&
  new DataType(channel.dataType).equals(DataType.STRING);

// Auto-generated channel names and device map keys keep the released type
// spellings, so channels created before the labels were renamed keep matching.
const NAME_TYPES: Record<ReadChannelType, string> = {
  coil: "coil_input",
  discrete_input: "discrete_input",
  holding_register: "holding_register_input",
  input_register: "register_input",
  holding_register_bit: "holding_register_bit_input",
  input_register_bit: "register_bit_input",
};

export const readMapKey = (channel: ReadChannel) => {
  let s = `${NAME_TYPES[channel.type]}-${channel.address.toString()}`;
  if (isVariableDensityReadChannel(channel)) s += `-${channel.dataType}`;
  if ("bit" in channel) s += `-${channel.bit}`;
  return s.replaceAll("_", "-");
};

export const readChannelName = (deviceName: string, channel: ReadChannel) => {
  let s = `${deviceName}_${NAME_TYPES[channel.type]}_${channel.address}`;
  if (isVariableDensityReadChannel(channel))
    s += `_${new DataType(channel.dataType).toString(true)}`;
  if ("bit" in channel) s += `_${channel.bit}`;
  return s;
};

export type WriteChannel = modbus.WriteChannel;
export type WriteChannelType = modbus.WriteChannelType;

export const BIT_BOUNDS = { lower: 0, upper: 15 };
export const STRING_LENGTH_BOUNDS = { lower: 1, upper: 250 };

/**
 * Returns the first address after a channel. A register value spans one address per 16
 * bits of its data type, a string spans one address per two characters, and every
 * other channel spans one.
 */
const nextAddress = (channel: ReadChannel | WriteChannel): number => {
  if (!("dataType" in channel)) return channel.address + 1;
  if ("stringLength" in channel && isStringChannel(channel))
    return channel.address + Math.ceil(channel.stringLength / 2);
  const bytes = new DataType(channel.dataType).density.valueOf();
  return channel.address + Math.ceil(bytes / 2);
};

/**
 * Returns the channel to add after the given one: the next bit of a bit channel, then
 * bit 0 of the next register, or the first address after any other channel.
 */
export const nextChannel = <C extends ReadChannel | WriteChannel>(channel: C): C => {
  if (!("bit" in channel)) return { ...channel, address: nextAddress(channel) };
  if (channel.bit < BIT_BOUNDS.upper) return { ...channel, bit: channel.bit + 1 };
  return { ...channel, address: channel.address + 1, bit: 0 };
};

export const WRITE_CHANNEL_SCHEMAS = modbus.WRITE_CHANNEL_SCHEMAS;

const REGISTER_DATA_TYPES = [
  DataType.INT8,
  DataType.INT16,
  DataType.INT32,
  DataType.INT64,
  DataType.UINT8,
  DataType.UINT16,
  DataType.UINT32,
  DataType.UINT64,
  DataType.FLOAT32,
  DataType.FLOAT64,
];

/** Data types the Driver cannot write to registers. */
export const UNSUPPORTED_WRITE_DATA_TYPES = DataType.ALL.filter(
  (d) => !d.matches(...REGISTER_DATA_TYPES),
);

/** Data types the Driver cannot read from registers. */
export const UNSUPPORTED_READ_DATA_TYPES = DataType.ALL.filter(
  (d) => !d.matches(...REGISTER_DATA_TYPES, DataType.STRING),
);

const ADDRESS_LIMIT = 65536;

const addressRangeIssue = (
  channels: (ReadChannel | WriteChannel)[],
  ch: ReadChannel | WriteChannel,
  i: number,
): z.core.$ZodRawIssue | null => {
  if (nextAddress(ch) <= ADDRESS_LIMIT) return null;
  return {
    code: "custom",
    message: `Channel ends past address ${ADDRESS_LIMIT - 1}`,
    path: [i, "address"],
    input: channels,
  };
};

const validateReadChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<ReadChannel[]>) => {
  const bindings = new Set<string>();
  channels.forEach((ch, i) => {
    if (ch.disabled) return;
    const rangeIssue = addressRangeIssue(channels, ch, i);
    if (rangeIssue != null) issues.push(rangeIssue);
    if (
      isStringChannel(ch) &&
      (ch.stringLength < STRING_LENGTH_BOUNDS.lower ||
        ch.stringLength > STRING_LENGTH_BOUNDS.upper)
    )
      issues.push({
        code: "custom",
        message: `String length must be from ${STRING_LENGTH_BOUNDS.lower} to ${STRING_LENGTH_BOUNDS.upper}`,
        path: [i, "stringLength"],
        input: channels,
      });
    const binding = readMapKey(ch);
    if (!bindings.has(binding)) {
      bindings.add(binding);
      return;
    }
    issues.push({
      code: "custom",
      message: "Another channel reads the same address and data type",
      path: [i, "address"],
      input: channels,
    });
  });
};

const isCoil = (ch: WriteChannel) => ch.type === "coil";

// Bits of one register are written together, so only the same bit overlaps.
const writesOverlap = (a: WriteChannel, b: WriteChannel): boolean => {
  if (isCoil(a) !== isCoil(b)) return false;
  if (a.type === "holding_register_bit" && b.type === "holding_register_bit")
    return a.address === b.address && a.bit === b.bit;
  return a.address < nextAddress(b) && b.address < nextAddress(a);
};

const validateWriteChannels = ({
  value: channels,
  issues,
}: z.core.ParsePayload<WriteChannel[]>) => {
  const enabled: WriteChannel[] = [];
  channels.forEach((ch, i) => {
    if (ch.disabled) return;
    const rangeIssue = addressRangeIssue(channels, ch, i);
    if (rangeIssue != null) issues.push(rangeIssue);
    const overlap = enabled.find((other) => writesOverlap(other, ch));
    enabled.push(ch);
    if (overlap == null) return;
    issues.push({
      code: "custom",
      message: `Overlaps the ${overlap.type.replaceAll("_", " ")} at address ${overlap.address}`,
      path: [i, "address"],
      input: channels,
    });
  });
};

export const READ_TYPE = `${PREFIX}_read`;

const readConfigZ = modbus.readConfigZ;

export const deployReadConfigZ = modbus.readConfigZ
  .extend({
    device: Task.deviceKeyZ,
    channels: modbus.readConfigZ.shape.channels.check(validateReadChannels),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .check(Task.validateStreamRate);

const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish()
  .optional();

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: readStatusDataZ,
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = modbus.writeConfigZ;

export const deployWriteConfigZ = modbus.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  channels: modbus.writeConfigZ.shape.channels.check(validateWriteChannels),
});

const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .nullish()
  .optional();

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: writeStatusDataZ,
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export const SCAN_TYPE = `${PREFIX}_scan`;

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.object({}).nullish().optional(),
} as const satisfies task.Schemas;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";
