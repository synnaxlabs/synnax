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

const baseInputZ = Common.Task.readChannelZ.extend({
  address: z.number(),
});

const coilInputZ = baseInputZ.extend({
  type: z.literal(COIL_INPUT_TYPE),
});

export type CoilInput = z.infer<typeof coilInputZ>;

const discreteInputZ = baseInputZ.extend({
  type: z.literal(DISCRETE_INPUT_TYPE),
});

export type DiscreteInput = z.infer<typeof discreteInputZ>;

export const typedInputZ = baseInputZ.extend({
  dataType: z.string(),
});

export type TypedInput = z.infer<typeof typedInputZ>;

const holdingRegisterInputZ = typedInputZ.extend({
  type: z.literal(HOLDING_REGISTER_INPUT_TYPE),
});

export type HoldingRegisterInput = z.infer<typeof holdingRegisterInputZ>;

const registerInputZ = typedInputZ.extend({
  type: z.literal(REGISTER_INPUT_TYPE),
});

export type RegisterInput = z.infer<typeof registerInputZ>;

export const variableDensityInputChannelZ = z.union([
  holdingRegisterInputZ,
  registerInputZ,
]);

export type VariableDensityInputChannel = z.infer<typeof variableDensityInputChannelZ>;

export const fixedDensityInputChannelZ = z.union([coilInputZ, discreteInputZ]);

export type FixedDensityInputChannel = z.infer<typeof fixedDensityInputChannelZ>;

export const inputChannelZ = z.union([
  fixedDensityInputChannelZ,
  variableDensityInputChannelZ,
]);
export type InputChannel = z.infer<typeof inputChannelZ>;
export type InputChannelType = InputChannel["type"];

export const ZERO_INPUT_CHANNELS: Record<InputChannelType, InputChannel> = {
  [COIL_INPUT_TYPE]: {
    type: COIL_INPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
  },
  [DISCRETE_INPUT_TYPE]: {
    type: DISCRETE_INPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
  },
  [HOLDING_REGISTER_INPUT_TYPE]: {
    type: HOLDING_REGISTER_INPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    dataType: DataType.UINT8.toString(),
  },
  [REGISTER_INPUT_TYPE]: {
    type: REGISTER_INPUT_TYPE,
    dataType: DataType.UINT8.toString(),
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
  },
};

export const INPUT_CHANNEL_SCHEMAS: Record<
  InputChannelType,
  z.ZodType<InputChannel>
> = {
  [COIL_INPUT_TYPE]: coilInputZ,
  [DISCRETE_INPUT_TYPE]: discreteInputZ,
  [HOLDING_REGISTER_INPUT_TYPE]: holdingRegisterInputZ,
  [REGISTER_INPUT_TYPE]: registerInputZ,
};

export const VARIABLE_DENSITY_INPUT_CHANNEL_TYPES = new Set([
  HOLDING_REGISTER_INPUT_TYPE,
  REGISTER_INPUT_TYPE,
]);

export const FIXED_DENSITY_INPUT_CHANNEL_TYPES = new Set([
  COIL_INPUT_TYPE,
  DISCRETE_INPUT_TYPE,
]);

export const isVariableDensityInputChannelType = (
  type: InputChannelType,
): type is VariableDensityInputChannel["type"] =>
  VARIABLE_DENSITY_INPUT_CHANNEL_TYPES.has(type);

export const isVariableDensityInputChannel = (
  channel: InputChannel,
): channel is VariableDensityInputChannel =>
  isVariableDensityInputChannelType(channel.type);

export const isFixedDensityInputChannelType = (
  type: InputChannelType,
): type is FixedDensityInputChannel["type"] =>
  FIXED_DENSITY_INPUT_CHANNEL_TYPES.has(type);

export const readConfigZ = Common.Task.baseConfigZ
  .extend({
    channels: z.array(inputChannelZ),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .check(Common.Task.validateStreamRate);

export type ReadConfig = z.infer<typeof readConfigZ>;
export const ZERO_READ_CONFIG: ReadConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
  sampleRate: 10,
  streamRate: 5,
};

export const readStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z
      .array(
        z.object({
          message: z.string(),
          path: z.string(),
        }),
      )
      .optional(),
  })
  .or(z.null());
export type ReadStatus = task.Status<typeof readStatusDataZ>;

export const READ_TYPE = `${PREFIX}_read`;
export const readTypeZ = z.literal(READ_TYPE);
export type ReadType = typeof READ_TYPE;

export interface ReadPayload
  extends task.Payload<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ> {}
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "Modbus Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export interface ReadTask
  extends task.Task<typeof readTypeZ, typeof readConfigZ, typeof readStatusDataZ> {}
export interface NewReadTask extends task.New<typeof readTypeZ, typeof readConfigZ> {}

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
export type TestConnectionCommandType = typeof TEST_CONNECTION_COMMAND_TYPE;

export type TestConnectionStatus = task.Status;

export const SCAN_TYPE = `${PREFIX}_scan`;

const baseOutputZ = Common.Task.channelZ.extend({
  address: z.number(),
  channel: z.number(),
});

const coilOutputZ = baseOutputZ.extend({
  type: z.literal(COIL_OUTPUT_TYPE),
});

export type CoilOutput = z.infer<typeof coilOutputZ>;

const holdingRegisterOutputZ = baseOutputZ.extend({
  type: z.literal(HOLDING_REGISTER_OUTPUT_TYPE),
  dataType: z.string(),
});

export type HoldingRegisterOutput = z.infer<typeof holdingRegisterOutputZ>;

export const outputChannelZ = z.union([coilOutputZ, holdingRegisterOutputZ]);
export type OutputChannel = z.infer<typeof outputChannelZ>;
export type OutputChannelType = OutputChannel["type"];

export const ZERO_OUTPUT_CHANNELS: Record<OutputChannelType, OutputChannel> = {
  [COIL_OUTPUT_TYPE]: {
    type: COIL_OUTPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
  },
  [HOLDING_REGISTER_OUTPUT_TYPE]: {
    type: HOLDING_REGISTER_OUTPUT_TYPE,
    address: 0,
    channel: 0,
    enabled: true,
    key: id.create(),
    dataType: DataType.UINT8.toString(),
  },
};

export const writeConfigZ = Common.Task.baseConfigZ.extend({
  channels: z.array(outputChannelZ),
});

export type WriteConfig = z.infer<typeof writeConfigZ>;

export const ZERO_WRITE_CONFIG: WriteConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
};

export const writeStatusDataZ = z
  .object({
    running: z.boolean(),
    message: z.string(),
    errors: z
      .array(
        z.object({
          message: z.string(),
          path: z.string(),
        }),
      )
      .optional(),
  })
  .or(z.null());
export type WriteStatus = task.Status<typeof writeStatusDataZ>;

export const WRITE_TYPE = `${PREFIX}_write`;
export const writeTypeZ = z.literal(WRITE_TYPE);
export type WriteType = typeof WRITE_TYPE;

export interface WritePayload
  extends task.Payload<
    typeof writeTypeZ,
    typeof writeConfigZ,
    typeof writeStatusDataZ
  > {}

export const ZERO_WRITE_PAYLOAD: WritePayload = {
  key: "",
  name: "Modbus Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
};

export interface WriteTask
  extends task.Task<typeof writeTypeZ, typeof writeConfigZ, typeof writeStatusDataZ> {}
export interface NewWriteTask
  extends task.New<typeof writeTypeZ, typeof writeConfigZ> {}

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

export const scanConfigZ = z.object({});
export type ScanConfig = z.infer<typeof scanConfigZ>;
export const ZERO_SCAN_CONFIG: ScanConfig = {};

export const scanStatusDataZ = z.object({}).or(z.null());
export type ScanStatus = task.Status<typeof scanStatusDataZ>;

export const scanTypeZ = z.literal(SCAN_TYPE);
export type ScanType = typeof SCAN_TYPE;

export const SCAN_SCHEMAS: task.Schemas<
  typeof scanTypeZ,
  typeof scanConfigZ,
  typeof scanStatusDataZ
> = {
  typeSchema: scanTypeZ,
  configSchema: scanConfigZ,
  statusDataSchema: scanStatusDataZ,
};

export interface ScanPayload
  extends task.Payload<typeof scanTypeZ, typeof scanConfigZ, typeof scanStatusDataZ> {}
export interface ScanTask
  extends task.Task<typeof scanTypeZ, typeof scanConfigZ, typeof scanStatusDataZ> {}
export interface NewScanTask extends task.New<typeof scanTypeZ, typeof scanConfigZ> {}
