// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mqtt, type task } from "@synnaxlabs/client";
import { json, record } from "@synnaxlabs/x";
import { z } from "zod";

export const SCAN_TYPE = "mqtt_scan";

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const BROWSE_COMMAND_TYPE = "browse";

const browsedFieldZ = json.inferredFieldZ
  .pick({ pointer: true, type: true, kind: true, timeFormat: true })
  .extend({
    /** The start of the JSON text of the last value heard. */
    sample: z.string(),
  });

export interface BrowsedField extends z.infer<typeof browsedFieldZ> {}

/** The JSON messages of a topic that hold the same fields. */
const browsedShapeZ = z.object({
  fields: browsedFieldZ.array(),
  /** The pointer of the timestamp in fields that indexes the messages. */
  index: z.string().optional(),
  /** The pointer of the field that only the messages of the shape hold. */
  present: z.string().optional(),
  /** The number of messages heard. */
  count: z.number(),
});

const browsedTopicZ = z.object({
  topic: z.string(),
  /** The start of the last payload. */
  payload: z.string(),
  /** The shapes of the JSON messages heard, in the order first heard. */
  shapes: browsedShapeZ.array(),
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
  statusData: z
    .union([mqtt.sampleResultZ, browseResultZ, browseSparkplugResultZ])
    .nullish(),
} as const satisfies task.Schemas;
