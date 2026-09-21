// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mqtt, type task } from "@synnaxlabs/client";
import { DataType, json, record } from "@synnaxlabs/x";
import { z } from "zod";

import { Task } from "@/platform/task";

export const PREFIX = "mqtt";

export type TimeFormat = mqtt.TimeFormat;

export type QoS = mqtt.QoS;

const UNSUPPORTED_SPARKPLUG_MESSAGE = "Sparkplug B is not supported yet";

const deployTopicZ = z
  .string()
  .min(1, "Topic is required")
  .regex(/^[^+#]*$/, "Topic must not hold the wildcards + or #");

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadField extends mqtt.ReadField {}

export interface ReadEntry extends mqtt.PlainReadEntry {}

const validateEnumLabels = (ctx: z.core.ParsePayload<mqtt.EnumEntry[]>) => {
  const seen = new Set<string>();
  ctx.value.forEach(({ label }, i) => {
    if (seen.has(label))
      ctx.issues.push({
        code: "custom",
        input: ctx.value,
        message: `Duplicate enum label "${label}"`,
        path: [i, "label"],
      });
    else seen.add(label);
  });
};

const deployReadFieldZ = mqtt.readFieldZ.extend({
  pointer: json.pointerZ,
  enumValues: mqtt.enumEntryZ
    .array()
    .check(validateEnumLabels)
    .default(() => []),
});

const isVariable = (field: ReadField): boolean =>
  new DataType(field.dataType).isVariable;

const validateReadEntry = (ctx: z.core.ParsePayload<ReadEntry>) => {
  const { value } = ctx;
  const pointers = new Set<string>();
  const enabled = value.fields.filter(({ disabled }) => !disabled);
  // A variable-length channel is virtual and has no index.
  const mixed = enabled.some(isVariable) && !enabled.every(isVariable);
  value.fields.forEach((field, i) => {
    if (mixed && !field.disabled && isVariable(field))
      ctx.issues.push({
        code: "custom",
        input: value,
        message:
          "The fields of a topic share one index, so a variable-length data type cannot mix with the others",
        path: ["fields", i, "dataType"],
      });
    if (pointers.has(field.pointer))
      ctx.issues.push({
        code: "custom",
        input: value,
        message: `Pointer "${field.pointer}" is already used by another field`,
        path: ["fields", i, "pointer"],
      });
    else pointers.add(field.pointer);
    const isTimestamp =
      field.key === value.index || DataType.TIMESTAMP.equals(field.dataType);
    if (isTimestamp && field.timeFormat == null)
      ctx.issues.push({
        code: "custom",
        input: value,
        message: "A timestamp field requires a time format",
        path: ["fields", i, "timeFormat"],
      });
  });
};

const deployReadEntryZ = mqtt.plainReadEntryZ
  .extend({
    type: z.literal("plain", UNSUPPORTED_SPARKPLUG_MESSAGE),
    topic: deployTopicZ,
    fields: deployReadFieldZ.array().check(Task.validateReadChannels),
  })
  .check(validateReadEntry);

interface Topic {
  topic: string;
  disabled: boolean;
}

// The driver skips disabled entries, so they may share a topic with an enabled one.
const validateTopics = (label: string) => (ctx: z.core.ParsePayload<Topic[]>) => {
  if (ctx.value.every(({ disabled }) => disabled))
    ctx.issues.push({
      code: "custom",
      input: ctx.value,
      message: `At least one ${label} must be enabled`,
      path: [],
    });
  const seen = new Set<string>();
  ctx.value.forEach(({ topic, disabled }, i) => {
    if (disabled || topic === "") return;
    if (seen.has(topic))
      ctx.issues.push({
        code: "custom",
        input: ctx.value,
        message: `Duplicate topic "${topic}"`,
        path: [i, "topic"],
      });
    else seen.add(topic);
  });
};

export const deployReadConfigZ = mqtt.readConfigZ.extend({
  device: Task.deviceKeyZ,
  entries: deployReadEntryZ.array().check(validateTopics("entry")),
});

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: mqtt.readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export interface ReadPayload extends task.Payload<ReadSchemas> {}

export const WRITE_TYPE = `${PREFIX}_write`;

export type GeneratorType = mqtt.GeneratorType;

export interface ChannelField extends mqtt.ChannelField {}

export type WriteField = mqtt.WriteField;

export interface WriteTarget extends mqtt.PlainWriteTarget {}

const validateChannelField = (ctx: z.core.ParsePayload<ChannelField>) => {
  const { value } = ctx;
  if (value.enumValues.length > 0 && value.jsonType !== "string")
    ctx.issues.push({
      code: "custom",
      input: value,
      message: "Enum values require the string JSON type",
      path: ["jsonType"],
    });
  if (DataType.TIMESTAMP.equals(value.dataType) && value.timeFormat == null)
    ctx.issues.push({
      code: "custom",
      input: value,
      message: "A timestamp channel requires a time format",
      path: ["timeFormat"],
    });
  const seen = new Set<number>();
  value.enumValues.forEach((entry, i) => {
    if (seen.has(entry.value))
      ctx.issues.push({
        code: "custom",
        input: value,
        message: `Duplicate enum value ${entry.value}`,
        path: ["enumValues", i, "value"],
      });
    else seen.add(entry.value);
  });
};

const validateWriteTarget = (ctx: z.core.ParsePayload<WriteTarget>) => {
  const { value } = ctx;
  const { channel, fields } = value;
  if (channel.pointer === "" && fields.length > 0)
    ctx.issues.push({
      code: "custom",
      input: value,
      message:
        "An empty channel pointer sends the raw value as the payload, so additional fields are not allowed",
      path: ["channel", "pointer"],
    });
  const pointers = new Set<string>([channel.pointer]);
  fields.forEach((field, i) => {
    if (field.pointer === "") return;
    if (pointers.has(field.pointer))
      ctx.issues.push({
        code: "custom",
        input: value,
        message: `Pointer "${field.pointer}" is already used by another field`,
        path: ["fields", i, "pointer"],
      });
    else pointers.add(field.pointer);
  });
};

const deployPointerZ = json.pointerZ.min(1, "Additional field pointer cannot be empty");

const deployWriteFieldZ = z.discriminatedUnion("type", [
  mqtt.staticWriteFieldZ.extend({ pointer: deployPointerZ, value: json.primitiveZ }),
  mqtt.generatedWriteFieldZ.extend({ pointer: deployPointerZ }),
]);

const deployWriteTargetZ = mqtt.plainWriteTargetZ
  .extend({
    type: z.literal("plain", UNSUPPORTED_SPARKPLUG_MESSAGE),
    topic: deployTopicZ,
    channel: mqtt.channelFieldZ
      .extend({ pointer: json.pointerZ })
      .check(validateChannelField),
    fields: deployWriteFieldZ.array().default(() => []),
  })
  .check(validateWriteTarget);

export const deployWriteConfigZ = mqtt.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  targets: deployWriteTargetZ.array().check(validateTopics("target")),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: mqtt.writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export interface WritePayload extends task.Payload<WriteSchemas> {}

export const SCAN_TYPE = `${PREFIX}_scan`;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

const browsedTopicZ = z.object({
  topic: z.string(),
  payload: z.string(),
  retained: z.boolean(),
});

export interface BrowsedTopic extends z.infer<typeof browsedTopicZ> {}

const browseResultZ = z
  .object({ topics: browsedTopicZ.array(), truncated: z.boolean() })
  .nullish()
  .optional();

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: browseResultZ,
} as const satisfies task.Schemas;
