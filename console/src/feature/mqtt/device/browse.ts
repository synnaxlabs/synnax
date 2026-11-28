// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mqtt, status, type Synnax } from "@synnaxlabs/client";
import { json, TimeSpan } from "@synnaxlabs/x";
import { z } from "zod";

import { retrieveScanTask } from "@/feature/mqtt/device/retrieveScanTask";
import {
  BROWSE_COMMAND_TYPE,
  BROWSE_SPARKPLUG_COMMAND_TYPE,
  SAMPLE_COMMAND_TYPE,
} from "@/feature/mqtt/device/scan";
import type { Device } from "@/feature/mqtt/device/types";
import type { ICD } from "@/platform/icd";

const browsedTopicZ = z.object({
  topic: z.string(),
  /** The start of the last payload. */
  payload: z.string(),
  /** The shapes of the JSON messages heard, in the order first heard. */
  shapes: json.shapeZ.array(),
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

/**
 * @returns the last message that the broker delivers on topic while the Core listens,
 * or null when none arrives. The Core replies at the first message of a topic with no
 * wildcard.
 * @throws when the scan task fails the command.
 */
export const sampleTopic = async (
  client: Synnax,
  device: Device,
  topic: string,
): Promise<ICD.Sample | null> => {
  const scanTask = await retrieveScanTask(client, device.rack);
  const res = await scanTask.executeCommandSync({
    type: SAMPLE_COMMAND_TYPE,
    timeout: BROWSE_TIMEOUT,
    args: {
      device: device.key,
      topics: [topic],
      count: 1,
      duration: BROWSE_DURATION.milliseconds,
    },
  });
  if (res.variant !== "success") throw status.toError(res);
  const messages = mqtt.sampleResultZ
    .parse(res.details?.data)
    .topics.flatMap((t) => t.messages);
  const last = messages.reduce<mqtt.SampledMessage | null>(
    (a, b) => (a == null || b.time.after(a.time) ? b : a),
    null,
  );
  if (last == null) return null;
  const { time, format, payload } = last;
  return { time, format, payload };
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
