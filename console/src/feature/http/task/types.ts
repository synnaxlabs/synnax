// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { http, type task } from "@synnaxlabs/client";
import { json, payload, record } from "@synnaxlabs/x";
import { z } from "zod";

import {
  checkDuplicateKeys,
  headersZ,
  queryParamsZ,
} from "@/feature/http/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "http";

export const READ_TYPE = `${PREFIX}_read`;

export type ReadField = http.ReadField;

export interface ReadEndpoint extends http.ReadEndpoint {}

const readMethodZ = z.enum(["GET", "POST"]);

const readConfigZ = http.readConfigZ;

const deployEntries = <E extends z.ZodType<{ label: string }>>(entry: E) =>
  entry
    .array()
    .check(checkDuplicateKeys("label", "enum label"))
    .default(() => []);

const deployReadFieldZ = z.discriminatedUnion("type", [
  http.floatReadFieldZ.extend({
    pointer: json.pointerZ,
    enumValues: deployEntries(payload.floatEntryZ),
  }),
  http.integerReadFieldZ.extend({
    pointer: json.pointerZ,
    enumValues: deployEntries(payload.integerEntryZ),
  }),
  http.unsignedReadFieldZ.extend({
    pointer: json.pointerZ,
    enumValues: deployEntries(payload.unsignedEntryZ),
  }),
  http.stringReadFieldZ.extend({
    pointer: json.pointerZ,
    enumValues: deployEntries(payload.stringEntryZ),
  }),
  http.booleanReadFieldZ.extend({
    pointer: json.pointerZ,
    enumValues: deployEntries(payload.booleanEntryZ),
  }),
  http.timestampReadFieldZ.extend({ pointer: json.pointerZ }),
  http.jsonReadFieldZ.extend({ pointer: json.pointerZ }),
  http.uuidReadFieldZ.extend({ pointer: json.pointerZ }),
  http.bytesReadFieldZ.extend({ pointer: json.pointerZ }),
]);

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

export const WRITE_TYPE = `${PREFIX}_write`;

export type ChannelField = http.ChannelField;

const validatePointer = (ctx: z.core.ParsePayload<ChannelField>) => {
  const { value } = ctx;
  const pointer = json.pointerZ.safeParse(value.pointer);
  if (!pointer.success)
    ctx.issues.push({
      code: "custom",
      input: value,
      message: pointer.error.issues[0].message,
      path: ["pointer"],
    });
};

const writeMethodZ = z.enum(["POST", "PUT", "PATCH"]);

export const writeEndpointZ = http.writeEndpointZ;

export interface WriteEndpoint extends z.infer<typeof writeEndpointZ> {}

const writeConfigZ = http.writeConfigZ.extend({
  endpoints: writeEndpointZ.array().default(() => []),
});

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
    if (field.pointer === "") continue;
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

const deployWriteEndpointZ = writeEndpointZ
  .extend({
    method: writeMethodZ,
    headers: headersZ.default(() => []),
    queryParams: queryParamsZ.default(() => []),
    channel: http.channelFieldZ.check(validatePointer),
    fields: Task.Payload.deployWriteFieldZ.array(),
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

export const SCAN_TYPE = `${PREFIX}_scan`;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.null().optional(),
} as const satisfies task.Schemas;
