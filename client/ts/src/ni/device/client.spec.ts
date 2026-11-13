// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, TimeStamp, uuid } from "@synnaxlabs/x";
import { describe, expect, it, onTestFinished } from "vitest";

import { NotFoundError } from "@/errors";
import { ni } from "@/ni";
import { query } from "@/query";
import { resource } from "@/resource";
import { createTestClient, expectLive, waitForStreamLive } from "@/testutil";

const client = createTestClient();
const other = createTestClient();

const configuration = (name: string): ni.device.ConfigurationUpdate => ({
  group: "configuration",
  name,
  identifier: "",
  analogInput: { index: 0, temperature: 0, channels: {} },
  analogOutput: {},
  counterInput: {},
  digitalInput: {},
  digitalOutput: {},
});

describe("NI device", async () => {
  await other.connect();
  const rack = await client.racks.create({ name: "ni-device-test" });
  const create = async (name = "Dev1"): Promise<ni.device.Device> =>
    await client.ni.devices.create({
      rack: rack.key,
      name,
      resourceName: id.create(),
      model: "PXIe-4300",
    });

  describe("create", () => {
    it("should mint a key and store the device", async () => {
      const d = await create();
      expect(d.key).not.toBe("");
      expect(d.identifier).toBe("");
      const retrieved = await client.ni.devices.retrieve({ key: d.key });
      expect(retrieved).toEqual(d);
    });

    it("should create several devices", async () => {
      const devices = await client.ni.devices.create([
        { rack: rack.key, name: "A", resourceName: id.create(), model: "USB-6001" },
        { rack: rack.key, name: "B", resourceName: id.create(), model: "USB-6001" },
      ]);
      expect(devices.map((d) => d.name)).toEqual(["A", "B"]);
    });

    it("should create a device again after deleting it", async () => {
      const d = await create();
      await client.ni.devices.delete(d.key);
      await client.ni.devices.create(d);
      expect((await client.ni.devices.retrieve({ key: d.key })).name).toBe(d.name);
    });

    it("should add a created device to a watched search", async () => {
      const token = `nidev${Math.random().toString(36).slice(2)}`;
      const list = client.ni.devices.watch({ searchTerm: token });
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      const d = await create(token);
      await expect
        .poll(() => list.getCached()?.records.map(({ key }) => key))
        .toContain(d.key);
    });

    it("should replace a device whose key exists", async () => {
      const d = await create();
      await client.ni.devices.create({ ...d, name: "Replaced" });
      const retrieved = await client.ni.devices.retrieve({ key: d.key });
      expect(retrieved.name).toBe("Replaced");
    });
  });

  describe("optimistic writes", () => {
    const WRITE_FAILED = new Error("write failed");
    const fail = () => {
      throw WRITE_FAILED;
    };
    const createNew = (): ni.device.New => ({
      key: uuid.create(),
      rack: rack.key,
      name: "Optimistic",
      resourceName: id.create(),
      model: "PXIe-4300",
    });

    it("should cache the device before the write commits", async () => {
      const d = createNew();
      let duringWrite: string | undefined;
      await client.ni.devices.create(d, {
        onOptimistic: () => {
          duringWrite = expectLive(
            client.ni.devices.watch(d.key ?? "").getCached(),
          ).name;
        },
      });
      expect(duringWrite).toEqual(d.name);
    });

    it("should drop the optimistic device when the write fails", async () => {
      const d = createNew();
      await expect(client.ni.devices.create(d, { onOptimistic: fail })).rejects.toBe(
        WRITE_FAILED,
      );
      expect(client.ni.devices.watch(d.key ?? "").getCached()).toBeUndefined();
    });

    it("should restore the previous device when a replace fails", async () => {
      const d = await create();
      await expect(
        client.ni.devices.create({ ...d, name: "Replacement" }, { onOptimistic: fail }),
      ).rejects.toBe(WRITE_FAILED);
      expect(expectLive(client.ni.devices.watch(d.key).getCached()).name).toBe(d.name);
    });

    it("should restore the device when a delete fails", async () => {
      const d = await create();
      await expect(
        client.ni.devices.delete(d.key, { onOptimistic: fail }),
      ).rejects.toBe(WRITE_FAILED);
      expect((await client.ni.devices.retrieve({ key: d.key })).name).toBe(d.name);
    });
  });

  describe("retrieve", () => {
    it("should include the status of the device", async () => {
      const d = await create();
      const retrieved = await client.ni.devices.retrieve({
        key: d.key,
        includeStatus: true,
      });
      expect(retrieved.status?.owner).toEqual(ni.device.resourceID(d.key));
    });

    it("should include the status of a device the stream cached", async () => {
      const d = await create();
      await other.ni.devices.retrieve({ key: d.key });
      await waitForStreamLive(other.connection);
      await client.ni.devices.update(d.key, configuration("Cached"));
      await expect
        .poll(async () => (await other.ni.devices.retrieve({ key: d.key })).name)
        .toBe("Cached");
      const retrieved = await other.ni.devices.retrieve({
        key: d.key,
        includeStatus: true,
      });
      expect(retrieved.status?.owner).toEqual(ni.device.resourceID(d.key));
    });

    it("should update the status of a cached device", async () => {
      const d = await create();
      const params = { key: d.key, includeStatus: true };
      onTestFinished(client.ni.devices.watch(params).onChange(() => {}));
      await client.ni.devices.retrieve(params);
      await client.statuses.set({
        owner: ni.device.resourceID(d.key),
        name: "",
        variant: "warning",
        message: "Device degraded",
        time: TimeStamp.now(),
        details: { rack: rack.key, device: d.key },
      });
      await expect
        .poll(() => {
          const cached = client.ni.devices.watch(params).getCached();
          return query.isLive(cached) ? cached.status?.message : undefined;
        })
        .toBe("Device degraded");
    });

    it("should filter devices by rack", async () => {
      const d = await create();
      const devices = await client.ni.devices.retrieve({ rack: rack.key });
      expect(devices.map(({ key }) => key)).toContain(d.key);
    });

    it("should throw a NotFoundError for a missing device", async () => {
      await expect(client.ni.devices.retrieve({ key: uuid.create() })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("update", () => {
    it("should write the configuration group", async () => {
      const d = await create();
      await client.ni.devices.update(d.key, {
        group: "configuration",
        name: "Renamed",
        identifier: "dev",
        analogInput: { index: 0, temperature: 0, channels: {} },
        analogOutput: {},
        counterInput: {},
        digitalInput: {},
        digitalOutput: {},
      });
      const retrieved = await client.ni.devices.retrieve({ key: d.key });
      expect(retrieved.name).toBe("Renamed");
      expect(retrieved.identifier).toBe("dev");
      expect(retrieved.model).toBe(d.model);
    });

    it("should write the identity group and keep the configuration", async () => {
      const d = await create();
      await client.ni.devices.update(d.key, {
        group: "identity",
        rack: rack.key,
        serial: "01ABCDEF",
        resourceName: d.resourceName,
        model: "PXIe-4302",
      });
      const retrieved = await client.ni.devices.retrieve({ key: d.key });
      expect(retrieved.serial).toBe("01ABCDEF");
      expect(retrieved.model).toBe("PXIe-4302");
      expect(retrieved.name).toBe(d.name);
    });
  });

  describe("update errors", () => {
    it("should throw a NotFoundError for a missing device", async () => {
      await expect(
        client.ni.devices.update(uuid.create(), configuration("Missing")),
      ).rejects.toThrow(NotFoundError);
    });
  });

  describe("delete", () => {
    it("should delete the device", async () => {
      const d = await create();
      await client.ni.devices.delete(d.key);
      await expect(client.ni.devices.retrieve({ key: d.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("cache", () => {
    it("should receive a device another client creates", async () => {
      await waitForStreamLive(other.connection);
      const seen: string[] = [];
      const stop = other.ni.devices.onSet((device) => seen.push(device.key));
      const d = await create();
      await expect.poll(() => seen).toContain(d.key);
      stop();
    });

    it("should receive a device another client updates", async () => {
      const seen: string[] = [];
      const d = await create();
      await other.ni.devices.retrieve({ key: d.key });
      await waitForStreamLive(other.connection);
      const stop = other.ni.devices.onSet((device) => seen.push(device.name));
      await client.ni.devices.update(d.key, configuration("Streamed"));
      await expect.poll(() => seen).toContain("Streamed");
      stop();
    });

    it("should drop a device another client deletes", async () => {
      const d = await create();
      await other.ni.devices.retrieve({ key: d.key });
      await waitForStreamLive(other.connection);
      const deleted: string[] = [];
      const stop = other.ni.devices.onDelete((key) => deleted.push(key));
      await client.ni.devices.delete(d.key);
      await expect.poll(() => deleted).toContain(d.key);
      stop();
    });
  });

  describe("resource search", () => {
    it("should rename a device in a subscribed search", async () => {
      const token = `nidev${Math.random().toString(36).slice(2)}`;
      const d = await create(token);
      await expect
        .poll(
          async () =>
            (await client.resources.search({ searchTerm: token, types: ["ni_device"] }))
              .length,
        )
        .toBe(1);
      const list = client.resources.watch({ searchTerm: token, types: ["ni_device"] });
      onTestFinished(list.onChange(() => {}));
      await list.retrieve();
      await client.ni.devices.update(d.key, configuration(`${token} renamed`));
      const key = resource.idToString(ni.device.resourceID(d.key));
      await expect
        .poll(() => list.getCached()?.records.find((r) => r.key === key)?.name)
        .toBe(`${token} renamed`);
    });
  });
});
