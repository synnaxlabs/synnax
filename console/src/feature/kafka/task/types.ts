// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { kafka, type task } from "@synnaxlabs/client";
import {
  type CrudeDataType,
  DataType,
  json,
  type payload,
  record,
} from "@synnaxlabs/x";
import { z } from "zod";

import { Task } from "@/platform/task";

export const PREFIX = "kafka";

export type TimeFormat = payload.TimeFormat;

const topicZ = z.string().min(1, "Topic is required");

export const READ_TYPE = `${PREFIX}_read`;

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
