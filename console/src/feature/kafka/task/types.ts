// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type http, kafka, type task } from "@synnaxlabs/client";
import { type CrudeDataType, DataType, json, record } from "@synnaxlabs/x";
import { z } from "zod";

import { Task } from "@/platform/task";

export const PREFIX = "kafka";

export type TimeFormat = http.TimeFormat;

const topicZ = z.string().min(1, "Topic is required");

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadField extends kafka.ReadField {}

export interface ReadRecord extends kafka.ReadRecord {}

export type StartOffset = kafka.StartOffset;

const readConfigZ = kafka.readConfigZ;

const deployReadFieldZ = kafka.readFieldZ
  .refine(({ disabled, pointer }) => disabled || pointer !== "", {
    message: "Pointer is required",
    path: ["pointer"],
  })
  .refine(
    ({ disabled, pointer }) => disabled || json.pointerZ.safeParse(pointer).success,
    { message: "Pointer must be a valid JSON pointer", path: ["pointer"] },
  );

const deployReadRecordZ = kafka.readRecordZ.extend({
  fields: deployReadFieldZ.array().check(Task.validateReadChannels),
});

/** Flags every record whose key an earlier record already claims. */
const validateRecordKeys = (ctx: z.core.ParsePayload<ReadRecord[]>) => {
  const seen = new Set<string>();
  ctx.value.forEach(({ recordKey }, i) => {
    if (!seen.has(recordKey)) {
      seen.add(recordKey);
      return;
    }
    ctx.issues.push({
      code: "custom",
      input: ctx.value,
      message:
        recordKey === ""
          ? "Only one record can read any key"
          : `Record key "${recordKey}" is used by another record`,
      path: [i, "recordKey"],
    });
  });
};

export const deployReadConfigZ = kafka.readConfigZ.extend({
  device: Task.deviceKeyZ,
  topic: topicZ,
  records: deployReadRecordZ
    .array()
    .check(validateRecordKeys)
    .refine(
      (records) => records.some(({ fields }) => fields.some((f) => !f.disabled)),
      {
        message: "At least one field must be enabled",
      },
    ),
});

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const WRITE_TYPE = `${PREFIX}_write`;

export interface WriteChannel extends kafka.WriteChannel {}

export type RecordKey = kafka.RecordKey;

const writeConfigZ = kafka.writeConfigZ;

const deployWriteChannelZ = kafka.writeChannelZ.refine(
  ({ disabled, channel }) => disabled || channel !== 0,
  { message: "Channel is required", path: ["channel"] },
);

export const deployWriteConfigZ = kafka.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  topic: topicZ,
  record: kafka.recordZ
    .extend({
      valuePointer: json.pointerZ.min(1, "Value pointer is required").default("/value"),
    })
    .prefault({}),
  channels: deployWriteChannelZ
    .array()
    .check(Task.validateChannels)
    .refine((channels) => channels.some(({ disabled }) => !disabled), {
      message: "At least one channel must be enabled",
    }),
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
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

/** Whether a value of the data type travels as a JSON string. */
export const isString = (dataType: CrudeDataType): boolean => {
  const dt = new DataType(dataType);
  return dt.isVariable || dt.equals(DataType.UUID);
};
