// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it } from "vitest";

import { Kafka } from "@/feature/kafka";
import { createTestDevice } from "@/platform/device/testutil";
import { createWithHook } from "@/platform/task/testutil";

const client = createTestClient();

describe("Kafka.Task create", () => {
  it("should create a read task on the given device", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      Kafka.Task.useCreateRead,
      Kafka.Task.READ_SCHEMAS,
      {
        deviceKey: dev.key,
      },
    );
    expect(tsk.name).toBe("Kafka read task");
    expect(tsk.config.device).toBe(dev.key);
  });

  it("should create a read task with the given config", async () => {
    const config = Kafka.Task.READ_SCHEMAS.config.parse({ topic: "pressure" });
    const tsk = await createWithHook(
      Kafka.Task.useCreateRead,
      Kafka.Task.READ_SCHEMAS,
      {
        config,
      },
    );
    expect(tsk.config.topic).toBe("pressure");
  });

  it("should create a write task that publishes the channel and timestamp", async () => {
    const dev = await createTestDevice(client);
    const tsk = await createWithHook(
      Kafka.Task.useCreateWrite,
      Kafka.Task.WRITE_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.name).toBe("Kafka write task");
    expect(tsk.config.device).toBe(dev.key);
    expect(tsk.config.record).toMatchObject({
      channelPointer: "/channel",
      timestampPointer: "/timestamp",
    });
  });

  it("should create a write task with the given config", async () => {
    const config = Kafka.Task.WRITE_SCHEMAS.config.parse({ topic: "valves" });
    const tsk = await createWithHook(
      Kafka.Task.useCreateWrite,
      Kafka.Task.WRITE_SCHEMAS,
      { config },
    );
    expect(tsk.config.topic).toBe("valves");
  });
});
