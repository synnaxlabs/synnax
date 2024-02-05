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
  describe("Rack", () => {
    describe("create", () => {
      it("should create a single rack", async () => {
        const r = await client.hardware.createRack({ name: "test" });
        expect(r.key).toBeGreaterThan(0n);
      });
    });
    describe("retrieve", () => {
      it("should retrieve a rack by its key", async () => {
        const r = await client.hardware.createRack({ name: "test" });
        const retrieved = await client.hardware.retrieveRack(r.key);
        expect(retrieved.key).toBe(r.key);
        expect(retrieved.name).toBe("test");
      });
    });
  });
  describe("Task", () => {
    describe("create", () => {
      it("should create a task on a rack", async () => {
        const r = await client.hardware.createRack({ name: "test" });
        const m = await r.createTask({ name: "test", config: {a: "dog"}, type: "ni" });
        expect(m.key).toBeGreaterThan(0n);
        const rackKey = BigInt(m.key) >> 32n;
        expect(Number(rackKey)).toBe(r.key);
      });
    });
    describe("retrieve", () => {
      it("should retrieve a task by its key", async () => {
        const r = await client.hardware.createRack({ name: "test" });
        const m = await r.createTask({ name: "test", config: {"a": "dog"}, type: "ni" });
        const retrieved = await client.hardware.retrieveTask(BigInt(m.key));
        expect(retrieved.key).toBe(m.key);
        expect(retrieved.name).toBe("test");
        expect(retrieved.config).toBe("dog");
        expect(retrieved.type).toBe("ni");
      });
    });
  });
  describe("Device", () => {
    describe("create", () => {
      it("should create a device on a rack", async () => {
        const rack = await client.hardware.createRack({ name: "test" });
        const d = await client.hardware.createDevice({ rack: rack.key, location: "Dev1", key: "SN222", name: "test", make: "ni", model: "dog", properties: "dog" });
        expect(d.key).toEqual("SN222");
        expect(d.name).toBe("test");
        expect(d.make).toBe("ni");
      });
    })
    describe("retrieve", () => {
      it("should retrieve a device by its key", async () => {
        const rack = await client.hardware.createRack({ name: "test" });
        const d = await client.hardware.createDevice({ key: "SN222", 
        rack: rack.key,
        location: "Dev1",
        name: "test", make: "ni", model: "dog", properties: "dog" });
        const retrieved = await client.hardware.retrieveDevice(d.key);
        expect(retrieved.key).toBe(d.key);
        expect(retrieved.name).toBe("test");
        expect(retrieved.make).toBe("ni");
      });
    });
  });
});
