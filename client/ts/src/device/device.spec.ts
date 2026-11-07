// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, record, TimeStamp, unique } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { device } from "@/device";
import { NotFoundError } from "@/index";
import { query } from "@/query";
import { status } from "@/status";
import {
  createTestClient,
  expectLive,
  holdResponses,
  spyOnSend,
  waitForStreamLive,
} from "@/testutil";

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
        details: { rack: 0, device: "", connected: true },
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
    it("should answer a cached status query for new devices", async () => {
      const local = createTestClient();
      await local.connect();
      const created = await local.devices.create(
        ["cached-status-1", "cached-status-2"].map((name) => ({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name,
          make: "ni",
          model: "dog",
          properties: {},
        })),
      );
      created.forEach(({ key }) => {
        const cached = local.devices.getCached({ key, includeStatus: true });
        expect(expectLive(cached).status?.key).toBe(device.statusKey(key));
      });
    });
    it("should keep the live status of an existing device", async () => {
      const local = createTestClient();
      await local.connect();
      const {
        status: _,
        parent: __,
        ...d
      } = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "live-status",
        make: "ni",
        model: "dog",
        properties: {},
      });
      await client.statuses.set(
        status.create<typeof device.statusDetailsZ>({
          key: device.statusKey(d.key),
          variant: "success",
          message: "Device is running",
          details: { rack: d.rack, device: d.key, connected: true },
        }),
      );
      const created = await local.devices.create(d);
      expect(created.status?.message).toBe("Device is running");
      const cached = local.devices.getCached({ key: d.key, includeStatus: true });
      expect(expectLive(cached).status?.message).toBe("Device is running");
    });
  });

  describe("toUpdate", () => {
    it("should keep the stored status when a create reply is sent back", async () => {
      const key = id.create();
      const reply = await client.devices.create({
        rack: testRack.key,
        location: "Dev1",
        key,
        name: "reply-sent-back",
        make: "ni",
        model: "dog",
        properties: {},
      });
      await client.statuses.set<typeof device.statusDetailsZ>({
        key: device.statusKey(key),
        name: reply.name,
        variant: "success",
        message: "Device connected",
        time: TimeStamp.now(),
        details: { rack: testRack.key, device: key },
      });
      await client.devices.create(device.toUpdate(reply));
      const { status } = await client.devices.retrieve({
        key,
        includeStatus: true,
        schemas: {
          properties: record.unknownZ(),
          make: z.string(),
          model: z.string(),
        },
      });
      expect(status?.message).toBe("Device connected");
    });
  });

  describe("optimistic create", () => {
    const WRITE_FAILED = new Error("write failed");
    const fail = () => {
      throw WRITE_FAILED;
    };

    const createNew = (): device.New => ({
      key: id.create(),
      rack: testRack.key,
      location: "Dev1",
      name: "optimistic",
      make: "ni",
      model: "dog",
      properties: { cat: "dog" },
    });

    it("should cache the device before the write commits", async () => {
      const dev = createNew();
      let duringWrite: query.Cached<device.Device> | undefined;
      await client.devices.create(dev, undefined, {
        onOptimistic: () => {
          duringWrite = client.devices.getCached(dev.key);
        },
      });
      expect(expectLive(duringWrite).name).toEqual(dev.name);
    });

    it("should cache every device of a batch before the write commits", async () => {
      const first = createNew();
      const second = createNew();
      let duringWrite: Array<query.Cached<device.Device> | undefined> = [];
      await client.devices.create([first, second], undefined, {
        onOptimistic: () => {
          duringWrite = [first, second].map(({ key }) => client.devices.getCached(key));
        },
      });
      expect(duringWrite.map((d) => expectLive(d).key)).toEqual([
        first.key,
        second.key,
      ]);
    });

    it("should drop the optimistic device when the write fails", async () => {
      const dev = createNew();
      await expect(
        client.devices.create(dev, undefined, { onOptimistic: fail }),
      ).rejects.toBe(WRITE_FAILED);
      expect(client.devices.getCached(dev.key)).toBeUndefined();
    });

    it("should restore the previous device when the write fails", async () => {
      const dev = createNew();
      const created = await client.devices.create(dev);
      await expect(
        client.devices.create({ ...dev, name: "replacement" }, undefined, {
          onOptimistic: fail,
        }),
      ).rejects.toBe(WRITE_FAILED);
      expect(expectLive(client.devices.getCached(dev.key)).name).toEqual(created.name);
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
    const retrieved = await client.devices.retrieve(d.key);
    expect(retrieved.key).toEqual(d.key);
    expect(retrieved.properties).toEqual(properties);
  });

  describe("rename", () => {
    const createDevice = async (name: string) =>
      await client.devices.create({
        rack: testRack.key,
        location: "Dev1",
        key: id.create(),
        name,
        make: "ni",
        model: "dog",
        properties: { cat: "dog" },
      });
    it("should keep a rename that lands before an older fetch", async () => {
      const d = await createDevice(`device-${id.create()}`);
      const local = createTestClient();
      await local.connect();
      await waitForStreamLive(local.connection);
      await local.devices.retrieve(d.key);
      const { responded, release } = holdResponses(local, "/device/retrieve");
      const fetching = local.devices.retrieve({ where: { name: { eq: d.name } } });
      await responded;
      const name = `device-renamed-${id.create()}`;
      await client.devices.rename(d.key, name);
      await expect
        .poll(() => expectLive(local.devices.getCached(d.key)).name)
        .toBe(name);
      release();
      await fetching;
      expect(expectLive(local.devices.getCached(d.key)).name).toBe(name);
    });
    it("should rename a device and keep its properties", async () => {
      const d = await createDevice("device-rename");
      await client.devices.rename(d.key, "device-renamed");
      const retrieved = await createTestClient().devices.retrieve(d.key);
      expect(retrieved.name).toBe("device-renamed");
      expect(retrieved.properties).toEqual({ cat: "dog" });
      expect(retrieved.make).toBe("ni");
      expect(retrieved.rack).toBe(testRack.key);
    });
    it("should update the cache of another client", async () => {
      const remote = createTestClient();
      const d = await createDevice("device-rename-remote");
      expect((await client.devices.retrieve(d.key)).name).toBe("device-rename-remote");
      const name = `device-renamed-${id.create()}`;
      await waitForStreamLive(client.connection);
      await remote.devices.rename(d.key, name);
      await expect
        .poll(async () => (await client.devices.retrieve(d.key)).name)
        .toBe(name);
    });
    it("should reject an empty name", async () => {
      const d = await createDevice("device-rename-empty");
      await expect(client.devices.rename(d.key, "")).rejects.toThrow();
      expect((await createTestClient().devices.retrieve(d.key)).name).toBe(
        "device-rename-empty",
      );
    });
  });

  describe("retrieve", () => {
    it("coalesces concurrent single retrieves into one request", async () => {
      const created = await Promise.all(
        [id.create(), id.create()].map(
          async (key) =>
            await client.devices.create({
              key,
              rack: testRack.key,
              location: `coalesce_${key}`,
              name: `coalesce-${key}`,
              make: "ni",
              model: "dog",
              properties: {},
            }),
        ),
      );
      const local = createTestClient();
      await local.connect();
      const send = spyOnSend(local);
      const res = await Promise.all(
        created.map(async ({ key }) => await local.devices.retrieve(key)),
      );
      expect(res.map(({ key }) => key)).toEqual(created.map(({ key }) => key));
      expect(
        send.mock.calls.filter(([target]) => target === "/device/retrieve"),
      ).toHaveLength(1);
    });

    it("does not reject concurrent retrieves when a key in the window is missing", async () => {
      const d = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "isolation",
        name: "isolation",
        make: "ni",
        model: "dog",
        properties: {},
      });
      const local = createTestClient();
      await local.connect();
      const [ok, missing] = await Promise.allSettled([
        local.devices.retrieve(d.key),
        local.devices.retrieve(`missing-${id.create()}`),
      ]);
      expect(ok.status).toEqual("fulfilled");
      expect(missing.status).toEqual("rejected");
      if (missing.status === "rejected")
        expect(NotFoundError.matches(missing.reason)).toBe(true);
    });

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
      const retrieved = await client.devices.retrieve(d.key);
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
        where: { key: { in: [d1.key, d2.key] } },
      });
      expect(retrieved.length).toBe(2);
      expect(retrieved[0].key).toBe(d1.key);
      expect(retrieved[1].key).toBe(d2.key);
    });

    it("should retrieve devices by a key list in key order", async () => {
      const [d1, d2] = await client.devices.create(
        ["Dev1", "Dev2"].map((location) => ({
          key: id.create(),
          rack: testRack.key,
          location,
          name: location,
          make: "ni",
          model: "dog",
          properties: {},
        })),
      );
      const retrieved = await client.devices.retrieve([d2.key, d1.key]);
      expect(retrieved.map(({ key }) => key)).toEqual([d2.key, d1.key]);
    });

    it("should throw a NotFoundError for a missing key in a key list", async () => {
      const d = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "dog",
        properties: {},
      });
      const missing = id.create();
      await expect(client.devices.retrieve([d.key, missing])).rejects.toThrow(
        new NotFoundError(`device with key ${missing} not found`),
      );
    });

    it("should return one device for a plain key beside other fields", async () => {
      const d = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "test",
        make: "ni",
        model: "dog",
        properties: {},
      });
      const retrieved: device.Device = await client.devices.retrieve({
        key: d.key,
        make: "ni",
      });
      expect(retrieved.key).toEqual(d.key);
      await expect(
        client.devices.retrieve({ key: d.key, make: "labjack" }),
      ).rejects.toThrow(NotFoundError);
    });

    it("should treat a missing key as a non-match", async () => {
      const d = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "present",
        make: "ni",
        model: "dog",
        properties: {},
      });
      const retrieved = await createTestClient().devices.retrieve({
        where: { key: { in: [d.key, id.create()] } },
      });
      expect(retrieved.map(({ key }) => key)).toEqual([d.key]);
    });

    describe("status", () => {
      it("should not include status by default", async () => {
        // A cold client: cached answers carry status whenever one is already
        // in the cache, so absence is only deterministic with nothing cached.
        const cold = createTestClient();
        const d = await cold.devices.create({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name: "state_test1",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });

        const retrieved = await cold.devices.retrieve(d.key);
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

      describe("device cached without a status", () => {
        const createNew = (name: string): device.New => ({
          key: id.create(),
          rack: testRack.key,
          location: "Dev1",
          name,
          make: "ni",
          model: "dog",
          properties: {},
        });

        it("should fetch the status after a status-free read", async () => {
          const d = await client.devices.create(createNew("status-free-read"));
          const local = createTestClient();
          await local.devices.retrieve(d.key);
          const params = { key: d.key, includeStatus: true };
          expect(local.devices.getCached(params)).toBeUndefined();
          const retrieved = await local.devices.retrieve(params);
          expect(retrieved.status?.key).toBe(device.statusKey(d.key));
        });

        it("should fetch the status after a device arrives on the stream", async () => {
          // A rack of its own, as a rack outage writes its devices' statuses.
          const { key: rack } = await client.racks.create({ name: "stream-cached" });
          const d = { ...createNew("stream-cached"), rack };
          await client.devices.create(d);
          const local = createTestClient();
          const stop = local.devices.onChange(d.key, () => {});
          await waitForStreamLive(local.connection);
          try {
            // An update that keeps the name and the parent writes no status.
            await client.devices.create({ ...d, location: "Dev2" });
            await expect
              .poll(() => {
                const cached = local.devices.getCached(d.key);
                return query.isLive(cached) ? cached.location : undefined;
              })
              .toBe("Dev2");
            const params = { key: d.key, includeStatus: true };
            expect(local.devices.getCached(params)).toBeUndefined();
            const retrieved = await local.devices.retrieve(params);
            expect(retrieved.status?.key).toBe(device.statusKey(d.key));
          } finally {
            stop();
          }
        });

        it("should fetch the status for a subscriber with no fetch", async () => {
          const d = await client.devices.create(createNew("subscribed"));
          const local = createTestClient();
          await local.devices.retrieve(d.key);
          const seen: (device.Status | undefined)[] = [];
          const stop = local.devices.onChange(
            { key: d.key, includeStatus: true },
            (cached) => {
              if (query.isLive(cached)) seen.push(cached.status);
            },
          );
          try {
            await expect.poll(() => seen.at(-1)?.key).toBe(device.statusKey(d.key));
            expect(seen.every((st) => st != null)).toBe(true);
          } finally {
            stop();
          }
        });

        it("should keep the answer of a fetch that found no status", async () => {
          const d = await client.devices.create(createNew("no-status-row"));
          await client.statuses.delete(device.statusKey(d.key));
          const local = createTestClient();
          await local.devices.retrieve(d.key);
          const params = { key: d.key, includeStatus: true };
          const retrieved = await local.devices.retrieve(params);
          expect(retrieved.status).toBeUndefined();
          const cached = local.devices.getCached(params);
          expect(expectLive(cached).key).toBe(d.key);
          expect(expectLive(cached).status).toBeUndefined();
        });

        it("should answer a status-free query from the cache", async () => {
          const d = await client.devices.create(createNew("status-free-query"));
          const local = createTestClient();
          await local.devices.retrieve(d.key);
          expect(expectLive(local.devices.getCached(d.key)).key).toBe(d.key);
        });
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
              where: { key: { in: [d1.key, d2.key] } },
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
          where: { name: { in: ["sensor1", "actuator1"] } },
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(unique.unique(result.map((d) => d.name).sort())).toEqual([
          "actuator1",
          "sensor1",
        ]);
      });

      it("should retrieve devices by makes", async () => {
        const result = await client.devices.retrieve({
          where: { make: { eq: "ni" } },
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.make === "ni")).toBe(true);
      });

      it("should retrieve devices by models", async () => {
        const result = await client.devices.retrieve({
          where: { model: { in: ["pxi-6281", "t7"] } },
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(unique.unique(result.map((d) => d.model).sort())).toEqual([
          "pxi-6281",
          "t7",
        ]);
      });

      it("should retrieve devices by locations", async () => {
        const result = await client.devices.retrieve({
          where: { location: { eq: "Lab1" } },
        });
        expect(result.length).toBeGreaterThanOrEqual(2);
        expect(result.every((d) => d.location === "Lab1")).toBe(true);
      });

      it("should retrieve devices by racks", async () => {
        const result = await client.devices.retrieve({
          where: { rack: { eq: testRack.key } },
        });
        expect(result.length).toBeGreaterThanOrEqual(5);
        expect(result.every((d) => d.rack === testRack.key)).toBe(true);
      });

      it("should retrieve devices by search term", async () => {
        const sensor1 = testDevices.find((d) => d.name === "sensor1")!;
        await expect
          .poll(async () => {
            const result = await client.devices.retrieve({ searchTerm: "sensor1" });
            return result.find((d) => d.key === sensor1.key);
          })
          .toMatchObject({ name: "sensor1", key: sensor1.key });
      });

      it("should support pagination with limit and offset", async () => {
        const firstPage = await client.devices.retrieve({
          where: { rack: { eq: testRack.key } },
          limit: 2,
          offset: 0,
        });
        expect(firstPage).toHaveLength(2);

        const secondPage = await client.devices.retrieve({
          where: { rack: { eq: testRack.key } },
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
          where: { make: { eq: "ni" }, location: { in: ["Lab1", "Lab2"] } },
          includeStatus: true,
        });
        expect(result.length).toBeGreaterThanOrEqual(1);
        expect(
          result.every((d) => d.make === "ni" && ["Lab1", "Lab2"].includes(d.location)),
        ).toBe(true);

        await expect
          .poll(async () => {
            const devices = await client.devices.retrieve({
              where: { make: { eq: "ni" }, location: { in: ["Lab1", "Lab2"] } },
              includeStatus: true,
            });
            return devices.every((d) => d.status !== undefined);
          })
          .toBe(true);
      });
    });

    describe("with schemas", () => {
      const schemas = {
        properties: z.object({
          rate: z.number(),
          channels: z.array(z.string()),
        }),
        make: z.string(),
        model: z.string(),
      };

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
          schemas,
        );
        expect(d.properties.rate).toBe(1000);
        expect(d.properties.channels).toEqual(["ai0", "ai1"]);

        const retrieved = await client.devices.retrieve({
          key: d.key,
          schemas,
        });
        expect(retrieved.properties.rate).toBe(1000);
        expect(retrieved.properties.channels).toEqual(["ai0", "ai1"]);
      });

      it("should retrieve devices by a flat filter with schemas", async () => {
        const location = `flat-${id.create()}`;
        const d = await client.devices.create(
          {
            key: id.create(),
            rack: testRack.key,
            location,
            name: "flat-schema",
            make: "ni",
            model: "pxi",
            properties: { rate: 100, channels: ["ch1"] },
          },
          schemas,
        );
        const retrieved = await client.devices.retrieve({
          location,
          make: "ni",
          schemas,
        });
        expect(retrieved.map(({ key }) => key)).toEqual([d.key]);
        expect(retrieved[0].properties.rate).toBe(100);
      });

      it("should return one device for a key list that sets one", async () => {
        const d = await client.devices.create(
          {
            key: id.create(),
            rack: testRack.key,
            location: "Dev1",
            name: "one-option",
            make: "ni",
            model: "pxi",
            properties: { rate: 200, channels: [] },
          },
          schemas,
        );
        const retrieved = await client.devices.retrieve({
          key: [d.key],
          one: true,
          schemas,
        });
        expect(retrieved.properties.rate).toBe(200);
      });

      it("should return one device for a plain key beside other fields", async () => {
        const d = await client.devices.create(
          {
            key: id.create(),
            rack: testRack.key,
            location: "Dev1",
            name: "one-schema",
            make: "ni",
            model: "pxi",
            properties: { rate: 100, channels: ["ch1"] },
          },
          schemas,
        );
        const retrieved = await client.devices.retrieve({
          key: d.key,
          make: "ni",
          schemas,
        });
        expect(retrieved.properties.rate).toBe(100);
        await expect(
          client.devices.retrieve({ key: d.key, make: "labjack", schemas }),
        ).rejects.toThrow(NotFoundError);
      });

      it("should create and retrieve with typed make and model", async () => {
        const makeModelSchemas = {
          properties: record.unknownZ(),
          make: z.literal("labjack"),
          model: z.enum(["t7", "t4", "t8"]),
        };

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
          makeModelSchemas,
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
          schemas,
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
          schemas,
        );

        const retrieved = await client.devices.retrieve({
          where: { key: { in: [d1.key, d2.key] } },
          schemas,
        });
        expect(retrieved).toHaveLength(2);
        expect(retrieved[0].properties.rate).toBe(100);
        expect(retrieved[1].properties.rate).toBe(200);
      });
    });
  });

  describe("onChange", () => {
    it("merges status updates into a subscribed single query", async () => {
      const key = id.create();
      await client.devices.create({
        key,
        rack: testRack.key,
        location: "Dev1",
        name: "status-single",
        make: "ni",
        model: "dog",
        properties: {},
      });
      const params = { key, includeStatus: true };
      const off = client.devices.onChange(params, vi.fn());
      try {
        await client.devices.retrieve(params);
        await client.statuses.set({
          key: device.statusKey(key),
          name: "",
          variant: "warning",
          message: "device degraded",
          time: TimeStamp.now(),
          details: { rack: testRack.key, device: key },
        });
        await expect
          .poll(() => {
            const cached = client.devices.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.status?.message;
          })
          .toBe("device degraded");
      } finally {
        off();
      }
    });

    it("merges status updates into a subscribed request query", async () => {
      const make = id.create();
      const dev = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "status-request",
        make,
        model: "dog",
        properties: {},
      });
      const params = { where: { make: { eq: make } }, includeStatus: true };
      const off = client.devices.onChange(params, vi.fn());
      try {
        await client.devices.retrieve(params);
        await client.statuses.set({
          key: device.statusKey(dev.key),
          name: "",
          variant: "error",
          message: "device has issues",
          time: TimeStamp.now(),
          details: { rack: testRack.key, device: dev.key },
        });
        await expect
          .poll(() => {
            const cached = client.devices.getCached(params);
            if (!query.isLive(cached)) return undefined;
            return cached.find((d) => d.key === dev.key)?.status?.message;
          })
          .toBe("device has issues");
      } finally {
        off();
      }
    });
  });

  describe("getCached", () => {
    it("returns undefined for an unfetched filter query", async () => {
      const make = id.create();
      const d1 = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "cached-filter-1",
        make,
        model: "dog",
        properties: {},
      });
      const d2 = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev2",
        name: "cached-filter-2",
        make: "other",
        model: "dog",
        properties: {},
      });
      await client.devices.retrieve({ where: { key: { in: [d1.key, d2.key] } } });
      // Fetched records in the store do not answer a filter query that was
      // never fetched itself.
      expect(
        client.devices.getCached({ where: { make: { eq: make } } }),
      ).toBeUndefined();
    });

    it("returns undefined for unfetched server-computed query shapes", async () => {
      const make = id.create();
      const dev = await client.devices.create({
        key: id.create(),
        rack: testRack.key,
        location: "Dev1",
        name: "cached-no-approx",
        make,
        model: "dog",
        properties: {},
      });
      await client.devices.retrieve({ where: { key: { in: [dev.key] } } });
      expect(
        client.devices.getCached({ where: { make: { eq: make } }, limit: 1 }),
      ).toBeUndefined();
      expect(client.devices.getCached({ searchTerm: make })).toBeUndefined();
    });
  });
});
