// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { task } from "@/task";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const SCAN_SCHEMAS = {
  type: z.literal("mqtt_scan"),
  config: z.record(z.string(), z.unknown()),
  statusData: z.unknown(),
};

const TIMEOUT = TimeSpan.seconds(10);

const retrieveScanTask = async () => {
  const [rack] = await client.racks.retrieve({
    where: { embedded: { eq: true }, integrations: { any: ["mqtt"] } },
  });
  return await client.tasks.retrieve({
    type: SCAN_SCHEMAS.type.value,
    rack: rack.key,
    schemas: SCAN_SCHEMAS,
  });
};

describe("mqtt scan task", () => {
  it("should answer a connection test with the step that failed", async () => {
    const scanTask = await retrieveScanTask();
    const state = await scanTask.executeCommandSync({
      type: "test_connection",
      timeout: TIMEOUT,
      args: { location: "127.0.0.1", properties: { port: 1 } },
    });
    expect(state.variant).toBe("error");
    expect(state.message).toBe("Failed to reach broker");
    const { steps } = task.testResultZ.parse(state.details.data);
    expect(steps).toHaveLength(1);
    expect(steps[0]).toMatchObject({ stage: "dial", variant: "error" });
  });

  it.each(["browse", "sample"])(
    "should answer a %s of a device that does not exist with an error",
    async (type) => {
      const scanTask = await retrieveScanTask();
      const state = await scanTask.executeCommandSync({
        type,
        timeout: TIMEOUT,
        args: { device: id.create(), topics: ["plant/a"] },
      });
      expect(state.variant).toBe("error");
      expect(state.message).toMatch(/not found/);
    },
  );
});
