// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { ZodError } from "zod/v4";

import { NotFoundError } from "@/errors";
import { newClient } from "@/setupspecs";

const client = newClient();

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
      const retrieved = await client.hardware.racks.retrieve(r.key);
      expect(retrieved.name).toBe("updated");
    });
  });
  describe("retrieve", () => {
    it("should retrieve a rack by its key", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      const retrieved = await client.hardware.racks.retrieve(r.key);
      expect(retrieved.key).toBe(r.key);
      expect(retrieved.name).toBe("test");
    });
    it("should retrieve a rack by its name", async () => {
      const name = `TimeStamp.now().toString()}-${Math.random()}`;
      const r = await client.hardware.racks.create({ name });
      const retrieved = await client.hardware.racks.retrieve(name);
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
    it("should throw an error if a task cannot be found by name", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      await expect(
        async () => await r.retrieveTaskByName("nonexistent"),
      ).rejects.toThrow(NotFoundError);
    });
  });
  describe("state", () => {
    it("should include state when includeState is true", async () => {
      const r = await client.hardware.racks.create({ name: "test" });
      await expect
        .poll(async () => {
          const retrieved = await client.hardware.racks.retrieve(r.key, {
            includeState: true,
          });
          return (
            retrieved.state !== undefined &&
            retrieved.state.lastReceived instanceof TimeStamp &&
            retrieved.state.key === r.key
          );
        })
        .toBeTruthy();
    });
    it("should include state for multiple racks", async () => {
      const r1 = await client.hardware.racks.create({ name: "test1" });
      const r2 = await client.hardware.racks.create({ name: "test2" });

      await expect
        .poll(async () => {
          const retrieved = await client.hardware.racks.retrieve([r1.key, r2.key], {
            includeState: true,
          });
          return retrieved.every(
            (rack) =>
              rack.state !== undefined &&
              rack.state.lastReceived instanceof TimeStamp &&
              rack.state.key === rack.key,
          );
        })
        .toBeTruthy();
    });
  });
});
