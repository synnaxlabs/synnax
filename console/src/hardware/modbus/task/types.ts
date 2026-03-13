// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { DataType, id, record } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { Common } from "@/hardware/common";

export const PREFIX = "modbus";

const baseInputZ = Common.Task.readChannelZ.extend({ address: z.number() });

const coilInputZ = baseInputZ.extend({ type: z.literal("coil_input") });

const discreteInputZ = baseInputZ.extend({ type: z.literal("discrete_input") });

const typedInputZ = baseInputZ.extend({ dataType: z.string() });

export interface TypedInput extends z.infer<typeof typedInputZ> {}

const holdingRegisterInputZ = typedInputZ.extend({
  type: z.literal("holding_register_input"),
});

const registerInputZ = typedInputZ.extend({ type: z.literal("register_input") });

const variableDensityInputChannelZ = z.union([holdingRegisterInputZ, registerInputZ]);

type VariableDensityInputChannel = z.infer<typeof variableDensityInputChannelZ>;

const fixedDensityInputChannelZ = z.union([coilInputZ, discreteInputZ]);

const inputChannelZ = z.union([
  fixedDensityInputChannelZ,
  variableDensityInputChannelZ,
]);

export type InputChannel = z.infer<typeof inputChannelZ>;

export type InputChannelType = InputChannel["type"];

export const ZERO_INPUT_CHANNELS = {
  coil_input: {
    type: "coil_input",
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    name: "",
  },
  discrete_input: {
    type: "discrete_input",
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    name: "",
  },
  holding_register_input: {
    type: "holding_register_input",
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    dataType: DataType.UINT8.toString(),
    name: "",
  },
  register_input: {
    type: "register_input",
    dataType: DataType.UINT8.toString(),
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    name: "",
  },
} as const satisfies Record<InputChannelType, InputChannel>;

export const INPUT_CHANNEL_SCHEMAS: Record<
  InputChannelType,
  z.ZodType<InputChannel>
> = {
  coil_input: coilInputZ,
  discrete_input: discreteInputZ,
  holding_register_input: holdingRegisterInputZ,
  register_input: registerInputZ,
};

const VARIABLE_DENSITY_INPUT_CHANNEL_TYPES = new Set([
  "holding_register_input",
  "register_input",
]);

const isVariableDensityInputChannelType = (
  type: InputChannelType,
): type is VariableDensityInputChannel["type"] =>
  VARIABLE_DENSITY_INPUT_CHANNEL_TYPES.has(type);

export const isVariableDensityInputChannel = (
  channel: InputChannel,
): channel is VariableDensityInputChannel =>
  isVariableDensityInputChannelType(channel.type);

const baseOutputZ = Common.Task.channelZ.extend({
  address: z.number(),
  channel: z.number(),
  name: Common.Task.nameZ,
});

const coilOutputZ = baseOutputZ.extend({ type: z.literal("coil_output") });

const holdingRegisterOutputZ = baseOutputZ.extend({
  type: z.literal("holding_register_output"),
  dataType: z.string(),
});

const outputChannelZ = z.union([coilOutputZ, holdingRegisterOutputZ]);

export type OutputChannel = z.infer<typeof outputChannelZ>;

export type OutputChannelType = OutputChannel["type"];

export const ZERO_OUTPUT_CHANNELS = {
  coil_output: {
    type: "coil_output",
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    name: "",
  },
  holding_register_output: {
    type: "holding_register_output",
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    dataType: DataType.UINT8.toString(),
    name: "",
  },
} as const satisfies Record<OutputChannelType, OutputChannel>;

export const OUTPUT_CHANNEL_SCHEMAS: Record<
  OutputChannelType,
  z.ZodType<OutputChannel>
> = {
  coil_output: coilOutputZ,
  holding_register_output: holdingRegisterOutputZ,
};

export const READ_TYPE = `${PREFIX}_read`;

const readConfigZ = Common.Task.baseReadConfigZ
  .extend({
    channels: z.array(inputChannelZ),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .check(Common.Task.validateStreamRate);

interface ReadConfig extends z.infer<typeof readConfigZ> {}

const ZERO_READ_CONFIG = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  channels: [],
  sampleRate: 10,
  streamRate: 5,
} as const satisfies ReadConfig;

const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .or(z.null());

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: readStatusDataZ,
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

interface ReadPayload extends task.Payload<ReadSchemas> {}

export const ZERO_READ_PAYLOAD = {
  key: "",
  name: "Modbus Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
} as const satisfies ReadPayload;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = Common.Task.baseConfigZ.extend({
  channels: z.array(outputChannelZ),
});

interface WriteConfig extends z.infer<typeof writeConfigZ> {}

const ZERO_WRITE_CONFIG = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
} as const satisfies WriteConfig;

const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .or(z.null());

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: writeStatusDataZ,
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD = {
  key: "",
  name: "Modbus Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
} as const satisfies WritePayload;

export const SCAN_TYPE = `${PREFIX}_scan`;

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.object({}).or(z.null()),
} as const satisfies task.Schemas;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";
