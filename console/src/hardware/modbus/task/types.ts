// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { DataType, id } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { Common } from "@/hardware/common";

export const PREFIX = "modbus";
export const COIL_INPUT_TYPE = "coil_input";
export const DISCRETE_INPUT_TYPE = "discrete_input";
export const HOLDING_REGISTER_INPUT_TYPE = "holding_register_input";
export const REGISTER_INPUT_TYPE = "register_input";
export const COIL_OUTPUT_TYPE = "coil_output";
export const HOLDING_REGISTER_OUTPUT_TYPE = "holding_register_output";

const baseInputZ = Common.Task.readChannelZ.extend({ address: z.number() });

const coilInputZ = baseInputZ.extend({ type: z.literal(COIL_INPUT_TYPE) });

const discreteInputZ = baseInputZ.extend({ type: z.literal(DISCRETE_INPUT_TYPE) });

const typedInputZ = baseInputZ.extend({ dataType: z.string() });

export interface TypedInput extends z.infer<typeof typedInputZ> {}

const holdingRegisterInputZ = typedInputZ.extend({
  type: z.literal(HOLDING_REGISTER_INPUT_TYPE),
});

const registerInputZ = typedInputZ.extend({ type: z.literal(REGISTER_INPUT_TYPE) });

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
  [COIL_INPUT_TYPE]: {
    type: COIL_INPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    name: "",
  },
  [DISCRETE_INPUT_TYPE]: {
    type: DISCRETE_INPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    name: "",
  },
  [HOLDING_REGISTER_INPUT_TYPE]: {
    type: HOLDING_REGISTER_INPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    dataType: DataType.UINT8.toString(),
    name: "",
  },
  [REGISTER_INPUT_TYPE]: {
    type: REGISTER_INPUT_TYPE,
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
  [COIL_INPUT_TYPE]: coilInputZ,
  [DISCRETE_INPUT_TYPE]: discreteInputZ,
  [HOLDING_REGISTER_INPUT_TYPE]: holdingRegisterInputZ,
  [REGISTER_INPUT_TYPE]: registerInputZ,
};

const VARIABLE_DENSITY_INPUT_CHANNEL_TYPES = new Set([
  HOLDING_REGISTER_INPUT_TYPE,
  REGISTER_INPUT_TYPE,
]);

const isVariableDensityInputChannelType = (
  type: InputChannelType,
): type is VariableDensityInputChannel["type"] =>
  VARIABLE_DENSITY_INPUT_CHANNEL_TYPES.has(type);

export const isVariableDensityInputChannel = (
  channel: InputChannel,
): channel is VariableDensityInputChannel =>
  isVariableDensityInputChannelType(channel.type);

export const readConfigZ = Common.Task.baseReadConfigZ
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

export const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .or(z.null());

export const READ_TYPE = `${PREFIX}_read`;
export const readTypeZ = z.literal(READ_TYPE);

interface ReadPayload
  extends task.Payload<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ> {}
export const ZERO_READ_PAYLOAD = {
  key: "",
  name: "Modbus Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
} as const satisfies ReadPayload;

export const READ_SCHEMAS: task.Schemas<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = {
  typeSchema: readTypeZ,
  configSchema: readConfigZ,
  statusDataSchema: readStatusDataZ,
};

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const SCAN_TYPE = `${PREFIX}_scan`;

const baseOutputZ = Common.Task.channelZ.extend({
  address: z.number(),
  channel: z.number(),
  name: Common.Task.nameZ,
});

const coilOutputZ = baseOutputZ.extend({ type: z.literal(COIL_OUTPUT_TYPE) });

const holdingRegisterOutputZ = baseOutputZ.extend({
  type: z.literal(HOLDING_REGISTER_OUTPUT_TYPE),
  dataType: z.string(),
});

const outputChannelZ = z.union([coilOutputZ, holdingRegisterOutputZ]);
export type OutputChannel = z.infer<typeof outputChannelZ>;
export type OutputChannelType = OutputChannel["type"];

export const ZERO_OUTPUT_CHANNELS = {
  [COIL_OUTPUT_TYPE]: {
    type: COIL_OUTPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    name: "",
  },
  [HOLDING_REGISTER_OUTPUT_TYPE]: {
    type: HOLDING_REGISTER_OUTPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    dataType: DataType.UINT8.toString(),
    name: "",
  },
} as const satisfies Record<OutputChannelType, OutputChannel>;

export const writeConfigZ = Common.Task.baseConfigZ.extend({
  channels: z.array(outputChannelZ),
});
interface WriteConfig extends z.infer<typeof writeConfigZ> {}

export const ZERO_WRITE_CONFIG = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
} as const satisfies WriteConfig;

export const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z.array(z.object({ message: z.string(), path: z.string() })).optional(),
  })
  .or(z.null());

export const WRITE_TYPE = `${PREFIX}_write`;
export const writeTypeZ = z.literal(WRITE_TYPE);

interface WritePayload
  extends task.Payload<
    typeof writeTypeZ,
    typeof writeConfigZ,
    typeof writeStatusDataZ
  > {}

export const ZERO_WRITE_PAYLOAD = {
  key: "",
  name: "Modbus Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
} as const satisfies WritePayload;

export const WRITE_SCHEMAS: task.Schemas<
  typeof writeTypeZ,
  typeof writeConfigZ,
  typeof writeStatusDataZ
> = {
  typeSchema: writeTypeZ,
  configSchema: writeConfigZ,
  statusDataSchema: writeStatusDataZ,
};

export const OUTPUT_CHANNEL_SCHEMAS: Record<
  OutputChannelType,
  z.ZodType<OutputChannel>
> = {
  [COIL_OUTPUT_TYPE]: coilOutputZ,
  [HOLDING_REGISTER_OUTPUT_TYPE]: holdingRegisterOutputZ,
};

const scanTypeZ = z.literal(SCAN_TYPE);

const scanConfigZ = z.object({});

const scanStatusDataZ = z.object({}).or(z.null());

export const SCAN_SCHEMAS: task.Schemas<
  typeof scanTypeZ,
  typeof scanConfigZ,
  typeof scanStatusDataZ
> = {
  typeSchema: scanTypeZ,
  configSchema: scanConfigZ,
  statusDataSchema: scanStatusDataZ,
};
