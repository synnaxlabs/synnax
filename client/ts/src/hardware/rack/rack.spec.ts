// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { ZodError } from "zod";

import { type rack } from "@/hardware/rack";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Rack", () => {
  describe("create", () => {
    it("should create a single rack", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      expect(r.key).toBeGreaterThan(0n);
    });
    it("should return an error if the rack doesn't have a name", async () => {
      // @ts-expect-error - Testing for error
      await expect(client.hardware.racks.create({})).rejects.toThrow(ZodError);
    });
  });
  describe("update", () => {
    it("should update a rack if the key is provided", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      const updated = await client.hardware.racks.create({
        key: r.key,
        name: "updated",
      });
      expect(updated.name).toBe("updated");
      const retrieved = await client.hardware.racks.retrieve({ key: r.key });
      expect(retrieved.name).toBe("updated");
    });
  });
  describe("retrieve", () => {
    it("should retrieve a rack by its key", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      const retrieved = await client.hardware.racks.retrieve({ key: r.key });
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toBe("test");
    });
    it("should retrieve a rack by its name", async () => {
      const name = `TimeStamp.now().toString()}-${Math.random()}`;
      const r = await client.hardware.racks.create({ name });
      const retrieved = await client.hardware.racks.retrieve({ name });
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toEqual(name);
    });
  });
  describe("tasks", () => {
    it("should list the tasks on a rack", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      const tasks = await r.listTasks();
      expect(tasks).toHaveLength(0);
    });
  });
  describe("status", () => {
    it("should include the rack's status when includeStatus is true", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      let status: rack.Status | undefined;
      await expect
        .poll(async () => {
          const retrieved = await client.hardware.racks.retrieve({
            key: r.key,
            includeStatus: true,
          });
          status = retrieved.status;
          return status;
        })
        .toBeDefined();
      expect(status?.details?.rack).toBe(r.key);
    });
    it("should include the status for multiple racks", async () => {
      const r1 = await client.hardware.racks.create({ name: "test1" });
      const r2 = await client.hardware.racks.create({ name: "test2" });
      let statuses: (rack.Status | undefined)[] = [];
      await expect
        .poll(async () => {
          const retrieved = await client.hardware.racks.retrieve({
            keys: [r1.key, r2.key],
            includeStatus: true,
          });
          statuses = retrieved.map((r) => r.status);
          return statuses.every((s) => s != null);
        })
        .toBe(true);
      expect(statuses).toHaveLength(2);
      expect(statuses[0]?.details?.rack).toBe(r1.key);
      expect(statuses[1]?.details?.rack).toBe(r2.key);
    });
  });
});
