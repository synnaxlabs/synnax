// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { newClient } from "@/setupspecs";

const client = newClient();

describe("Hardware", () => {
  describe("Task", () => {
    describe("create", () => {
      it("should create a task on a rack", async () => {
        const r = await client.hardware.racks.create({ name: "test" });
        const m = await r.createTask({
          name: "test",
          config: { a: "dog" },
          type: "ni",
        });
        expect(m.key).not.toHaveLength(0);
        const rackKey = BigInt(m.key) >> 32n;
        expect(Number(rackKey)).toBe(r.key);
      });
    });
    describe("retrieve", () => {
      it("should retrieve a task by its key", async () => {
        const r = await client.hardware.racks.create({ name: "test" });
        const m = await r.createTask({
          name: "test",
          config: { a: "dog" },
          type: "ni",
        });
        const retrieved = await client.hardware.tasks.retrieve(m.key);
        expect(retrieved.key).toBe(m.key);
        expect(retrieved.name).toBe("test");
        expect(retrieved.config).toStrictEqual({ a: "dog" });
        expect(retrieved.type).toBe("ni");
      });
    });
  });
});
