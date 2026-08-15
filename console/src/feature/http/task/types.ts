// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { http, type task } from "@synnaxlabs/client";
import { json, record } from "@synnaxlabs/x";
import { z } from "zod";

import {
  checkDuplicateKeys,
  headersZ,
  queryParamsZ,
} from "@/feature/http/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "http";

export type TimeFormat = http.TimeFormat;

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadField extends http.ReadField {}

export interface ReadEndpoint extends http.ReadEndpoint {}

const readMethodZ = z.enum(["GET", "POST"]);
export type ReadMethod = z.infer<typeof readMethodZ>;

export const readConfigZ = http.readConfigZ;

export interface ReadConfig extends http.ReadConfig {}

const deployReadFieldZ = http.readFieldZ.extend({
  pointer: json.pointerZ,
  enumValues: http.enumEntryZ
    .array()
    .check(checkDuplicateKeys("label", "enum label"))
    .default(() => []),
});

const deployReadEndpointZ = http.readEndpointZ.extend({
  method: readMethodZ,
  headers: headersZ.default(() => []),
  queryParams: queryParamsZ.default(() => []),
  fields: deployReadFieldZ.array().check(Task.validateReadChannels),
});

export const deployReadConfigZ = http.readConfigZ.extend({
  device: Task.deviceKeyZ,
  rate: z.number().positive("Rate must be positive"),
  endpoints: deployReadEndpointZ.array(),
});

const readStatusDataZ = z
  .object({ running: z.boolean(), message: z.string() })
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

export type GeneratorType = http.GeneratorType;

export interface ChannelField extends http.ChannelField {}

const validateEnumValues = (ctx: z.core.ParsePayload<ChannelField>) => {
  const { enumValues } = ctx.value;
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
};

// The form edits primitive values; the stored shape accepts any JSON.
const writeFieldStaticZ = http.writeFieldStaticZ.extend({
  value: json.primitiveZ,
});

const writeFieldZ = z.discriminatedUnion("type", [
  http.writeFieldStaticZ,
  http.writeFieldGeneratedZ,
]);

export type WriteField = z.infer<typeof writeFieldZ>;

const writeMethodZ = z.enum(["POST", "PUT", "PATCH"]);
export type WriteMethod = z.infer<typeof writeMethodZ>;

export const writeEndpointZ = http.writeEndpointZ.extend({
  fields: writeFieldZ.array().default(() => []),
});

export interface WriteEndpoint extends z.infer<typeof writeEndpointZ> {}

export const writeConfigZ = http.writeConfigZ.extend({
  endpoints: writeEndpointZ.array().default(() => []),
});

export interface WriteConfig extends z.infer<typeof writeConfigZ> {}

const validateWriteEndpoint = (ctx: z.core.ParsePayload<WriteEndpoint>) => {
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
};

const deployPointerZ = json.pointerZ.min(1, "Pointer cannot be empty");

const deployWriteFieldZ = z.discriminatedUnion("type", [
  writeFieldStaticZ.extend({ pointer: deployPointerZ }),
  http.writeFieldGeneratedZ.extend({ pointer: deployPointerZ }),
]);

const deployWriteEndpointZ = writeEndpointZ
  .extend({
    method: writeMethodZ,
    headers: headersZ.default(() => []),
    queryParams: queryParamsZ.default(() => []),
    channel: http.channelFieldZ
      .extend({ pointer: json.pointerZ })
      .check(validateEnumValues),
    fields: deployWriteFieldZ.array(),
  })
  .check(validateWriteEndpoint);

export const deployWriteConfigZ = writeConfigZ.extend({
  device: Task.deviceKeyZ,
  endpoints: deployWriteEndpointZ.array(),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const SCAN_TYPE = `${PREFIX}_scan`;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.null().optional(),
} as const satisfies task.Schemas;
