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

export type WriteChannel = modbus.WriteChannel;
export type WriteChannelType = modbus.WriteChannelType;

/**
 * Returns the first address after a channel. A register channel spans one address per
 * 16 bits of its data type, and a coil or discrete input spans one.
 */
export const nextAddress = (channel: ReadChannel | WriteChannel): number => {
  if (!("dataType" in channel)) return channel.address + 1;
  const bytes = new DataType(channel.dataType).density.valueOf();
  return channel.address + Math.ceil(bytes / 2);
};

/** @returns the first and last addresses of a channel. */
export const addressSpan = (channel: ReadChannel): [number, number] => [
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

/** @returns where channels of the same type read the same addresses. */
export const findOverlaps = (channels: ReadChannel[]): Overlaps => {
  const keys = new Set<string>();
  const ranges: Overlaps["ranges"] = {};
  const byType = Map.groupBy(channels, (c) => c.type);
  for (const [type, typed] of byType) {
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
    if (shared.length > 0) ranges[type] = shared;
  }
  return { keys, ranges };
};

export const WRITE_CHANNEL_SCHEMAS = modbus.WRITE_CHANNEL_SCHEMAS;

export const READ_TYPE = `${PREFIX}_read`;

const readConfigZ = modbus.readConfigZ;

export const deployReadConfigZ = modbus.readConfigZ
  .extend({
    device: Task.deviceKeyZ,
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
