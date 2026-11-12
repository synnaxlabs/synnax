// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, mqtt, type task } from "@synnaxlabs/client";
import { DataType, json, payload, record } from "@synnaxlabs/x";
import { z } from "zod";

import { sparkplugIDZ } from "@/feature/mqtt/device/types";
import { Task } from "@/platform/task";

export const PREFIX = "mqtt";

export type QoS = mqtt.QoS;

export type SparkplugDataType = mqtt.SparkplugDataType;

/** The identity of a Sparkplug B tag. An empty device names a tag of the edge node. */
export interface SparkplugTagID extends Pick<
  mqtt.SparkplugReadEntry,
  "group" | "edgeNode" | "device" | "tag"
> {}

/** @returns the key of a Sparkplug B edge node. */
export const nodeKey = ({
  group,
  edgeNode,
}: Pick<SparkplugTagID, "group" | "edgeNode">): string => `${group}/${edgeNode}`;

/**
 * @returns the key of a Sparkplug B tag. The device level is always present, because a
 * tag name can hold a slash.
 */
export const tagKey = (id: SparkplugTagID): string =>
  `${nodeKey(id)}/${id.device}/${id.tag}`;

const deployTopicZ = z
  .string()
  .min(1, "Topic is required")
  .regex(/^[^+#]*$/, "Topic must not hold the wildcards + or #");

// The Core skips disabled items, so only the enabled ones must be complete.
const validateEnabled =
  (strict: z.ZodType) => (ctx: z.core.ParsePayload<{ disabled: boolean }[]>) =>
    ctx.value.forEach((item, i) => {
      if (item.disabled) return;
      const result = strict.safeParse(item);
      if (result.success) return;
      for (const { message, path } of result.error.issues)
        ctx.issues.push({ code: "custom", input: item, message, path: [i, ...path] });
    });

export const READ_TYPE = `${PREFIX}_read`;

export type ReadField = mqtt.ReadField;

/** @returns the fields that every kind of read field shares. */
export const readBaseOf = ({
  key,
  name,
  disabled,
  channel,
  pointer,
}: ReadField): mqtt.BaseReadField => ({ key, name, disabled, channel, pointer });

/** @returns a read field of base with the kind and the options. */
export const createReadField = (
  options: Task.Payload.ReadOptions,
  base: Partial<mqtt.BaseReadField> = {},
): ReadField => mqtt.readFieldZ.parse({ ...base, ...options });

/**
 * @returns field with the kind that a channel of the given data type takes. A field of
 * that kind comes back as it is.
 */
export const matchDataType = (field: ReadField, dataType: string): ReadField =>
  payload.kindOf(dataType) === field.type
    ? field
    : { ...readBaseOf(field), ...Task.Payload.readOptionsOf(dataType, field) };

export interface PlainReadEntry extends mqtt.PlainReadEntry {}

export interface SparkplugReadEntry extends mqtt.SparkplugReadEntry {}

export type ReadEntry = PlainReadEntry | SparkplugReadEntry;

const enumLabelsOf = (field: ReadField): string[] => {
  switch (field.type) {
    case "float":
    case "integer":
    case "unsigned":
    case "string":
    case "boolean":
      return field.enumValues.map(({ label }) => label);
    default:
      return [];
  }
};

const validateReadField = (ctx: z.core.ParsePayload<ReadField>) => {
  const { value } = ctx;
  const pointer = json.pointerZ.safeParse(value.pointer);
  if (!pointer.success)
    for (const { message } of pointer.error.issues)
      ctx.issues.push({ code: "custom", input: value, message, path: ["pointer"] });
  const seen = new Set<string>();
  enumLabelsOf(value).forEach((label, i) => {
    if (label === "" && value.type !== "string")
      ctx.issues.push({
        code: "custom",
        input: value,
        message: "Enum label cannot be empty",
        path: ["enumValues", i, "label"],
      });
    else if (seen.has(label))
      ctx.issues.push({
        code: "custom",
        input: value,
        message: `Duplicate enum label "${label}"`,
        path: ["enumValues", i, "label"],
      });
    else seen.add(label);
  });
};

const deployReadFieldZ = mqtt.readFieldZ.check(validateReadField);

const isVariable = (field: ReadField): boolean =>
  new DataType(Task.Payload.dataTypeOf(field)).isVariable;

const validateReadEntry = (ctx: z.core.ParsePayload<PlainReadEntry>) => {
  const { value } = ctx;
  const pointers = new Set<string>();
  const enabled = value.fields.filter(({ disabled }) => !disabled);
  // A variable-length channel is virtual and has no index.
  const mixed = enabled.some(isVariable) && !enabled.every(isVariable);
  value.fields.forEach((field, i) => {
    if (field.disabled) return;
    if (mixed && isVariable(field))
      ctx.issues.push({
        code: "custom",
        input: value,
        message:
          "The fields of a topic share one index, so a variable-length data type cannot mix with the others",
        path: ["fields", i, "type"],
      });
    if (pointers.has(field.pointer))
      ctx.issues.push({
        code: "custom",
        input: value,
        message: `Pointer "${field.pointer}" is already used by another field`,
        path: ["fields", i, "pointer"],
      });
    else pointers.add(field.pointer);
    if (field.key === value.index && field.type !== "timestamp")
      ctx.issues.push({
        code: "custom",
        input: value,
        message: "The index field must read a timestamp",
        path: ["fields", i, "type"],
      });
  });
};

const deployPlainReadEntryZ = mqtt.plainReadEntryZ
  .extend({
    topic: deployTopicZ,
    fields: mqtt.readFieldZ
      .array()
      .check(validateEnabled(deployReadFieldZ), Task.validateReadChannels),
  })
  .check(validateReadEntry);

const DEPLOY_SPARKPLUG_TAG_SHAPE = {
  group: sparkplugIDZ("Group").min(1, "Group is required"),
  edgeNode: sparkplugIDZ("Edge node").min(1, "Edge node is required"),
  device: sparkplugIDZ("Device"),
  tag: z.string().min(1, "Tag is required"),
};

const deployReadEntryZ = z.discriminatedUnion("type", [
  deployPlainReadEntryZ,
  mqtt.sparkplugReadEntryZ.extend(DEPLOY_SPARKPLUG_TAG_SHAPE),
]);

type Item =
  | Pick<PlainReadEntry, "type" | "disabled" | "topic">
  | Pick<SparkplugReadEntry, "type" | "disabled" | keyof SparkplugTagID>;

// The Core skips disabled items, so they may share a topic or a tag with an enabled
// one.
const validateItems = (label: string) => (ctx: z.core.ParsePayload<Item[]>) => {
  if (ctx.value.every(({ disabled }) => disabled))
    ctx.issues.push({
      code: "custom",
      input: ctx.value,
      message: `At least one ${label} must be enabled`,
      path: [],
    });
  const seen = { plain: new Set<string>(), sparkplug: new Set<string>() };
  ctx.value.forEach((item, i) => {
    if (item.disabled) return;
    const [field, name, key] =
      item.type === "plain"
        ? ["topic", item.topic, item.topic]
        : ["tag", item.tag, tagKey(item)];
    if (name === "") return;
    if (seen[item.type].has(key))
      ctx.issues.push({
        code: "custom",
        input: ctx.value,
        message: `Duplicate ${field} "${name}"`,
        path: [i, field],
      });
    else seen[item.type].add(key);
  });
};

export const deployReadConfigZ = mqtt.readConfigZ.extend({
  device: Task.deviceKeyZ,
  entries: mqtt.readEntryZ
    .array()
    .check(validateEnabled(deployReadEntryZ), validateItems("entry")),
});

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: mqtt.readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const WRITE_TYPE = `${PREFIX}_write`;

export interface ChannelField extends mqtt.ChannelField {}

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

// The Core sets each pointer in turn, so one inside another replaces or blocks it.
const overlaps = (a: string, b: string) =>
  a === b || a.startsWith(`${b}/`) || b.startsWith(`${a}/`);

const validateWriteTarget = (ctx: z.core.ParsePayload<PlainWriteTarget>) => {
  const { value } = ctx;
  const { channel, fields } = value;
  if (channel.pointer === "") {
    if (fields.length > 0)
      ctx.issues.push({
        code: "custom",
        input: value,
        message:
          "An empty channel pointer sends the raw value as the payload, so additional fields are not allowed",
        path: ["channel", "pointer"],
      });
    return;
  }
  const pointers = [channel.pointer];
  fields.forEach((field, i) => {
    if (field.pointer === "") return;
    if (pointers.some((pointer) => overlaps(pointer, field.pointer)))
      ctx.issues.push({
        code: "custom",
        input: value,
        message: `Pointer "${field.pointer}" overlaps another field`,
        path: ["fields", i, "pointer"],
      });
    else pointers.push(field.pointer);
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
  targets: mqtt.writeTargetZ
    .array()
    .check(validateEnabled(deployWriteTargetZ), validateItems("target")),
});

export const WRITE_SCHEMAS = {
  type: z.literal(WRITE_TYPE),
  config: mqtt.writeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type WriteSchemas = typeof WRITE_SCHEMAS;

export const EDGE_TYPE = `${PREFIX}_sparkplug_edge`;

export interface EdgeTag extends mqtt.EdgeTag {}

// Every birth of an edge node declares these tags.
const RESERVED_TAG_NAMES = new Set(["bdSeq", "Node Control/Rebirth"]);

const deployEdgeTagZ = mqtt.edgeTagZ.extend({
  name: z
    .string()
    .min(1, "Name is required")
    .refine((name) => !RESERVED_TAG_NAMES.has(name), "Sparkplug B reserves the name"),
  channel: channel.keyZ.refine((key) => key !== 0, "A channel is required"),
});

// Two enabled tags may publish one channel under two names, so only the name is
// checked for duplicates.
const validateEdgeTags = (ctx: z.core.ParsePayload<EdgeTag[]>) => {
  if (ctx.value.every(({ disabled }) => disabled))
    ctx.issues.push({
      code: "custom",
      input: ctx.value,
      message: "At least one tag must be enabled",
      path: [],
    });
  const seen = new Set<string>();
  ctx.value.forEach(({ disabled, name }, i) => {
    if (disabled || name === "") return;
    if (seen.has(name))
      ctx.issues.push({
        code: "custom",
        input: ctx.value,
        message: `Duplicate name "${name}"`,
        path: [i, "name"],
      });
    else seen.add(name);
  });
};

export const deployEdgeConfigZ = mqtt.edgeConfigZ.extend({
  device: Task.deviceKeyZ,
  group: sparkplugIDZ("Group").min(1, "Group is required"),
  edgeNode: sparkplugIDZ("Edge node").min(1, "Edge node is required"),
  tags: mqtt.edgeTagZ.array().check(validateEnabled(deployEdgeTagZ), validateEdgeTags),
});

export const EDGE_SCHEMAS = {
  type: z.literal(EDGE_TYPE),
  config: mqtt.edgeConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type EdgeSchemas = typeof EDGE_SCHEMAS;

export const SCAN_TYPE = `${PREFIX}_scan`;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

const browsedFieldZ = z.object({
  pointer: z.string(),
  type: payload.jsonTypeZ,
  /** The start of the JSON text of the value. */
  sample: z.string(),
});

export interface BrowsedField extends z.infer<typeof browsedFieldZ> {}

const browsedTopicZ = z.object({
  topic: z.string(),
  /** The start of the last payload. */
  payload: z.string(),
  /** The scalars of the whole last payload. */
  fields: browsedFieldZ.array(),
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

/** A Sparkplug B tag and the data type the broker reports for it. */
export interface SparkplugTag
  extends SparkplugTagID, Pick<BrowsedSparkplugTag, "dataType"> {}

const browseSparkplugResultZ = z.object({
  nodes: browsedSparkplugNodeZ.array(),
  tags: browsedSparkplugTagZ.array(),
});

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z
    .union([mqtt.sampleResultZ, browseResultZ, browseSparkplugResultZ])
    .nullish(),
} as const satisfies task.Schemas;
