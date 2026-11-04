// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type Synnax } from "@synnaxlabs/client";

import { MAKE, type Properties } from "@/feature/kafka/device/types";
import { createTestDevice } from "@/platform/device/testutil";
import { uniqueName } from "@/testutil";

export interface CreateKafkaDeviceOptions {
  configured?: boolean;
  properties?: Partial<Properties>;
}

/** Creates a rack and a configured Kafka cluster device on the live Core. */
export const createKafkaDevice = async (
  client: Synnax,
  { configured = true, properties }: CreateKafkaDeviceOptions = {},
): Promise<device.Device> =>
  await createTestDevice(client, {
    name: uniqueName("kafka_cluster"),
    make: MAKE,
    model: "cluster",
    configured,
    properties: {
      brokers: ["localhost:9092"],
      tls: false,
      sasl: { mechanism: "none" },
      ...properties,
    },
  });
