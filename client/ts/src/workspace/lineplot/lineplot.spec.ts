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

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("LinePlot", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Line Plot",
        layout: { one: 1 },
      });
      const linePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "Line Plot",
        data: { one: 1 },
      });
      expect(linePlot.name).toEqual("Line Plot");
      expect(linePlot.key).not.toEqual(uuid.ZERO);
      expect(linePlot.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Line Plot",
        layout: { one: 1 },
      });
      const linePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "Line Plot",
        data: { one: 1 },
      });
      await client.workspaces.lineplots.rename(linePlot.key, "Line Plot2");
      const res = await client.workspaces.lineplots.retrieve({ key: linePlot.key });
      expect(res.name).toEqual("Line Plot2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({
        name: "Line Plot",
        layout: { one: 1 },
      });
      const linePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "Line Plot",
        data: { one: 1 },
      });
      await client.workspaces.lineplots.setData(linePlot.key, { two: 2 });
      const res = await client.workspaces.lineplots.retrieve({ key: linePlot.key });
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Line Plot",
        layout: { one: 1 },
      });
      const linePlot = await client.workspaces.lineplots.create(ws.key, {
        name: "Line Plot",
        data: { one: 1 },
      });
      await client.workspaces.lineplots.delete(linePlot.key);
      await expect(
        client.workspaces.lineplots.retrieve({ key: linePlot.key }),
      ).rejects.toThrow(NotFoundError);
    });
  });
});
