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

describe("Device", () => {
  describe("Device", () => {
    describe("create", () => {
      it("should create a device on a rack", async () => {
        const rack = await client.hardware.racks.create({ name: "test" });
        const d = await client.hardware.devices.create({
          rack: rack.key,
          location: "Dev1",
          key: "SN222",
          name: "test",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });
        expect(d.key).toEqual("SN222");
        expect(d.name).toBe("test");
        expect(d.make).toBe("ni");
      });
    });
    describe("retrieve", () => {
      it("should retrieve a device by its key", async () => {
        const rack = await client.hardware.racks.create({ name: "test" });
        const d = await client.hardware.devices.create({
          key: "SN222",
          rack: rack.key,
          location: "Dev1",
          name: "test",
          make: "ni",
          model: "dog",
          properties: { cat: "dog" },
        });
        const retrieved = await client.hardware.devices.retrieve(d.key);
        expect(retrieved.key).toBe(d.key);
        expect(retrieved.name).toBe("test");
        expect(retrieved.make).toBe("ni");
      });
    });
  });
});
