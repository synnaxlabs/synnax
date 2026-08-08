// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { labjack, type task } from "@synnaxlabs/client";
import { z } from "zod";

import * as Device from "@/feature/labjack/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "labjack";

export type Scale = labjack.Scale;
export type ScaleType = labjack.ScaleType;
export const SCALE_SCHEMAS = labjack.SCALE_SCHEMAS;

const NO_SCALE: Scale = { type: "none" };

export type TemperatureUnits = labjack.TemperatureUnits;
export type ThermocoupleType = labjack.ThermocoupleType;

export const DEVICE_CJC_SOURCE = "TEMPERATURE_DEVICE_K";

export const AIR_CJC_SOURCE = "TEMPERATURE_AIR_K";

export type InputChannel = labjack.InputChannel;
export type InputChannelType = labjack.InputChannelType;
export const INPUT_CHANNEL_SCHEMAS = labjack.INPUT_CHANNEL_SCHEMAS;

// The scale union has no schema default, so AI and TC seed it explicitly.
const INPUT_CHANNEL_SEEDS: Record<InputChannelType, object> = {
  AI: { type: "AI", scale: NO_SCALE },
  DI: { type: "DI" },
  TC: { type: "TC", scale: NO_SCALE },
};

export const createInputChannel = <T extends InputChannelType>(
  type: T,
): Extract<InputChannel, { type: T }> =>
  INPUT_CHANNEL_SCHEMAS[type].parse(INPUT_CHANNEL_SEEDS[type]);

export const outputChannelZ = labjack.outputChannelZ;
export type OutputChannel = labjack.OutputChannel;
export type OutputChannelType = labjack.OutputChannelType;
export const OUTPUT_CHANNEL_SCHEMAS = labjack.OUTPUT_CHANNEL_SCHEMAS;

export const createOutputChannel = <T extends OutputChannelType>(
  type: T,
): Extract<OutputChannel, { type: T }> =>
  OUTPUT_CHANNEL_SCHEMAS[type].parse({ type });

export type Channel = InputChannel | OutputChannel;

export type ChannelType = Channel["type"];

const deployPortZ = z.string().min(1, "Port must be specified");

const deployAIPortZ = deployPortZ.regex(
  Device.AIN_PORT_REGEX,
  "Invalid port, ports must start with AIN and end with an integer",
);

const deployDigitalPortZ = deployPortZ.regex(
  Device.DIO_PORT_REGEX,
  "Invalid port, port must start with DIO and end with an integer",
);

const deployAOPortZ = deployPortZ.regex(
  Device.DAC_PORT_REGEX,
  "Invalid port, ports must start with DAC and end with an integer",
);

const validateUniquePorts: z.core.CheckFn<Channel[]> = ({
  value: channels,
  issues,
}) => {
  const portToIndexMap = new Map<string, number>();
  channels.forEach(({ port }, i) => {
    if (!portToIndexMap.has(port)) {
      portToIndexMap.set(port, i);
      return;
    }
    const index = portToIndexMap.get(port) as number;
    const code = "custom";
    const message = `Port ${port} has already been used on another channel`;
    issues.push({ code, message, path: [index, "port"], input: channels });
    issues.push({ code, message, path: [i, "port"], input: channels });
  });
};

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadConfig extends labjack.ReadConfig {}

export const readConfigZ = labjack.readConfigZ;

const deployInputChannelZ = z.union([
  labjack.inputChannelAIZ.extend({
    range: z.number().positive().default(10),
    port: deployAIPortZ,
  }),
  labjack.inputChannelDIZ.extend({ port: deployDigitalPortZ }),
  labjack.inputChannelTcZ.extend({
    port: deployAIPortZ,
    cjcSource: z.string().min(1, "CJC Source must be specified"),
  }),
]);

export const deployReadConfigZ = labjack.readConfigZ
  .extend({
    device: Task.deviceKeyZ,
    channels: z
      .array(deployInputChannelZ)
      .check(Task.validateReadChannels)
      .check(validateUniquePorts),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .check(Task.validateStreamRate);

const readStatusDataZ = z
  .object({ errors: z.array(z.object({ message: z.string(), path: z.string() })) })
  .nullish()
  .optional();

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: readStatusDataZ,
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export interface ReadPayload extends task.Payload<ReadSchemas> {}

export const WRITE_TYPE = `${PREFIX}_write`;

export interface WriteConfig extends labjack.WriteConfig {}

export const writeConfigZ = labjack.writeConfigZ;

const deployOutputChannelZ = z.union([
  labjack.outputChannelAOZ.extend({ port: deployAOPortZ }),
  labjack.outputChannelDOZ.extend({ port: deployDigitalPortZ }),
]);

export const deployWriteConfigZ = labjack.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  channels: z
    .array(deployOutputChannelZ)
    .check(Task.validateWriteChannels)
    .check(validateUniquePorts),
  stateRate: z.number().positive().max(50000),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}
