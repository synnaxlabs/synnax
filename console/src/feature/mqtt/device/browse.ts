// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type Synnax } from "@synnaxlabs/client";
import { json, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { retrieveScanTask } from "@/feature/mqtt/device/retrieveScanTask";
import {
  BROWSE_COMMAND_TYPE,
  BROWSE_SPARKPLUG_COMMAND_TYPE,
} from "@/feature/mqtt/device/scan";
import type { Device } from "@/feature/mqtt/device/types";
import type { ICD } from "@/platform/icd";

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

/** How long the Core listens to the broker for one browse. */
export const BROWSE_DURATION = TimeSpan.seconds(3);

export const LISTENING = `Listening for ${BROWSE_DURATION.seconds} seconds`;

/** @returns the status text of a browse of the broker that fails. */
export const browseFailure = (device: Device): ICD.Picker.Failure => ({
  message: `Cannot reach ${device.name}`,
  description:
    `The Core cannot connect to ${device.location}. ` +
    "Check that the broker runs and that the Core can reach it.",
});

// The Core listens for up to 20 s before it replies.
const BROWSE_TIMEOUT = TimeSpan.seconds(25);

export interface KeyedTopic extends BrowsedTopic {
  key: string;
}

interface BrowseResult {
  topics: KeyedTopic[];
  truncated: boolean;
}

export const browseTopics = async (
  client: Synnax,
  device: Device,
  filter: string,
): Promise<BrowseResult> => {
  const scanTask = await retrieveScanTask(client, device.rack);
  const res = await scanTask.executeCommandSync({
    type: BROWSE_COMMAND_TYPE,
    timeout: BROWSE_TIMEOUT,
    args: { device: device.key, filter, duration: BROWSE_DURATION.milliseconds },
  });
  if (res.variant !== "success") throw status.toError(res);
  const data = browseResultZ.parse(res.details?.data);
  return {
    topics: data.topics.map((t) => ({ ...t, key: t.topic })),
    truncated: data.truncated,
  };
};

interface BrowseArgs {
  group: string;
  edgeNode: string;
}

export const browseSparkplug = async (
  client: Synnax,
  device: Device,
  { group, edgeNode }: BrowseArgs,
) => {
  const scanTask = await retrieveScanTask(client, device.rack);
  const res = await scanTask.executeCommandSync({
    type: BROWSE_SPARKPLUG_COMMAND_TYPE,
    timeout: BROWSE_TIMEOUT,
    args: {
      device: device.key,
      group,
      edge_node: edgeNode,
      duration: BROWSE_DURATION.milliseconds,
    },
  });
  if (res.variant !== "success") throw status.toError(res);
  return browseSparkplugResultZ.parse(res.details?.data);
};
