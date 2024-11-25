// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NIL as ZERO_UUID } from "uuid";
import { describe, expect, test } from "vitest";

import { newClient } from "@/setupspecs";

const client = newClient();

describe("Table", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const table = await client.workspaces.table.create(ws.key, {
        name: "Table",
        data: { one: 1 },
      });
      expect(table.name).toEqual("Table");
      expect(table.key).not.toEqual(ZERO_UUID);
      expect(table.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const table = await client.workspaces.table.create(ws.key, {
        name: "Table",
        data: { one: 1 },
      });
      await client.workspaces.table.rename(table.key, "Table2");
      const res = await client.workspaces.table.retrieve(table.key);
      expect(res.name).toEqual("Table2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const table = await client.workspaces.table.create(ws.key, {
        name: "Table",
        data: { one: 1 },
      });
      await client.workspaces.table.setData(table.key, { two: 2 });
      const res = await client.workspaces.table.retrieve(table.key);
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const table = await client.workspaces.table.create(ws.key, {
        name: "Table",
        data: { one: 1 },
      });
      await client.workspaces.table.delete(table.key);
      await expect(client.workspaces.table.retrieve(table.key)).rejects.toThrow();
    });
  });
});
