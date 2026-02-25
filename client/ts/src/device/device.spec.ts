// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp, unique } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";

import { type device } from "@/device";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Device", async () => {
  const testRack = await client.racks.create({ name: "test" });
  describe("create", () => {
    it("should create a device on a rack", async () => {
      const key = id.create();
      const d = await client.devices.create({
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
    it("should create a device with a custom status", async () => {
      const key = id.create();
      const customStatus: device.Status = {
        key: "",
        name: "",
        variant: "success",
        message: "Custom device status",
        description: "Device is connected",
        time: TimeStamp.now(),
        details: { rack: 0, device: "" },
      };
      const d = await client.devices.create({
        rack: testRack.key,
        location: "Dev1",
        key,
        name: "device-with-status",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
        status: customStatus,
      });
      expect(d.key).toEqual(key);
      const retrieved = await client.devices.retrieve({
        key: d.key,
        includeStatus: true,
      });
      expect(retrieved.status).toBeDefined();
      expect(retrieved.status?.variant).toBe("success");
      expect(retrieved.status?.message).toBe("Custom device status");
      expect(retrieved.status?.description).toBe("Device is connected");
      expect(retrieved.status?.details?.device).toBe(d.key);
    });
  });

  it("should properly encode and decode properties", async () => {
    const properties = {
      rate: 10,
      stateIndexChannel: 234,
      inputChannels: { port1: 34214 },
      outputChannels: [{ port2: 232 }],
    };
    const d = await client.devices.create({
      key: id.create(),
      rack: testRack.key,
      location: "Dev1",
      name: "test",
      make: "ni",
      model: "dog",
      properties,
    });
    const retrieved = await client.devices.retrieve({ key: d.key });
    expect(retrieved.key).toEqual(d.key);
    expect(retrieved.properties).toEqual(properties);
  });

  describe("retrieve", () => {
    it("should retrieve a device by its key", async () => {
      const d = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const retrieved = await client.devices.retrieve({ key: d.key });
      expect(retrieved.key).toBe(d.key);
      expect(retrieved.name).toBe("test");
      expect(retrieved.make).toBe("ni");
    });

    it("should retrieve multiple devices by their keys", async () => {
      const d1 = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "test1",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const d2 = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev2",
        name: "test2",
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
      const retrieved = await client.devices.retrieve({
        keys: [d1.key, d2.key],
      });
      expect(retrieved.length).toBe(2);
      expect(retrieved[0].key).toBe(d1.key);
      expect(retrieved[1].key).toBe(d2.key);
    });

    describe("status", () => {
      it("should not include status by default", async () => {
        const d = await client.devices.create({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name: "state_test1",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        const retrieved = await client.devices.retrieve({ key: d.key });
        expect(retrieved.status).toBeUndefined();
      });

      it("should include status when includeStatus is true", async () => {
        const d = await client.devices.create({
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
            const { status } = await client.devices.retrieve({
              key: d.key,
              includeStatus: true,
            });
            return status != null;
          })
          .toBe(true);
      });

      it("should include status for multiple devices", async () => {
        const d1 = await client.devices.create({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name: "state_test3",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        const d2 = await client.devices.create({
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
            const retrievedDevices = await client.devices.retrieve({
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
        await client.devices.create({
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
            const retrieved = await client.devices.retrieve({
              key,
              includeStatus: true,
            });
            return (
              retrieved.status !== undefined &&
              retrieved.status.variant === "warning" &&
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
          await client.devices.create({
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
        const result = await client.devices.retrieve({
          names: ["sensor1", "actuator1"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(unique.unique(result.map((d) => d.name).sort())).toEqual([
          "actuator1",
          "sensor1",
        ]);
      });

      it("should retrieve devices by makes", async () => {
        const result = await client.devices.retrieve({
          makes: ["ni"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.make === "ni")).toBe(true);
      });

      it("should retrieve devices by models", async () => {
        const result = await client.devices.retrieve({
          models: ["pxi-6281", "t7"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(unique.unique(result.map((d) => d.model).sort())).toEqual([
          "pxi-6281",
          "t7",
        ]);
      });

      it("should retrieve devices by locations", async () => {
        const result = await client.devices.retrieve({
          locations: ["Lab1"],
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.location === "Lab1")).toBe(true);
      });

      it("should retrieve devices by racks", async () => {
        const result = await client.devices.retrieve({
          racks: [testRack.key],
        });
        expect(result.length).toBeGreaterThanOrEqual(5);
        expect(result.every((d) => d.rack === testRack.key)).toBe(true);
      });

      it("should retrieve devices by search term", async () => {
        const result = await client.devices.retrieve({
          searchTerm: "sensor1",
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.name.includes("sensor"))).toBe(true);
      });

      it("should support pagination with limit and offset", async () => {
        const firstPage = await client.devices.retrieve({
          racks: [testRack.key],
          limit: 2,
          offset: 0,
        });
        expect(firstPage).toHaveLength(2);

        const secondPage = await client.devices.retrieve({
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
        const result = await client.devices.retrieve({
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
            const devices = await client.devices.retrieve({
              makes: ["ni"],
              locations: ["Lab1", "Lab2"],
              includeStatus: true,
            });
            return devices.every((d) => d.status !== undefined);
          })
          .toBe(true);
      });
    });

    describe("with schemas", () => {
      const propertiesSchema = z.object({
        rate: z.number(),
        channels: z.array(z.string()),
      });

      it("should create and retrieve with typed properties", async () => {
        const d = await client.devices.create(
          {
            key: id.create(),
            rack: testRack.key,
            location: "Dev1",
            name: "typed-device",
            make: "ni",
            model: "pxi-6281",
            properties: { rate: 1000, channels: ["ai0", "ai1"] },
          },
          { properties: propertiesSchema },
        );
        expect(d.properties.rate).toBe(1000);
        expect(d.properties.channels).toEqual(["ai0", "ai1"]);

        const retrieved = await client.devices.retrieve({
          key: d.key,
          schemas: { properties: propertiesSchema },
        });
        expect(retrieved.properties.rate).toBe(1000);
        expect(retrieved.properties.channels).toEqual(["ai0", "ai1"]);
      });

      it("should create and retrieve with typed make and model", async () => {
        const makeSchema = z.literal("labjack");
        const modelSchema = z.enum(["t7", "t4", "t8"]);

        const d = await client.devices.create(
          {
            key: id.create(),
            rack: testRack.key,
            location: "Dev1",
            name: "typed-make-model",
            make: "labjack",
            model: "t7",
            properties: {},
          },
          { make: makeSchema, model: modelSchema },
        );
        expect(d.make).toBe("labjack");
        expect(d.model).toBe("t7");
      });

      it("should retrieve multiple devices with schemas", async () => {
        const d1 = await client.devices.create(
          {
            key: id.create(),
            rack: testRack.key,
            location: "Dev1",
            name: "schema-multi-1",
            make: "ni",
            model: "pxi",
            properties: { rate: 100, channels: ["ch1"] },
          },
          { properties: propertiesSchema },
        );
        const d2 = await client.devices.create(
          {
            key: id.create(),
            rack: testRack.key,
            location: "Dev2",
            name: "schema-multi-2",
            make: "ni",
            model: "pxi",
            properties: { rate: 200, channels: ["ch2", "ch3"] },
          },
          { properties: propertiesSchema },
        );

        const retrieved = await client.devices.retrieve({
          keys: [d1.key, d2.key],
          schemas: { properties: propertiesSchema },
        });
        expect(retrieved).toHaveLength(2);
        expect(retrieved[0].properties.rate).toBe(100);
        expect(retrieved[1].properties.rate).toBe(200);
      });
    });
  });
});
