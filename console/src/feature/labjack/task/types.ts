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

import {
  AIN_PORT_REGEX,
  DAC_PORT_REGEX,
  DIO_PORT_REGEX,
  negativeChannelOf,
} from "@/feature/labjack/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "labjack";

export const SCAN_TYPE = `${PREFIX}_scan`;

const NO_SCALE: task.Scale = { type: "none" };

export type TemperatureUnits = labjack.TemperatureUnits;
export type ThermocoupleType = labjack.ThermocoupleType;

export const DEVICE_CJC_SOURCE = "TEMPERATURE_DEVICE_K";

export const AIR_CJC_SOURCE = "TEMPERATURE_AIR_K";

export type ReadChannel = labjack.ReadChannel;
export type ReadChannelType = labjack.ReadChannelType;
export const READ_CHANNEL_SCHEMAS = labjack.READ_CHANNEL_SCHEMAS;

// The scale union has no schema default, so analog and thermocouple seed it explicitly.
const READ_CHANNEL_SEEDS: Record<ReadChannelType, object> = {
  analog: { type: "analog", scale: NO_SCALE },
  digital: { type: "digital" },
  thermocouple: { type: "thermocouple", scale: NO_SCALE },
};

export const createReadChannel = <T extends ReadChannelType>(
  type: T,
): Extract<ReadChannel, { type: T }> =>
  READ_CHANNEL_SCHEMAS[type].parse(READ_CHANNEL_SEEDS[type]);

export type WriteChannel = labjack.WriteChannel;
export type WriteChannelType = labjack.WriteChannelType;
const WRITE_CHANNEL_SCHEMAS = labjack.WRITE_CHANNEL_SCHEMAS;

export const createWriteChannel = <T extends WriteChannelType>(
  type: T,
): Extract<WriteChannel, { type: T }> => WRITE_CHANNEL_SCHEMAS[type].parse({ type });

export type Channel = ReadChannel | WriteChannel;

const deployPortZ = z.string().min(1, "Port must be specified");

const deployAIPortZ = deployPortZ.regex(
  AIN_PORT_REGEX,
  "Invalid port, ports must start with AIN and end with an integer",
);

const deployDigitalPortZ = deployPortZ.regex(
  DIO_PORT_REGEX,
  "Invalid port, port must start with DIO and end with an integer",
);

const deployAOPortZ = deployPortZ.regex(
  DAC_PORT_REGEX,
  "Invalid port, ports must start with DAC and end with an integer",
);

export const READ_TYPE = `${PREFIX}_read`;

const readConfigZ = labjack.readConfigZ;

const validateNegativeChannel = ({
  value: { port, negChan },
  issues,
}: z.core.ParsePayload<{ port: string; negChan?: number }>) => {
  const pair = negativeChannelOf(port);
  if (negChan == null || negChan === pair) return;
  issues.push({
    code: "custom",
    message:
      pair == null
        ? `${port} can only read against ground`
        : `Negative channel must be ground or AIN${pair}`,
    path: ["negChan"],
    input: negChan,
  });
};

const deployReadChannelZ = z.discriminatedUnion("type", [
  labjack.analogReadChannelZ
    .extend({ range: z.number().positive().default(10), port: deployAIPortZ })
    .check(validateNegativeChannel),
  labjack.digitalReadChannelZ.extend({ port: deployDigitalPortZ }),
  labjack.thermocoupleReadChannelZ
    .extend({
      port: deployAIPortZ,
      cjcSource: z.string().min(1, "CJC source must be specified"),
    })
    .check(validateNegativeChannel),
]);

export const deployReadConfigZ = labjack.readConfigZ
  .extend({
    device: Task.deviceKeyZ,
    channels: z
      .array(deployReadChannelZ)
      .check(Task.validateReadChannels)
      .check(Task.validateEnabledChannel),
    sampleRate: z.number().positive().max(50000),
    streamRate: z.number().positive().max(50000),
  })
  .check(Task.validateStreamRate);

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: Task.fieldErrorsZ.nullish(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const WRITE_TYPE = `${PREFIX}_write`;

const writeConfigZ = labjack.writeConfigZ;

const deployWriteChannelZ = z.union([
  labjack.analogWriteChannelZ.extend({ port: deployAOPortZ }),
  labjack.digitalWriteChannelZ.extend({ port: deployDigitalPortZ }),
]);

export const deployWriteConfigZ = labjack.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  channels: z
    .array(deployWriteChannelZ)
    .check(Task.validateWriteChannels)
    .check(Task.validateEnabledChannel),
  stateRate: z.number().positive().max(50000),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;
