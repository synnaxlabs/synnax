// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mqtt, type task } from "@synnaxlabs/client";
import { payload, record } from "@synnaxlabs/x";
import { z } from "zod";

export const SCAN_TYPE = "mqtt_scan";

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
