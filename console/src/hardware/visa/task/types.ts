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

export const PREFIX = "visa";

// Response format types matching C++ enum
export const RESPONSE_FORMAT_FLOAT = "float";
export const RESPONSE_FORMAT_INTEGER = "integer";
export const RESPONSE_FORMAT_STRING = "string";
export const RESPONSE_FORMAT_FLOAT_ARRAY = "float_array";
export const RESPONSE_FORMAT_BINARY_BLOCK = "binary_block";
export const RESPONSE_FORMAT_BOOLEAN = "boolean";

export const responseFormatZ = z.enum([
  RESPONSE_FORMAT_FLOAT,
  RESPONSE_FORMAT_INTEGER,
  RESPONSE_FORMAT_STRING,
  RESPONSE_FORMAT_FLOAT_ARRAY,
  RESPONSE_FORMAT_BINARY_BLOCK,
  RESPONSE_FORMAT_BOOLEAN,
]);
export type ResponseFormat = z.infer<typeof responseFormatZ>;

// Input channel configuration
export const inputChannelZ = Common.Task.readChannelZ.extend({
  scpiCommand: z.string().min(1, "SCPI command is required"),
  format: responseFormatZ,
  dataType: z.string().default(DataType.FLOAT64.toString()),
  delimiter: z.string().default(","),
  arrayLength: z.number().int().nonnegative().default(0),
});
export interface InputChannel extends z.infer<typeof inputChannelZ> {}

export const ZERO_INPUT_CHANNEL = {
  key: id.create(),
  enabled: true,
  channel: 0,
  scpiCommand: "",
  format: RESPONSE_FORMAT_FLOAT,
  dataType: DataType.FLOAT64.toString(),
  delimiter: ",",
  arrayLength: 0,
} as const satisfies InputChannel;

// Output channel configuration
export const outputChannelZ = Common.Task.channelZ.extend({
  channel: z.number(),
  scpiCommand: z.string().min(1, "SCPI command is required"),
  commandTemplate: z
    .string()
    .min(1, "Command template is required")
    .refine((val) => val.includes("{value}"), {
      message: "Command template must contain {value} placeholder",
    }),
});
export interface OutputChannel extends z.infer<typeof outputChannelZ> {}

export const ZERO_OUTPUT_CHANNEL = {
  key: id.create(),
  enabled: true,
  channel: 0,
  scpiCommand: "",
  commandTemplate: "",
} as const satisfies OutputChannel;

// Read task configuration
export const readConfigZ = Common.Task.baseReadConfigZ
  .extend({
    channels: z.array(inputChannelZ),
    sampleRate: z.number().positive().max(10000),
    streamRate: z.number().positive().max(10000),
  })
  .check(Common.Task.validateStreamRate)
  .check(Common.Task.validateReadChannels);

interface ReadConfig extends z.infer<typeof readConfigZ> {}

const ZERO_READ_CONFIG = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  channels: [],
  sampleRate: 1,
  streamRate: 1,
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
  name: "VISA Read Task",
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

// Write task configuration
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
  name: "VISA Write Task",
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

// Scan task configuration
export const SCAN_TYPE = `${PREFIX}_scan`;
export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

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

// Helper functions
export const readMapKey = (ch: InputChannel): string =>
  `${ch.scpiCommand}_${ch.format}`;

export const writeMapKey = (ch: OutputChannel): string =>
  `${ch.scpiCommand}_${ch.commandTemplate}`;

export const channelName = (
  deviceName: string,
  scpiCommand: string,
  format: ResponseFormat,
): string => {
  const sanitized = scpiCommand.replace(/[^a-zA-Z0-9]/g, "_");
  return `${deviceName}_${sanitized}_${format}`;
};