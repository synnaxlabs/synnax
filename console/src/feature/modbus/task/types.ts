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

/** @returns the first and last addresses of a channel. */
export const addressSpan = (channel: ReadChannel | WriteChannel): [number, number] => [
  channel.address,
  nextAddress(channel) - 1,
];

/** Where channels of the same type read the same addresses. */
export interface Overlaps {
  /** The keys of the channels that read an address another channel reads. */
  keys: Set<string>;
  /** For each type, the shared address ranges, in order and apart. */
  ranges: Partial<Record<ReadChannelType, [number, number][]>>;
}

const mergeRanges = (spans: [number, number][]): [number, number][] =>
  spans
    .toSorted((a, b) => a[0] - b[0])
    .reduce<[number, number][]>((merged, [lo, hi]) => {
      const last = merged.at(-1);
      if (last != null && lo <= last[1] + 1) last[1] = Math.max(last[1], hi);
      else merged.push([lo, hi]);
      return merged;
    }, []);

/** @returns where channels of the same type read the same addresses. */
export const findOverlaps = (channels: ReadChannel[]): Overlaps => {
  const keys = new Set<string>();
  const ranges: Overlaps["ranges"] = {};
  const sharedByType = new Map<ReadChannelType, [number, number][]>();
  // Bits of one register read apart, so only channels on the same bit can overlap.
  const groups = Map.groupBy(channels, (c) =>
    "bit" in c ? `${c.type}.${c.bit}` : c.type,
  );
  for (const typed of groups.values()) {
    const { type } = typed[0];
    // A channel stops reading one address past its last, and a stop comes before a
    // start at the same address, so channels that only touch do not overlap.
    const events = typed
      .flatMap((c): [number, number, string][] => {
        const [lo, hi] = addressSpan(c);
        return [
          [lo, 1, c.key],
          [hi + 1, -1, c.key],
        ];
      })
      .sort((a, b) => a[0] - b[0] || a[1] - b[1]);
    const reading = new Set<string>();
    const shared: [number, number][] = [];
    let start = 0;
    for (const [address, step, key] of events) {
      if (step < 0) {
        reading.delete(key);
        if (reading.size === 1) shared.push([start, address - 1]);
        continue;
      }
      reading.add(key);
      if (reading.size < 2) continue;
      keys.add(key);
      if (reading.size > 2) continue;
      reading.forEach((k) => keys.add(k));
      const last = shared.at(-1);
      if (last?.[1] === address - 1) {
        shared.pop();
        start = last[0];
      } else start = address;
    }
    if (shared.length > 0)
      sharedByType.set(type, [...(sharedByType.get(type) ?? []), ...shared]);
  }
  for (const [type, spans] of sharedByType) ranges[type] = mergeRanges(spans);
  return { keys, ranges };
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
