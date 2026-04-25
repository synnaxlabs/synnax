// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp, zod } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type rack } from "@/rack";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Rack", () => {
  describe("create", () => {
    it("should create a single rack", async () => {
      const r = await client.racks.create({ name: "test" });
      expect(r.key).toBeGreaterThan(0n);
    });
    it("should return an error if the rack doesn't have a name", async () => {
      // @ts-expect-error - Testing for error
      await expect(client.racks.create({})).rejects.toThrow(zod.ParseError);
    });
    it("should create a rack with a custom status", async () => {
      const customStatus: rack.Status = {
        key: "",
        name: "",
        variant: "success",
        message: "Custom rack status",
        description: "Rack is running",
        time: TimeStamp.now(),
        details: { rack: 0 },
      };
      const r = await client.racks.create({
        name: "rack-with-status",
        status: customStatus,
      });
      expect(r.key).toBeGreaterThan(0n);
      const retrieved = await client.racks.retrieve({
        key: r.key,
        includeStatus: true,
      });
      expect(retrieved.status).toBeDefined();
      expect(retrieved.status?.variant).toBe("success");
      expect(retrieved.status?.message).toBe("Custom rack status");
      expect(retrieved.status?.description).toBe("Rack is running");
      expect(retrieved.status?.details?.rack).toBe(r.key);
    });
  });
  describe("update", () => {
    it("should update a rack if the key is provided", async () => {
      const r = await client.racks.create({ name: "test" });
      const updated = await client.racks.create({
        key: r.key,
        name: "updated",
      });
      expect(updated.name).toBe("updated");
      const retrieved = await client.racks.retrieve({ key: r.key });
      expect(retrieved.name).toBe("updated");
    });
  });
  describe("retrieve", () => {
    it("should retrieve a rack by its key", async () => {
      const r = await client.racks.create({ name: "test" });
      const retrieved = await client.racks.retrieve({ key: r.key });
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toBe("test");
    });
    it("should retrieve a rack by its name", async () => {
      const name = `${TimeStamp.now().toString()}-${Math.random()}`;
      const r = await client.racks.create({ name });
      const retrieved = await client.racks.retrieve({ name });
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toEqual(name);
    });
    it("should retrieve racks by search term", async () => {
      const prefix = `searchable-rack-${id.create()}`;
      await client.racks.create([{ name: `${prefix}-1` }, { name: `${prefix}-2` }]);
      await expect
        .poll(async () => (await client.racks.retrieve({ searchTerm: prefix })).length)
        .toBeGreaterThanOrEqual(2);
      const results = await client.racks.retrieve({ searchTerm: prefix });
      expect(results.every((r) => r.name.includes(prefix))).toBe(true);
    });
  });
  describe("integrations", () => {
    it("should create a rack with integrations and retrieve them", async () => {
      const r = await client.racks.create({
        name: "rack-with-integrations",
        integrations: ["ni", "opc", "modbus"],
      });
      const retrieved = await client.racks.retrieve({ key: r.key });
      expect(retrieved.integrations).toEqual(["ni", "opc", "modbus"]);
    });
    it("should filter racks by integration", async () => {
      const r1 = await client.racks.create({
        name: "rack-ni-opc",
        integrations: ["ni", "opc"],
      });
      const r2 = await client.racks.create({
        name: "rack-modbus-only",
        integrations: ["modbus"],
      });
      const results = await client.racks.retrieve({ integration: "ni" });
      const keys = results.map((r) => r.key);
      expect(keys).toContain(r1.key);
      expect(keys).not.toContain(r2.key);
    });
    it("should update integrations on upsert", async () => {
      const r = await client.racks.create({
        name: "rack-update-integrations",
        integrations: ["ni"],
      });
      await client.racks.create({
        key: r.key,
        name: "rack-update-integrations",
        integrations: ["ni", "opc", "arc"],
      });
      const retrieved = await client.racks.retrieve({ key: r.key });
      expect(retrieved.integrations).toEqual(["ni", "opc", "arc"]);
    });
  });
  describe("tasks", () => {
    it("should list the tasks on a rack", async () => {
      const r = await client.racks.create({ name: "test" });
      const tasks = await r.listTasks();
      expect(tasks).toHaveLength(0);
    });
  });
  describe("status", () => {
    it("should include the rack's status when includeStatus is true", async () => {
      const r = await client.racks.create({ name: "test" });
      let status: rack.Status | undefined;
      await expect
        .poll(async () => {
          const retrieved = await client.racks.retrieve({
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
      const r1 = await client.racks.create({ name: "test1" });
      const r2 = await client.racks.create({ name: "test2" });
      let statuses: (rack.Status | undefined)[] = [];
      await expect
        .poll(async () => {
          const retrieved = await client.racks.retrieve({
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
