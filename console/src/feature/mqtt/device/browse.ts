// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type Synnax } from "@synnaxlabs/client";
import { TimeSpan } from "@synnaxlabs/x";

import { retrieveScanTask } from "@/feature/mqtt/device/retrieveScanTask";
import type { Device } from "@/feature/mqtt/device/types";
import {
  BROWSE_COMMAND_TYPE,
  BROWSE_SPARKPLUG_COMMAND_TYPE,
  type BrowsedTopic,
} from "@/feature/mqtt/task/types";

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
  const data = res.details?.data;
  // A sample answer holds topics too, beside its shapes.
  if (data == null || !("topics" in data) || "shapes" in data)
    throw new Error("The scan task replied without topics");
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
  const data = res.details?.data;
  if (data == null || !("nodes" in data))
    throw new Error("The scan task replied without edge nodes");
  return data;
};
