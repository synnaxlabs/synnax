// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { ni } from "@/ni";
import { createTestClient, waitForStreamLive } from "@/testutil";

const client = createTestClient();
const other = createTestClient();

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

    it("should filter devices by rack", async () => {
      const d = await create();
      const devices = await client.ni.devices.retrieve({ rack: rack.key });
      expect(devices.map(({ key }) => key)).toContain(d.key);
    });

    it("should throw a NotFoundError for a missing device", async () => {
      await expect(client.ni.devices.retrieve({ key: id.create() })).rejects.toThrow(
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
      const seen: string[] = [];
      const d = await create();
      await other.ni.devices.retrieve({ key: d.key });
      await waitForStreamLive(other.connection);
      const stop = other.ni.devices.onSet((device) => seen.push(device.name));
      await client.ni.devices.update(d.key, {
        group: "configuration",
        name: "Streamed",
        identifier: "",
        analogInput: { index: 0, temperature: 0, channels: {} },
        analogOutput: {},
        counterInput: {},
        digitalInput: {},
        digitalOutput: {},
      });
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
});
