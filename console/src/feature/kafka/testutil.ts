// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { device, Synnax, task } from "@synnaxlabs/client";
import { payload } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { onTestFinished } from "vitest";

import { Device } from "@/feature/kafka/device";
import { createTestDevice, startRackHeartbeat } from "@/platform/device/testutil";

export interface CreateKafkaDeviceOptions {
  configured?: boolean;
  properties?: Partial<Device.Properties>;
}

/** Creates a rack and a configured Kafka cluster device on the live Core. */
export const createKafkaDevice = async (
  client: Synnax,
  { configured = true, properties }: CreateKafkaDeviceOptions = {},
): Promise<device.Device> =>
  await createTestDevice(client, {
    name: uniqueName("kafka_cluster"),
    make: Device.MAKE,
    model: "cluster",
    configured,
    properties: {
      brokers: ["localhost:9092"],
      secure: false,
      sasl: { mechanism: "none" },
      ...properties,
    },
  });

/** Creates a Kafka scan task on the rack of dev, with a heartbeat. */
export const createScanTask = async (
  client: Synnax,
  dev: device.Device,
): Promise<task.Task> => {
  const rck = await client.racks.retrieve({ key: dev.rack });
  onTestFinished(await startRackHeartbeat(client, rck));
  return await rck.createTask({
    name: uniqueName("kafka_scan"),
    type: Device.SCAN_TYPE,
    config: {},
  });
};

/** @returns a topic of a cluster as a browse lists it, holding ten JSON records. */
export const listTopic = (name: string, overrides: object = {}) => ({
  name,
  partitions: 1,
  records: 10,
  compacted: false,
  rate: 0,
  newest: 0,
  format: "json",
  readable: true,
  error: "",
  ...overrides,
});

/** @returns the data of a browse that lists the topics. */
export const browseOf = (...topics: object[]) => ({ topics, truncated: false });

/** @returns the data of a sample of a topic that read values of the shapes. */
export const sampleOfTopic = (topic: string, ...shapes: object[]) => ({
  topics: [{ topic, records: [], shapes, error: "" }],
  truncated: false,
});

/** @returns a field of a sampled shape, held by two records. */
export const createInferredField = (
  pointer: string,
  dataType: string,
  type: "number" | "boolean" | "string",
  timeFormat?: payload.TimeFormat,
) => ({
  pointer,
  dataType,
  kind: payload.kindOf(dataType),
  type,
  timeFormat,
  enum: [],
  count: 2,
  mismatched: 0,
});
