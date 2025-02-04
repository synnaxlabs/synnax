// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { newClient } from "@/setupspecs";

const client = newClient();

const ZERO_UUID = "00000000-0000-0000-0000-000000000000";

describe("Log", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Log",
        layout: { one: 1 },
      });
      const log = await client.workspaces.log.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      expect(log.name).toEqual("Log");
      expect(log.key).not.toEqual(ZERO_UUID);
      expect(log.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Log",
        layout: { one: 1 },
      });
      const log = await client.workspaces.log.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      await client.workspaces.log.rename(log.key, "Log2");
      const res = await client.workspaces.log.retrieve(log.key);
      expect(res.name).toEqual("Log2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({
        name: "Log",
        layout: { one: 1 },
      });
      const log = await client.workspaces.log.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      await client.workspaces.log.setData(log.key, { two: 2 });
      const res = await client.workspaces.log.retrieve(log.key);
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Log",
        layout: { one: 1 },
      });
      const log = await client.workspaces.log.create(ws.key, {
        name: "Log",
        data: { one: 1 },
      });
      await client.workspaces.log.delete(log.key);
      await expect(client.workspaces.log.retrieve(log.key)).rejects.toThrow();
    });
  });
});
