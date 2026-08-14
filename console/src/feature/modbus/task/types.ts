// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { modbus, type task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
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
): channel is TypedReadChannel =>
  VARIABLE_DENSITY_READ_CHANNEL_TYPES.has(channel.type);

export type WriteChannel = modbus.WriteChannel;
export type WriteChannelType = modbus.WriteChannelType;

export const WRITE_CHANNEL_SCHEMAS = modbus.WRITE_CHANNEL_SCHEMAS;

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadConfig extends modbus.ReadConfig {}

export const readConfigZ = modbus.readConfigZ;

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

export interface WriteConfig extends modbus.WriteConfig {}

export const writeConfigZ = modbus.writeConfigZ;

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
