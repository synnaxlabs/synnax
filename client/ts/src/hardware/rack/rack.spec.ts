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

describe("Rack", () => {
  describe("create", () => {
    it("should create a single rack", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      expect(r.key).toBeGreaterThan(0n);
    });
  });
  describe("retrieve", () => {
    it("should retrieve a rack by its key", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      const retrieved = await client.hardware.racks.retrieve(r.key);
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toBe("test");
    });
  });
  describe("tasks", () => {
    it("should list the tasks on a rack", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      const tasks = await r.listTasks();
      expect(tasks).toHaveLength(0);
    });
  })
});
