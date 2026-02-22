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
import { id, Rate } from "@synnaxlabs/x";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "http";

export const SCAN_TYPE = `${PREFIX}_scan`;

export const scanTypeZ = z.literal(SCAN_TYPE);

const responseValidationZ = z.object({
  field: z.string().min(1, "JSON pointer is required"),
  expectedValue: JSON.primitiveZ,
});

export interface ResponseValidation extends z.infer<typeof responseValidationZ> {}

export const ZERO_RESPONSE_VALIDATION: ResponseValidation = {
  field: "",
  expectedValue: null,
};

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

const timeFormatZ = z.enum(["iso8601", "unix_sec", "unix_ms", "unix_us", "unix_ns"]);
export type TimeFormat = z.infer<typeof timeFormatZ>;

const timeInfoZ = z.object({
  pointer: z.string().min(1, "Pointer is required"),
  format: timeFormatZ,
});
export interface TimeInfo extends z.infer<typeof timeInfoZ> {}

const readFieldZ = Common.Task.readChannelZ.extend({
  pointer: z.string().min(1, "JSON pointer is required"),
  timestampFormat: timeFormatZ.optional(),
  timePointer: timeInfoZ.optional(),
});
export interface ReadField extends z.infer<typeof readFieldZ> {}

export const ZERO_READ_FIELD: ReadField = {
  ...Common.Task.ZERO_READ_CHANNEL,
  pointer: "",
};

const readEndpointZ = z.object({
  key: z.string(),
  method: z.enum(["GET", "POST"]),
  path: z.string().min(1, "Path is required"),
  queryParams: z.record(z.string(), z.string()).optional(),
  body: z.string().optional(),
  fields: z.array(readFieldZ),
});
export interface ReadEndpoint extends z.infer<typeof readEndpointZ> {}

export const ZERO_READ_ENDPOINT: ReadEndpoint = {
  key: id.create(),
  method: "GET",
  path: "",
  fields: [],
};

export const readConfigZ = Common.Task.baseReadConfigZ.extend({
  rate: z.number().positive("Rate must be positive"),
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
  .object({
    running: z.boolean(),
    message: z.string(),
  })
  .or(z.null());

interface ReadPayload extends task.Payload<
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
