// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { kafka, status, type Synnax } from "@synnaxlabs/client";
import { type record, TimeSpan } from "@synnaxlabs/x";

import type { Device } from "@/feature/kafka/device/types";
import { SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/kafka/task/types";

const TIMEOUT = TimeSpan.seconds(25);

/**
 * @returns the data that answers a command of the Kafka scan task.
 * @throws when the scan task fails the command.
 */
const execute = async (
  client: Synnax,
  device: Device,
  type: string,
  args: record.Unknown,
): Promise<unknown> => {
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: device.rack,
    one: true,
    schemas: SCAN_SCHEMAS,
  });
  const res = await scanTask.executeCommandSync({ type, timeout: TIMEOUT, args });
  if (res.variant !== "success") throw status.toError(res);
  return res.details?.data;
};

/**
 * @returns the topics of a cluster, other than internal topics, sorted by name.
 * @throws when the scan task fails the command.
 */
export const browseTopics = async (
  client: Synnax,
  device: Device,
): Promise<kafka.BrowseResult> =>
  kafka.browseResultZ.parse(
    await execute(client, device, "browse", { device: device.key }),
  );

/**
 * @returns the records that the Kafka scan task reads from a topic of a cluster, and
 * the shapes of their values. The error of the result says why it read fewer records
 * than it asked for.
 * @throws when the scan task fails the command.
 */
export const sampleTopic = async (
  client: Synnax,
  device: Device,
  topic: string,
): Promise<kafka.SampledTopic> => {
  const args = kafka.sampleArgsZ.parse({ device: device.key, topics: [topic] });
  const data = await execute(client, device, "sample", args);
  return kafka.sampleResultZ.parse(data).topics[0];
};
