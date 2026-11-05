// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("kafka", () => {
  it("should stop a test at a broker that does not answer", async () => {
    const steps = await client.kafka.test({ brokers: ["127.0.0.1:1"] });
    expect(steps).toHaveLength(1);
    expect(steps[0]).toMatchObject({
      stage: "dial",
      variant: "error",
      message: "Failed to reach broker",
    });
  });

  it("should not browse a device that does not exist", async () => {
    await expect(client.kafka.browse(id.create())).rejects.toThrow(NotFoundError);
  });

  it("should not sample a device that does not exist", async () => {
    await expect(
      client.kafka.sample({ device: id.create(), topics: ["readings"] }),
    ).rejects.toThrow(NotFoundError);
  });
});
