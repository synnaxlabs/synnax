// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { kafka, type Synnax } from "@synnaxlabs/client";
import { TimeSpan } from "@synnaxlabs/x";

import { type Device } from "@/feature/kafka/device/types";
import { SCAN_SCHEMAS, SCAN_TYPE } from "@/feature/kafka/task/types";

const SAMPLE_COMMAND_TYPE = "sample";

const SAMPLE_TIMEOUT = TimeSpan.seconds(25);

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
  const scanTask = await client.tasks.retrieve({
    type: SCAN_TYPE,
    rack: device.rack,
    one: true,
    schemas: SCAN_SCHEMAS,
  });
  const { variant, message, description, details } = await scanTask.executeCommandSync({
    type: SAMPLE_COMMAND_TYPE,
    timeout: SAMPLE_TIMEOUT,
    args: kafka.sampleArgsZ.parse({ device: device.key, topics: [topic] }),
  });
  if (variant !== "success")
    throw new Error([message, description].filter(Boolean).join(": "));
  return kafka.sampleResultZ.parse(details?.data).topics[0];
};
