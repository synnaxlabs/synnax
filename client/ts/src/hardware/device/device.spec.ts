// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, unique } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Device", async () => {
  const testRack = await client.hardware.racks.create({ name: "test" });
  describe("create", () => {
    it("should create a device on a rack", async () => {
      const key = id.create();
      const d = await client.hardware.devices.create({
        rack: testRack.key,
        location: "Dev1",
        key,
        name: "test",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      expect(d.key).toEqual(key);
      expect(d.name).toBe("test");
      expect(d.make).toBe("ni");
    });
  });
  it("should properly encode and decode properties", async () => {
    const properties = {
      rate: 10,
      stateIndexChannel: 234,
      inputChannels: { port1: 34214 },
      outputChannels: [{ port2: 232 }],
    };
    const d = await client.hardware.devices.create({
      key: id.create(),
      rack: testRack.key,
      location: "Dev1",
      name: "test",
      make: "ni",
      model: "dog",
      properties,
    });
    const retrieved = await client.hardware.devices.retrieve({ key: d.key });
    expect(retrieved.key).toEqual(d.key);
    expect(retrieved.properties).toEqual(properties);
  });
  describe("retrieve", () => {
    it("should retrieve a device by its key", async () => {
      const d = await client.hardware.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const retrieved = await client.hardware.devices.retrieve({ key: d.key });
      expect(retrieved.key).toBe(d.key);
      expect(retrieved.name).toBe("test");
      expect(retrieved.make).toBe("ni");
    });
    it("should retrieve multiple devices by their keys", async () => {
      const d1 = await client.hardware.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "test1",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const d2 = await client.hardware.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev2",
        name: "test2",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const retrieved = await client.hardware.devices.retrieve({
        keys: [d1.key, d2.key],
      });
      expect(retrieved.length).toBe(2);
      expect(retrieved[0].key).toBe(d1.key);
      expect(retrieved[1].key).toBe(d2.key);
    });

    describe("state", () => {
      it("should not include state by default", async () => {
        const d = await client.hardware.devices.create({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name: "state_test1",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        const retrieved = await client.hardware.devices.retrieve({ key: d.key });
        expect(retrieved.status).toBeUndefined();
      });

      it("should include status when includeStatus is true", async () => {
        const d = await client.hardware.devices.create({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name: "state_test2",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        await expect
          .poll(async () => {
            const { status } = await client.hardware.devices.retrieve({
              key: d.key,
              includeStatus: true,
            });
            return status != null;
          })
          .toBe(true);
      });

      it("should include state for multiple devices", async () => {
        const d1 = await client.hardware.devices.create({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name: "state_test3",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        const d2 = await client.hardware.devices.create({
          key: id.create(),
          rack: testRack.key,
          location: "Dev2",
          name: "state_test4",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        await expect
          .poll(async () => {
            const retrievedDevices = await client.hardware.devices.retrieve({
              keys: [d1.key, d2.key],
              includeStatus: true,
            });
            if (retrievedDevices.length !== 2) return false;
            return retrievedDevices.every(({ status }) => status !== undefined);
          })
          .toBe(true);
      });

      it("should handle state with type-safe details", async () => {
        const key = id.create();
        await client.hardware.devices.create({
          key,
          rack: testRack.key,
          location: "Dev1",
          name: "state_test5",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        await expect
          .poll(async () => {
            const retrieved = await client.hardware.devices.retrieve({
              key,
              includeStatus: true,
            });
            return (
              retrieved.status !== undefined &&
              retrieved.status.variant === "info" &&
              retrieved.status.details.device === key
            );
          })
          .toBe(true);
      });
    });

    describe("request object format", () => {
      const testDevices: Array<{
        key: string;
        name: string;
        make: string;
        model: string;
        location: string;
      }> = [];

      beforeAll(async () => {
        const deviceConfigs = [
          { name: "sensor1", make: "ni", model: "pxi-6281", location: "Lab1" },
          { name: "sensor2", make: "ni", model: "pxi-6284", location: "Lab2" },
          { name: "actuator1", make: "labjack", model: "t7", location: "Lab1" },
          { name: "actuator2", make: "labjack", model: "t4", location: "Lab3" },
          { name: "controller", make: "opc", model: "server", location: "Lab2" },
        ];

        for (const config of deviceConfigs) {
          const key = id.create();
          await client.hardware.devices.create({
            key,
            rack: testRack.key,
            location: config.location,
            name: config.name,
            make: config.make,
            model: config.model,
            properties: { test: true },
          });
          testDevices.push({ key, ...config });
        }
      });

      it("should retrieve devices by names", async () => {
        const result = await client.hardware.devices.retrieve({
          names: ["sensor1", "actuator1"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(unique.unique(result.map((d) => d.name).sort())).toEqual([
          "actuator1",
          "sensor1",
        ]);
      });

      it("should retrieve devices by makes", async () => {
        const result = await client.hardware.devices.retrieve({
          makes: ["ni"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.make === "ni")).toBe(true);
      });

      it("should retrieve devices by models", async () => {
        const result = await client.hardware.devices.retrieve({
          models: ["pxi-6281", "t7"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(unique.unique(result.map((d) => d.model).sort())).toEqual([
          "pxi-6281",
          "t7",
        ]);
      });

      it("should retrieve devices by locations", async () => {
        const result = await client.hardware.devices.retrieve({
          locations: ["Lab1"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.location === "Lab1")).toBe(true);
      });

      it("should retrieve devices by racks", async () => {
        const result = await client.hardware.devices.retrieve({
          racks: [testRack.key],
        });
        expect(result.length).toBeGreaterThanOrEqual(5);
        expect(result.every((d) => d.rack === testRack.key)).toBe(true);
      });

      it("should retrieve devices by search term", async () => {
        const result = await client.hardware.devices.retrieve({
          searchTerm: "sensor1",
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.name.includes("sensor"))).toBe(true);
      });

      it("should support pagination with limit and offset", async () => {
        const firstPage = await client.hardware.devices.retrieve({
          racks: [testRack.key],
          limit: 2,
          offset: 0,
        });
        expect(firstPage).toHaveLength(2);

        const secondPage = await client.hardware.devices.retrieve({
          racks: [testRack.key],
          limit: 2,
          offset: 2,
        });
        expect(secondPage).toHaveLength(2);

        const firstPageKeys = firstPage.map((d) => d.key);
        const secondPageKeys = secondPage.map((d) => d.key);
        expect(firstPageKeys.every((key) => !secondPageKeys.includes(key))).toBe(true);
      });

      it("should support combined filters", async () => {
        const result = await client.hardware.devices.retrieve({
          makes: ["ni"],
          locations: ["Lab1", "Lab2"],
          includeStatus: true,
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(
          result.every((d) => d.make === "ni" && ["Lab1", "Lab2"].includes(d.location)),
        ).toBe(true);

        await expect
          .poll(async () => {
            const devices = await client.hardware.devices.retrieve({
              makes: ["ni"],
              locations: ["Lab1", "Lab2"],
              includeStatus: true,
            });
            return devices.every((d) => d.status !== undefined);
          })
          .toBe(true);
      });
    });
  });
});
