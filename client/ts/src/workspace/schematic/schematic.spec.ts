// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it,test } from "vitest";

import { ValidationError } from "@/errors";
import { newClient } from "@/setupspecs";

const client = newClient();

const ZERO_UUID = "00000000-0000-0000-0000-000000000000";

describe("Schematic", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematic.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      expect(schematic.name).toEqual("Schematic");
      expect(schematic.key).not.toEqual(ZERO_UUID);
      expect(schematic.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematic.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      await client.workspaces.schematic.rename(schematic.key, "Schematic2");
      const res = await client.workspaces.schematic.retrieve(schematic.key);
      expect(res.name).toEqual("Schematic2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematic.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      await client.workspaces.schematic.setData(schematic.key, { two: 2 });
      const res = await client.workspaces.schematic.retrieve(schematic.key);
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematic.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      await client.workspaces.schematic.delete(schematic.key);
      await expect(client.workspaces.schematic.retrieve(schematic.key)).rejects.toThrow();
    });
  });
  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematic.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      const schematic2 = await client.workspaces.schematic.copy(schematic.key, "Schematic2", false);
      expect(schematic2.name).toEqual("Schematic2");
      expect(schematic2.key).not.toEqual(ZERO_UUID);
      expect(schematic2.data.one).toEqual(1);
    });
    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const ws = await client.workspaces.create({
          name: "Schematic",
          layout: { one: 1 },
        });
        const schematic = await client.workspaces.schematic.create(ws.key, {
          name: "Schematic",
          data: { one: 1 },
        });
        const schematic2 = await client.workspaces.schematic.copy(schematic.key, "Schematic2", true);
        await expect(
          client.workspaces.schematic.setData(schematic2.key, { two: 2 }),
        ).rejects.toThrow(ValidationError);
      });
    });
  });
});
