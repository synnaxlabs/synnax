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

import { sparkplugIDZ } from "@/feature/mqtt/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "mqtt";

export type TimeFormat = mqtt.TimeFormat;

export type QoS = mqtt.QoS;

export type SparkplugDataType = mqtt.SparkplugDataType;

/** The identity of a Sparkplug B tag. An empty device names a tag of the edge node. */
export interface SparkplugTagID extends Pick<
  mqtt.SparkplugReadEntry,
  "group" | "edgeNode" | "device" | "tag"
> {}

const deployTopicZ = z
  .string()
  .min(1, "Topic is required")
  .regex(/^[^+#]*$/, "Topic must not hold the wildcards + or #");

export const READ_TYPE = `${PREFIX}_read`;

export interface ReadField extends mqtt.ReadField {}

export interface PlainReadEntry extends mqtt.PlainReadEntry {}

export interface SparkplugReadEntry extends mqtt.SparkplugReadEntry {}

export type ReadEntry = PlainReadEntry | SparkplugReadEntry;

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

const validateReadEntry = (ctx: z.core.ParsePayload<PlainReadEntry>) => {
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

const deployPlainReadEntryZ = mqtt.plainReadEntryZ
  .extend({
    topic: deployTopicZ,
    fields: deployReadFieldZ.array().check(Task.validateReadChannels),
  })
  .check(validateReadEntry);

const DEPLOY_SPARKPLUG_TAG_SHAPE = {
  group: sparkplugIDZ("Group").min(1, "Group is required").prefault(""),
  edgeNode: sparkplugIDZ("Edge node").min(1, "Edge node is required").prefault(""),
  device: sparkplugIDZ("Device").prefault(""),
  tag: z.string().min(1, "Tag is required").prefault(""),
};

const deployReadEntryZ = z.discriminatedUnion("type", [
  deployPlainReadEntryZ,
  mqtt.sparkplugReadEntryZ.extend(DEPLOY_SPARKPLUG_TAG_SHAPE),
]);

type Item =
  | Pick<PlainReadEntry, "type" | "disabled" | "topic">
  | Pick<SparkplugReadEntry, "type" | "disabled" | keyof SparkplugTagID>;

// The driver skips disabled items, so they may share a topic or a tag with an enabled
// one.
const validateItems = (label: string) => (ctx: z.core.ParsePayload<Item[]>) => {
  if (ctx.value.every(({ disabled }) => disabled))
    ctx.issues.push({
      code: "custom",
      input: ctx.value,
      message: `At least one ${label} must be enabled`,
      path: [],
    });
  const seen = new Set<string>();
  ctx.value.forEach((item, i) => {
    if (item.disabled) return;
    const [field, name, identity] =
      item.type === "plain"
        ? ["topic", item.topic, [item.topic]]
        : ["tag", item.tag, [item.group, item.edgeNode, item.device, item.tag]];
    if (name === "") return;
    const key = JSON.stringify(identity);
    if (seen.has(key))
      ctx.issues.push({
        code: "custom",
        input: ctx.value,
        message: `Duplicate ${field} "${name}"`,
        path: [i, field],
      });
    else seen.add(key);
  });
};

export const deployReadConfigZ = mqtt.readConfigZ.extend({
  device: Task.deviceKeyZ,
  entries: deployReadEntryZ.array().check(validateItems("entry")),
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

export interface PlainWriteTarget extends mqtt.PlainWriteTarget {}

export interface SparkplugWriteTarget extends mqtt.SparkplugWriteTarget {}

export type WriteTarget = PlainWriteTarget | SparkplugWriteTarget;

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

const validateWriteTarget = (ctx: z.core.ParsePayload<PlainWriteTarget>) => {
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

const deployPlainWriteTargetZ = mqtt.plainWriteTargetZ
  .extend({
    topic: deployTopicZ,
    channel: mqtt.channelFieldZ
      .extend({ pointer: json.pointerZ })
      .check(validateChannelField),
    fields: deployWriteFieldZ.array().default(() => []),
  })
  .check(validateWriteTarget);

const deployWriteTargetZ = z.discriminatedUnion("type", [
  deployPlainWriteTargetZ,
  mqtt.sparkplugWriteTargetZ.extend(DEPLOY_SPARKPLUG_TAG_SHAPE),
]);

export const deployWriteConfigZ = mqtt.writeConfigZ.extend({
  device: Task.deviceKeyZ,
  targets: deployWriteTargetZ.array().check(validateItems("target")),
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

const browseResultZ = z.object({
  topics: browsedTopicZ.array(),
  truncated: z.boolean(),
});

export const BROWSE_SPARKPLUG_COMMAND_TYPE = "browse_sparkplug";

const browsedSparkplugNodeZ = z.object({
  group: z.string(),
  edgeNode: z.string(),
  devices: z.string().array(),
});

export interface BrowsedSparkplugNode extends z.infer<typeof browsedSparkplugNodeZ> {}

const browsedSparkplugTagZ = z.object({
  device: z.string(),
  name: z.string(),
  dataType: z.string(),
  value: z.string(),
  supported: z.boolean(),
});

export interface BrowsedSparkplugTag extends z.infer<typeof browsedSparkplugTagZ> {}

const browseSparkplugResultZ = z.object({
  nodes: browsedSparkplugNodeZ.array(),
  tags: browsedSparkplugTagZ.array(),
});

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.union([browseResultZ, browseSparkplugResultZ]).nullish(),
} as const satisfies task.Schemas;
