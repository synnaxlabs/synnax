// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { kafka } from "@/kafka";
import { task } from "@/task";
import { createTestClient } from "@/testutil";

const client = createTestClient();

const READ_SCHEMAS = {
  type: z.literal("kafka_read"),
  config: kafka.readConfigZ,
  statusData: z.unknown(),
};

describe("kafka", () => {
  it("should keep a non-finite float fill and entry through a save", async () => {
    const config = kafka.readConfigZ.parse({
      records: [
        {
          key: "r",
          fields: [
            {
              type: "float",
              key: "f",
              pointer: "/v",
              fill: "NaN",
              enumValues: [{ label: "MAX", value: "Infinity" }],
            },
          ],
        },
      ],
    });
    const created = await client.tasks.create(
      { name: "kafka_read", type: "kafka_read", config },
      READ_SCHEMAS,
    );
    const retrieved = await client.tasks.retrieve({
      key: created.key,
      schemas: READ_SCHEMAS,
    });
    const field = retrieved.config.records[0].fields[0];
    expect(field.type).toBe("float");
    if (field.type !== "float") return;
    expect(field.fill).toBeNaN();
    expect(field.enumValues[0].value).toBe(Infinity);
  });

  it("should keep a non-finite float that an action sets", async () => {
    const config = kafka.readConfigZ.parse({
      records: [{ key: "r", fields: [{ type: "float", key: "f", fill: 1.5 }] }],
    });
    const created = await client.tasks.create(
      { name: "kafka_read", type: "kafka_read", config },
      READ_SCHEMAS,
    );
    await client.tasks.dispatch(
      created.key,
      task.set({ path: ["config", "records", "r", "fields", "f", "fill"], value: NaN }),
    );
    const retrieved = await createTestClient().tasks.retrieve({
      key: created.key,
      schemas: READ_SCHEMAS,
    });
    const field = retrieved.config.records[0].fields[0];
    if (field.type !== "float") throw new Error("expected a float field");
    expect(field.fill).toBeNaN();
  });
});
