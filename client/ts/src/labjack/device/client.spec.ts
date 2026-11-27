// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { NotFoundError } from "@/index";
import { labjack } from "@/labjack";
import { query } from "@/query";
import { resource } from "@/resource";
import { createTestClient, expectLive, waitForStreamLive } from "@/testutil";

const client = createTestClient();
const other = createTestClient();

const configuration = (name: string): labjack.device.ConfigurationUpdate => ({
  group: "configuration",
  name,
  identifier: "",
  ports: {
    model: "t7",
    mux80Attached: false,
    analogInput: {},
    analogOutput: {},
    digitalInput: {},
    digitalOutput: {},
  },
});

const createSerial = (): number => Math.floor(Math.random() * 0xfffffffe) + 1;

describe("LabJack device", async () => {
  await other.connect();
  const rack = await client.racks.create({ name: "labjack-device-test" });
  const createNew = (name = "T7"): labjack.device.New => ({
    rack: rack.key,
    name,
    serial: createSerial(),
    connection: "LJM_ctUSB",
    model: "t7",
    ports: { model: "t7" },
  });
  const create = async (name?: string): Promise<labjack.device.Device> =>
    await client.labjack.devices.create(createNew(name));

  describe("create", () => {
    it("should mint a key and store the device", async () => {
      const d = await create();
      expect(d.key).not.toBe("");
      expect(d.identifier).toBe("");
      const retrieved = await client.labjack.devices.retrieve({ key: d.key });
      expect(retrieved).toEqual(d);
    });

    it("should create several devices", async () => {
      const devices = await client.labjack.devices.create([
        createNew("A"),
        createNew("B"),
      ]);
      expect(devices.map((d) => d.name)).toEqual(["A", "B"]);
    });

    it("should create a device again after deleting it", async () => {
      const d = await create();
      await client.labjack.devices.delete(d.key);
      await client.labjack.devices.create(d);
      expect((await client.labjack.devices.retrieve({ key: d.key })).name).toBe(d.name);
    });

    it("should add a created device to a watched search", async () => {
      const token = `ljdev${Math.random().toString(36).slice(2)}`;
      const list = client.labjack.devices.watch({ searchTerm: token });
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      const d = await create(token);
      await expect
        .poll(() => list.getCached()?.records.map(({ key }) => key))
        .toContain(d.key);
    });

    it("should replace a device whose key exists", async () => {
      const d = await create();
      await client.labjack.devices.create({ ...d, name: "Replaced" });
      const retrieved = await client.labjack.devices.retrieve({ key: d.key });
      expect(retrieved.name).toBe("Replaced");
    });
  });

  describe("optimistic writes", () => {
    const WRITE_FAILED = new Error("write failed");
    const fail = () => {
      throw WRITE_FAILED;
    };
    const createOptimistic = (): labjack.device.New => ({
      ...createNew("Optimistic"),
      key: uuid.create(),
    });

    it("should cache the device before the write commits", async () => {
      const d = createOptimistic();
      let duringWrite: string | undefined;
      await client.labjack.devices.create(d, {
        onOptimistic: () => {
          duringWrite = expectLive(
            client.labjack.devices.watch(d.key ?? "").getCached(),
          ).name;
        },
      });
      expect(duringWrite).toEqual(d.name);
    });

    it("should drop the optimistic device when the write fails", async () => {
      const d = createOptimistic();
      await expect(
        client.labjack.devices.create(d, { onOptimistic: fail }),
      ).rejects.toBe(WRITE_FAILED);
      expect(client.labjack.devices.watch(d.key ?? "").getCached()).toBeUndefined();
    });

    it("should restore the previous device when a replace fails", async () => {
      const d = await create();
      await expect(
        client.labjack.devices.create(
          { ...d, name: "Replacement" },
          { onOptimistic: fail },
        ),
      ).rejects.toBe(WRITE_FAILED);
      expect(expectLive(client.labjack.devices.watch(d.key).getCached()).name).toBe(
        d.name,
      );
    });

    it("should restore the device when a delete fails", async () => {
      const d = await create();
      await expect(
        client.labjack.devices.delete(d.key, { onOptimistic: fail }),
      ).rejects.toBe(WRITE_FAILED);
      expect((await client.labjack.devices.retrieve({ key: d.key })).name).toBe(d.name);
    });
  });

  describe("retrieve", () => {
    it("should include the status of the device", async () => {
      const d = await create();
      const retrieved = await client.labjack.devices.retrieve({
        key: d.key,
        includeStatus: true,
      });
      expect(retrieved.status?.owner).toEqual(labjack.device.resourceID(d.key));
    });

    it("should include the status of a device the stream cached", async () => {
      const d = await create();
      await other.labjack.devices.retrieve({ key: d.key });
      await waitForStreamLive(other.connection);
      await client.labjack.devices.update(d.key, configuration("Cached"));
      await expect
        .poll(async () => (await other.labjack.devices.retrieve({ key: d.key })).name)
        .toBe("Cached");
      const retrieved = await other.labjack.devices.retrieve({
        key: d.key,
        includeStatus: true,
      });
      expect(retrieved.status?.owner).toEqual(labjack.device.resourceID(d.key));
    });

    it("should update the status of a cached device", async () => {
      const d = await create();
      const params = { key: d.key, includeStatus: true };
      onTestFinished(client.labjack.devices.watch(params).onChange(() => {}));
      await client.labjack.devices.retrieve(params);
      await client.statuses.set({
        owner: labjack.device.resourceID(d.key),
        name: "",
        variant: "warning",
        message: "Device degraded",
        time: TimeStamp.now(),
        details: { rack: rack.key, device: d.key },
      });
      await expect
        .poll(() => {
          const cached = client.labjack.devices.watch(params).getCached();
          return query.isLive(cached) ? cached.status?.message : undefined;
        })
        .toBe("Device degraded");
    });

    it("should filter devices by rack", async () => {
      const d = await create();
      const devices = await client.labjack.devices.retrieve({ rack: rack.key });
      expect(devices.map(({ key }) => key)).toContain(d.key);
    });

    it("should include the statuses of a filtered list", async () => {
      const d = await create();
      const devices = await client.labjack.devices.retrieve({
        rack: rack.key,
        includeStatus: true,
      });
      const listed = devices.find(({ key }) => key === d.key);
      expect(listed?.status?.owner).toEqual(labjack.device.resourceID(d.key));
    });

    it("should update the status of a device in a watched list", async () => {
      const d = await create();
      const list = client.labjack.devices.watch({
        rack: rack.key,
        includeStatus: true,
      });
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      await client.statuses.set({
        owner: labjack.device.resourceID(d.key),
        name: "",
        variant: "warning",
        message: "Listed device degraded",
        time: TimeStamp.now(),
        details: { rack: rack.key, device: d.key },
      });
      await expect
        .poll(
          () =>
            list.getCached()?.records.find(({ key }) => key === d.key)?.status?.message,
        )
        .toBe("Listed device degraded");
    });

    it("should throw a NotFoundError for a missing device", async () => {
      await expect(
        client.labjack.devices.retrieve({ key: uuid.create() }),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe("update", () => {
    it("should write the configuration group", async () => {
      const d = await create();
      const stored = await client.labjack.devices.update(d.key, {
        ...configuration("Renamed"),
        identifier: "dev",
      });
      const retrieved = await client.labjack.devices.retrieve({ key: d.key });
      expect(retrieved.name).toBe("Renamed");
      expect(retrieved.identifier).toBe("dev");
      expect(retrieved.model).toBe(d.model);
      expect(stored).toEqual(retrieved);
    });

    it("should write the identity group and keep the configuration", async () => {
      const d = await create();
      const serial = createSerial();
      const stored = await client.labjack.devices.update(d.key, {
        group: "identity",
        rack: rack.key,
        serial,
        connection: "LJM_ctETHERNET",
        model: "t7",
      });
      const retrieved = await client.labjack.devices.retrieve({ key: d.key });
      expect(retrieved.serial).toBe(serial);
      expect(retrieved.connection).toBe("LJM_ctETHERNET");
      expect(retrieved.name).toBe(d.name);
      expect(stored).toEqual(retrieved);
    });
  });

  describe("update errors", () => {
    it("should throw a NotFoundError for a missing device", async () => {
      await expect(
        client.labjack.devices.update(uuid.create(), configuration("Missing")),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe("delete", () => {
    it("should delete the device", async () => {
      const d = await create();
      await client.labjack.devices.delete(d.key);
      await expect(client.labjack.devices.retrieve({ key: d.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("cache", () => {
    it("should receive a device another client creates", async () => {
      await waitForStreamLive(other.connection);
      const seen: string[] = [];
      const stop = other.labjack.devices.onSet((device) => seen.push(device.key));
      const d = await create();
      await expect.poll(() => seen).toContain(d.key);
      stop();
    });

    it("should receive a device another client updates", async () => {
      const seen: string[] = [];
      const d = await create();
      await other.labjack.devices.retrieve({ key: d.key });
      await waitForStreamLive(other.connection);
      const stop = other.labjack.devices.onSet((device) => seen.push(device.name));
      await client.labjack.devices.update(d.key, configuration("Streamed"));
      await expect.poll(() => seen).toContain("Streamed");
      stop();
    });

    it("should drop a device another client deletes", async () => {
      const d = await create();
      await other.labjack.devices.retrieve({ key: d.key });
      await waitForStreamLive(other.connection);
      const deleted: string[] = [];
      const stop = other.labjack.devices.onDelete((key) => deleted.push(key));
      await client.labjack.devices.delete(d.key);
      await expect.poll(() => deleted).toContain(d.key);
      stop();
    });
  });

  describe("resource search", () => {
    it("should rename a device in a subscribed search", async () => {
      const token = `ljdev${Math.random().toString(36).slice(2)}`;
      const d = await create(token);
      expect(
        await client.resources.search({ searchTerm: token, types: ["labjack_device"] }),
      ).toHaveLength(1);
      const list = client.resources.watch({
        searchTerm: token,
        types: ["labjack_device"],
      });
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      await client.labjack.devices.update(d.key, configuration(`${token} renamed`));
      const key = resource.idToString(labjack.device.resourceID(d.key));
      await expect
        .poll(() => list.getCached()?.records.find((r) => r.key === key)?.name)
        .toBe(`${token} renamed`);
    });
  });
});
