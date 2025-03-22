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
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "modbus";
export const COIL_INPUT_TYPE = "coil_input";
export const DISCRETE_INPUT_TYPE = "discrete_input";
export const HOLDING_REGISTER_INPUT_TYPE = "holding_register_input";
export const REGISTER_INPUT_TYPE = "register_input";

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

export const inputChannelZ = z.union([
  coilInputZ,
  discreteInputZ,
  holdingRegisterInputZ,
  registerInputZ,
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
  z.ZodSchema<InputChannel>
> = {
  [COIL_INPUT_TYPE]: coilInputZ,
  [DISCRETE_INPUT_TYPE]: discreteInputZ,
  [HOLDING_REGISTER_INPUT_TYPE]: holdingRegisterInputZ,
  [REGISTER_INPUT_TYPE]: registerInputZ,
};

export const readConfigZ = Common.Task.baseConfigZ
  .extend({
    channels: z.array(inputChannelZ),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .superRefine(Common.Task.validateStreamRate);

export type ReadConfig = z.infer<typeof readConfigZ>;
export const ZERO_READ_CONFIG: ReadConfig = {
  ...Common.Task.ZERO_BASE_CONFIG,
  channels: [],
  sampleRate: 10,
  streamRate: 5,
};

export interface ReadStateDetails extends Common.Task.StateDetails {
  running: boolean;
  message: string;
  errors?: { message: string; path: string }[];
}
export interface ReadState extends task.State<ReadStateDetails> {}

export const READ_TYPE = `${PREFIX}_read`;
export type ReadType = typeof READ_TYPE;

export interface ReadPayload
  extends task.Payload<ReadConfig, ReadStateDetails, ReadType> {}
export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "Modbus Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export interface ReadTask extends task.Task<ReadConfig, ReadStateDetails, ReadType> {}
export interface NewReadTask extends task.New<ReadConfig, ReadType> {}

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";
export type TestConnectionCommandType = typeof TEST_CONNECTION_COMMAND_TYPE;

export interface TestConnectionCommandState
  extends task.State<{
    message: string;
  }> {}

export interface TestConnectionCommandResponse {
  variant: "success" | "error";
  details?: {
    message: string;
  };
}

export const SCAN_TYPE = `${PREFIX}_scan`;
