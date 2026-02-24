// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "http";

const jsonPointerZ = z
  .string()
  .regex(/^(?:$|(?:\/(?:[^~/]|~0|~1)*)+)$/, "must be a valid JSON pointer (RFC 6901)");

const timeFormatZ = z.enum(["iso8601", "unix_sec", "unix_ms", "unix_us", "unix_ns"]);

export const READ_TYPE = `${PREFIX}_read`;

export const readTypeZ = z.literal(READ_TYPE);

const readFieldZ = Common.Task.readChannelZ.extend({
  pointer: jsonPointerZ,
  dataType: z.string().default("float64"),
  timestampFormat: timeFormatZ.optional(),
});
export interface ReadField extends z.infer<typeof readFieldZ> {}

export const ZERO_READ_FIELD = {
  ...Common.Task.ZERO_READ_CHANNEL,
  pointer: "",
  dataType: "float64",
} as const satisfies ReadField;

const baseReadEndpointZ = z.object({
  key: z.string(),
  path: z.string().min(1, "Path is required"),
  headers: z.record(z.string(), z.string()).optional(),
  queryParams: z.record(z.string(), z.string()).optional(),
  fields: z.array(readFieldZ).check(Common.Task.validateReadChannels),
  index: z.string().nullable().default(null),
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

export const ZERO_READ_ENDPOINT = {
  key: "",
  method: "GET",
  path: "",
  fields: [],
  index: null,
} as const satisfies ReadEndpoint;

export const readConfigZ = Common.Task.baseReadConfigZ.extend({
  rate: z.number().positive("Rate must be positive"),
  endpoints: z.array(readEndpointZ),
});
export interface ReadConfig extends z.infer<typeof readConfigZ> {}

export const ZERO_READ_CONFIG = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  rate: 1,
  endpoints: [],
} as const satisfies ReadConfig;

export const readStatusDataZ = z
  .object({ running: z.boolean(), message: z.string() })
  .or(z.null());

export interface ReadPayload extends task.Payload<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
> {}

export const ZERO_READ_PAYLOAD = {
  key: "",
  name: "HTTP Read Task",
  config: ZERO_READ_CONFIG,
  type: READ_TYPE,
} as const satisfies ReadPayload;

export const READ_SCHEMAS = {
  type: readTypeZ,
  config: readConfigZ,
  statusData: readStatusDataZ,
} as const satisfies task.PayloadSchemas<
  typeof readTypeZ,
  typeof readConfigZ,
  typeof readStatusDataZ
>;
