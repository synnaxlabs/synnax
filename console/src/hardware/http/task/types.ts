// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { JSON } from "@synnaxlabs/pluto";
import { Rate } from "@synnaxlabs/x";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "http";

const jsonPointerZ = z
  .string()
  .regex(/^(?:$|(?:\/(?:[^~/]|~0|~1)*)+)$/, "JSON pointers");

const timeFormatZ = z.enum(["iso8601", "unix_sec", "unix_ms", "unix_us", "unix_ns"]);

export type TimeFormat = z.infer<typeof timeFormatZ>;

export const SCAN_TYPE = `${PREFIX}_scan`;

export const scanTypeZ = z.literal(SCAN_TYPE);

const responseValidationZ = z.object({
  field: jsonPointerZ,
  expectedValue: JSON.primitiveZ,
});

export interface ResponseValidation extends z.infer<typeof responseValidationZ> {}

export const ZERO_RESPONSE_VALIDATION = {
  field: "",
  expectedValue: null,
} as const satisfies ResponseValidation;

export const scanConfigZ = Common.Task.baseConfigZ.extend({
  rate: z.number().positive("Rate must be positive"),
  path: z.string().min(1, "Path is required"),
  response: responseValidationZ.optional(),
});

export interface ScanConfig extends z.infer<typeof scanConfigZ> {}

export const ZERO_SCAN_CONFIG = {
  ...Common.Task.ZERO_BASE_CONFIG,
  rate: Rate.hz(1).valueOf(),
  path: "",
} as const satisfies ScanConfig;

interface ScanPayload extends task.Payload<typeof scanTypeZ, typeof scanConfigZ> {}

export const ZERO_SCAN_PAYLOAD: ScanPayload = {
  key: "",
  name: "HTTP Scan Task",
  config: ZERO_SCAN_CONFIG,
  type: SCAN_TYPE,
};

export const SCAN_SCHEMAS: task.Schemas<typeof scanTypeZ, typeof scanConfigZ> = {
  typeSchema: scanTypeZ,
  configSchema: scanConfigZ,
  statusDataSchema: z.unknown(),
};

// --- Read Task ---

export const READ_TYPE = `${PREFIX}_read`;

export const readTypeZ = z.literal(READ_TYPE);

const readFieldZ = Common.Task.readChannelZ.extend({
  pointer: jsonPointerZ,
  timestampFormat: timeFormatZ.optional(), // only used for channels where data type is timestamp.
});
export interface ReadField extends z.infer<typeof readFieldZ> {}

export const ZERO_READ_FIELD: ReadField = {
  ...Common.Task.ZERO_READ_CHANNEL,
  pointer: "",
};

const baseReadEndpointZ = z.object({
  key: z.string(),
  path: z.string().min(1, "Path is required"),
  queryParams: z.record(z.string(), z.string()).optional(),
  fields: z.array(readFieldZ),
});
const getReadEndpointZ = baseReadEndpointZ.extend({ method: z.literal("GET") });
const postReadEndpointZ = baseReadEndpointZ.extend({
  method: z.literal("POST"),
  body: z.string().optional(),
});

const readEndpointZ = z.discriminatedUnion("method", [
  getReadEndpointZ,
  postReadEndpointZ,
]);
export type ReadEndpoint = z.infer<typeof readEndpointZ>;

export const ZERO_READ_ENDPOINT: ReadEndpoint = {
  key: "",
  method: "GET",
  path: "",
  fields: [],
};

export const readConfigZ = Common.Task.baseReadConfigZ.extend({
  rate: z
    .number()
    .positive("Rate must be positive")
    .max(100, "Rate must be less than 100 Hz"),
  strict: z.boolean().default(false),
  endpoints: z.array(readEndpointZ),
});
export interface ReadConfig extends z.infer<typeof readConfigZ> {}

export const ZERO_READ_CONFIG: ReadConfig = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  rate: Rate.hz(1).valueOf(),
  strict: false,
  endpoints: [],
};

export const readStatusDataZ = z
  .object({ running: z.boolean(), message: z.string() })
  .or(z.null());

export interface ReadPayload extends task.Payload<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> {}

export const ZERO_READ_PAYLOAD: ReadPayload = {
  key: "",
  name: "HTTP Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
};

export const READ_SCHEMAS: task.Schemas<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> = {
  typeSchema: readTypeZ,
  configSchema: readConfigZ,
  statusDataSchema: readStatusDataZ,
};
