// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { http, type task } from "@synnaxlabs/client";
import { z } from "zod";

import { Device } from "@/feature/http/device";
import { Task } from "@/platform/task";

export const PREFIX = "http";

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadEndpoint extends http.ReadEndpoint {}

const readMethodZ = z.enum(["GET", "POST"]);

const readConfigZ = http.readConfigZ;

const deployReadEndpointZ = http.readEndpointZ.extend({
  method: readMethodZ,
  headers: Device.headersZ.default(() => []),
  queryParams: Device.queryParamsZ.default(() => []),
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

const writeMethodZ = z.enum(["POST", "PUT", "PATCH"]);

export interface WriteEndpoint extends http.WriteEndpoint {}

const writeConfigZ = http.writeConfigZ;

const deployWriteEndpointZ = http.writeEndpointZ.extend({
  method: writeMethodZ,
  headers: Device.headersZ.default(() => []),
  queryParams: Device.queryParamsZ.default(() => []),
});

/** Refuses an enabled endpoint that sends no message. */
const validateMessages = ({
  value,
  issues,
}: z.core.ParsePayload<Pick<WriteEndpoint, "disabled" | "icd" | "message">[]>) =>
  value.forEach(({ disabled, icd, message }, i) => {
    if (disabled || (icd != null && message != null)) return;
    issues.push({
      continue: true,
      code: "custom",
      input: value,
      message: "A message is required",
      path: [i, "message"],
    });
  });

export const deployWriteConfigZ = writeConfigZ.extend({
  device: Task.deviceKeyZ,
  endpoints: deployWriteEndpointZ.array().check(validateMessages),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;
