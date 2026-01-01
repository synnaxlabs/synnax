// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Workspace", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      expect(ws.name).toEqual("Schematic");
      expect(ws.key).not.toEqual(uuid.ZERO);
      expect(ws.layout.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      await client.workspaces.rename(ws.key, "Schematic2");
      const res = await client.workspaces.retrieve(ws.key);
      expect(res.name).toEqual("Schematic2");
    });
  });
  describe("setLayout", () => {
    test("set layout", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      await client.workspaces.setLayout(ws.key, { two: 2 });
      const res = await client.workspaces.retrieve(ws.key);
      expect(res.layout.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      await client.workspaces.delete(ws.key);
      await expect(client.workspaces.retrieve(ws.key)).rejects.toThrow();
    });
  });
});
