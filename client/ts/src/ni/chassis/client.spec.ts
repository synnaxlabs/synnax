// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { NotFoundError } from "@/errors";
import { ni } from "@/ni";
import { createTestClient, waitForStreamLive } from "@/testutil";

const client = createTestClient();
const other = createTestClient();

describe("NI chassis", async () => {
  await other.connect();
  const rack = await client.racks.create({ name: "ni-chassis-test" });
  const create = async (name = "PXI1"): Promise<ni.chassis.Chassis> =>
    await client.ni.chassis.create({
      rack: rack.key,
      name,
      resourceName: id.create(),
      model: "PXIe-1083",
    });

  describe("create", () => {
    it("should mint a key and store the chassis", async () => {
      const c = await create();
      expect(c.key).not.toBe("");
      expect(await client.ni.chassis.retrieve({ key: c.key })).toEqual(c);
    });

    it("should create several chassis", async () => {
      const chassis = await client.ni.chassis.create([
        { rack: rack.key, name: "A", resourceName: id.create(), model: "cDAQ-9178" },
        { rack: rack.key, name: "B", resourceName: id.create(), model: "cDAQ-9178" },
      ]);
      expect(chassis.map((c) => c.name)).toEqual(["A", "B"]);
    });
  });

  describe("retrieve", () => {
    it("should include the status of the chassis", async () => {
      const c = await create();
      const retrieved = await client.ni.chassis.retrieve({
        key: c.key,
        includeStatus: true,
      });
      expect(retrieved.status?.owner).toEqual(ni.chassis.resourceID(c.key));
    });

    it("should throw a NotFoundError for a missing chassis", async () => {
      await expect(client.ni.chassis.retrieve({ key: uuid.create() })).rejects.toThrow(
        NotFoundError,
      );
    });

    it("should filter chassis by rack", async () => {
      const c = await create();
      const chassis = await client.ni.chassis.retrieve({ rack: rack.key });
      expect(chassis.map(({ key }) => key)).toContain(c.key);
    });
  });

  describe("update", () => {
    it("should write the configuration group", async () => {
      const c = await create();
      await client.ni.chassis.update(c.key, {
        group: "configuration",
        name: "Renamed",
      });
      const retrieved = await client.ni.chassis.retrieve({ key: c.key });
      expect(retrieved.name).toBe("Renamed");
      expect(retrieved.model).toBe(c.model);
    });

    it("should write the identity group and keep the configuration", async () => {
      const c = await create("Kept");
      await client.ni.chassis.update(c.key, {
        group: "identity",
        rack: rack.key,
        serial: "01ABCDEF",
        resourceName: c.resourceName,
        model: "PXIe-1085",
      });
      const retrieved = await client.ni.chassis.retrieve({ key: c.key });
      expect(retrieved.serial).toBe("01ABCDEF");
      expect(retrieved.name).toBe("Kept");
    });
  });

  describe("delete", () => {
    it("should delete the chassis and the devices it holds", async () => {
      const c = await create();
      const module = await client.ni.devices.create({
        rack: rack.key,
        name: "Module",
        resourceName: id.create(),
        model: "PXIe-4300",
        chassis: c.key,
      });
      await client.ni.chassis.delete(c.key);
      await expect(client.ni.chassis.retrieve({ key: c.key })).rejects.toThrow(
        NotFoundError,
      );
      await expect
        .poll(
          async () =>
            await client.ni.devices.retrieve({ where: { key: { eq: module.key } } }),
        )
        .toEqual([]);
    });
  });

  describe("cache", () => {
    it("should receive a chassis another client updates", async () => {
      const c = await create();
      await other.ni.chassis.retrieve({ key: c.key });
      await waitForStreamLive(other.connection);
      const seen: string[] = [];
      const stop = other.ni.chassis.onSet((chassis) => seen.push(chassis.name));
      await client.ni.chassis.update(c.key, {
        group: "configuration",
        name: "Streamed",
      });
      await expect.poll(() => seen).toContain("Streamed");
      stop();
    });

    it("should drop a chassis another client deletes", async () => {
      const c = await create();
      await other.ni.chassis.retrieve({ key: c.key });
      await waitForStreamLive(other.connection);
      const deleted: string[] = [];
      const stop = other.ni.chassis.onDelete((key) => deleted.push(key));
      await client.ni.chassis.delete(c.key);
      await expect.poll(() => deleted).toContain(c.key);
      stop();
    });
  });
});
