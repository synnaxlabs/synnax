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

export type InputChannel = modbus.InputChannel;
export type InputChannelType = modbus.InputChannelType;
export type TypedInput =
  | modbus.InputChannelHoldingRegisterInput
  | modbus.InputChannelRegisterInput;

export const INPUT_CHANNEL_SCHEMAS = modbus.INPUT_CHANNEL_SCHEMAS;

export const ZERO_INPUT_CHANNELS = Object.fromEntries(
  modbus.INPUT_CHANNEL_TYPES.map((t) => [
    t,
    modbus.INPUT_CHANNEL_SCHEMAS[t].parse({ type: t }),
  ]),
) as Record<InputChannelType, InputChannel>;

const VARIABLE_DENSITY_INPUT_CHANNEL_TYPES = new Set<InputChannelType>([
  "holding_register_input",
  "register_input",
]);

export const isVariableDensityInputChannel = (
  channel: InputChannel,
): channel is TypedInput => VARIABLE_DENSITY_INPUT_CHANNEL_TYPES.has(channel.type);

export type OutputChannel = modbus.OutputChannel;
export type OutputChannelType = modbus.OutputChannelType;

export const OUTPUT_CHANNEL_SCHEMAS = modbus.OUTPUT_CHANNEL_SCHEMAS;

export const ZERO_OUTPUT_CHANNELS = Object.fromEntries(
  modbus.OUTPUT_CHANNEL_TYPES.map((t) => [
    t,
    modbus.OUTPUT_CHANNEL_SCHEMAS[t].parse({ type: t }),
  ]),
) as Record<OutputChannelType, OutputChannel>;

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

const ZERO_READ_CONFIG = modbus.readConfigZ.parse({});

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

interface ReadPayload extends task.Payload<ReadSchemas> {}

export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  rack: 0,
  name: "Modbus Read Task",
  config: ZERO_READ_CONFIG,
  configHash: "",
  type: READ_TYPE,
  internal: false,
  snapshot: false,
};

export const WRITE_TYPE = `${PREFIX}_write`;

export interface WriteConfig extends modbus.WriteConfig {}

export const writeConfigZ = modbus.writeConfigZ;

export const deployWriteConfigZ = modbus.writeConfigZ.extend({
  device: Task.deviceKeyZ,
});

const ZERO_WRITE_CONFIG = modbus.writeConfigZ.parse({});

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

interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  rack: 0,
  name: "Modbus Write Task",
  config: ZERO_WRITE_CONFIG,
  configHash: "",
  type: WRITE_TYPE,
  internal: false,
  snapshot: false,
};

export const SCAN_TYPE = `${PREFIX}_scan`;

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.object({}).nullish().optional(),
} as const satisfies task.Schemas;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";
