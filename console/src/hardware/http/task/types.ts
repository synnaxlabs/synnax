// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, type task } from "@synnaxlabs/client";
import { DataType, json } from "@synnaxlabs/x";
import { z } from "zod";

import { Common } from "@/hardware/common";
import {
  checkDuplicateKeys,
  headersZ,
  queryParamsZ,
} from "@/hardware/http/device/types";

export type { HeaderEntry, QueryParamEntry } from "@/hardware/http/device/types";

export const PREFIX = "http";

const timeFormatZ = z.enum(["iso8601", "unix_sec", "unix_ms", "unix_us", "unix_ns"]);
export type TimeFormat = z.infer<typeof timeFormatZ>;

export const READ_TYPE = `${PREFIX}_read`;

const readEnumEntryZ = z.object({ label: z.string(), value: z.number() });
export interface ReadEnumEntry extends z.infer<typeof readEnumEntryZ> {}

const v0ReadEnumValuesZ = z.record(z.string(), z.number());
const v1ReadEnumValuesZ = z.array(readEnumEntryZ);
const readEnumValuesZ = v1ReadEnumValuesZ.or(
  v0ReadEnumValuesZ.transform((rec) =>
    Object.entries(rec).map(([label, value]) => ({ label, value })),
  ),
);

const readFieldZ = Common.Task.readChannelZ.extend({
  pointer: json.pointerZ,
  dataType: DataType.z,
  timestampFormat: timeFormatZ.optional(),
  enumValues: readEnumValuesZ
    .check(checkDuplicateKeys("label", "enum label"))
    .optional(),
});

export interface ReadField extends z.infer<typeof readFieldZ> {}

export const ZERO_READ_FIELD = {
  ...Common.Task.ZERO_READ_CHANNEL,
  pointer: "",
  dataType: DataType.FLOAT64,
} as const satisfies ReadField;

const baseReadEndpointZ = z.object({
  key: z.string(),
  path: z.string().min(1, "Path is required"),
  headers: headersZ.optional(),
  queryParams: queryParamsZ.optional(),
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

export type ReadMethod = ReadEndpoint["method"];

export const ZERO_READ_ENDPOINT = {
  key: "",
  method: "GET",
  path: "",
  fields: [],
  index: null,
} as const satisfies ReadEndpoint;

const readConfigZ = Common.Task.baseReadConfigZ.extend({
  rate: z.number().positive("Rate must be positive"),
  endpoints: z.array(readEndpointZ),
});

interface ReadConfig extends z.infer<typeof readConfigZ> {}

const ZERO_READ_CONFIG = {
  ...Common.Task.ZERO_BASE_READ_CONFIG,
  rate: 1,
  endpoints: [],
} as const satisfies ReadConfig;

const readStatusDataZ = z
  .object({ running: z.boolean(), message: z.string() })
  .or(z.null());

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: readStatusDataZ,
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export interface ReadPayload extends task.Payload<ReadSchemas> {}

export const ZERO_READ_PAYLOAD = {
  key: "",
  name: "HTTP Read Task",
  config: ZERO_READ_CONFIG,
  type: "http_read",
} as const satisfies ReadPayload;

export const WRITE_TYPE = `${PREFIX}_write`;

const jsonTypeZ = z.enum(["number", "string", "boolean"]);

const writeEnumEntryZ = z.object({ value: z.number(), label: z.string() });

const channelFieldZ = z
  .object({
    pointer: json.pointerZ,
    jsonType: jsonTypeZ,
    channel: channel.keyZ.default(0),
    name: z.string().default(""),
    dataType: DataType.z.default(DataType.FLOAT64),
    timeFormat: timeFormatZ.optional(),
    enumValues: z.array(writeEnumEntryZ).optional(),
  })
  .check((ctx) => {
    const { enumValues } = ctx.value;
    if (enumValues == null) return;
    const seen = new Set<number>();
    enumValues.forEach((entry, i) => {
      if (seen.has(entry.value))
        ctx.issues.push({
          code: "custom",
          input: ctx.value,
          message: `Duplicate enum value ${entry.value}`,
          path: ["enumValues", i, "value"],
        });
      else seen.add(entry.value);
    });
  });

export interface ChannelField extends z.infer<typeof channelFieldZ> {}

export const ZERO_CHANNEL_FIELD = {
  pointer: "",
  jsonType: "number",
  channel: 0,
  name: "",
  dataType: DataType.FLOAT64,
} as const satisfies ChannelField;

const generatorTypeZ = z.enum(["uuid", "timestamp"]);
export type GeneratorType = z.infer<typeof generatorTypeZ>;

const staticFieldZ = z.object({
  key: z.string(),
  pointer: json.pointerZ.min(1, "Pointer cannot be empty"),
  jsonType: jsonTypeZ,
  type: z.literal("static"),
  value: json.primitiveZ,
});

const generatedFieldZ = z.object({
  key: z.string(),
  pointer: json.pointerZ.min(1, "Pointer cannot be empty"),
  type: z.literal("generated"),
  generator: generatorTypeZ,
  timeFormat: timeFormatZ.optional(),
});

const writeFieldZ = z.discriminatedUnion("type", [staticFieldZ, generatedFieldZ]);

export type WriteField = z.infer<typeof writeFieldZ>;

const writeMethodZ = z.enum(["POST", "PUT", "PATCH"]);
export type WriteMethod = z.infer<typeof writeMethodZ>;

const writeEndpointZ = z
  .object({
    enabled: z.boolean().default(true),
    key: z.string(),
    path: z.string().min(1, "Path is required"),
    method: writeMethodZ,
    headers: headersZ.optional(),
    queryParams: queryParamsZ.optional(),
    channel: channelFieldZ,
    fields: z.array(writeFieldZ),
  })
  .check((ctx) => {
    const { value } = ctx;
    const { channel, fields } = value;
    const isBarePrimitive = channel.pointer === "";
    if (isBarePrimitive && fields.length > 0)
      ctx.issues.push({
        code: "custom",
        input: value,
        message:
          "An empty channel pointer sends the raw value as the body, so additional fields are not allowed",
        path: ["channel", "pointer"],
      });
    const pointers = new Set<string>();
    pointers.add(channel.pointer);
    for (const [i, field] of fields.entries()) {
      if (field.pointer === "") {
        ctx.issues.push({
          code: "custom",
          input: value,
          message: "Additional field pointer cannot be empty",
          path: ["fields", i, "pointer"],
        });
        continue;
      }
      if (pointers.has(field.pointer))
        ctx.issues.push({
          code: "custom",
          input: value,
          message: `Pointer "${field.pointer}" is already used by another field`,
          path: ["fields", i, "pointer"],
        });
      else pointers.add(field.pointer);
    }
  });

export type WriteEndpoint = z.infer<typeof writeEndpointZ>;

export const ZERO_WRITE_ENDPOINT = {
  enabled: true,
  key: "",
  method: "POST",
  path: "",
  channel: ZERO_CHANNEL_FIELD,
  fields: [],
} as const satisfies WriteEndpoint;

const writeConfigZ = z.object({
  device: z.string(),
  autoStart: z.boolean().default(false),
  endpoints: z.array(writeEndpointZ),
});

interface WriteConfig extends z.infer<typeof writeConfigZ> {}

const ZERO_WRITE_CONFIG = {
  device: "",
  autoStart: false,
  endpoints: [],
} as const satisfies WriteConfig;

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const ZERO_WRITE_PAYLOAD = {
  key: "",
  name: "HTTP Write Task",
  config: ZERO_WRITE_CONFIG,
  type: WRITE_TYPE,
} as const satisfies WritePayload;

export const SCAN_TYPE = `${PREFIX}_scan`;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: z.null(),
  statusData: z.null(),
} as const satisfies task.Schemas;
