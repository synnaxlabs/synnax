// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, test, expect, it } from "vitest";

import { ValidationError } from "@/errors";
import { newClient } from "@/setupspecs";

const client = newClient();

const ZERO_UUID = "00000000-0000-0000-0000-000000000000";

describe("PID", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "PID",
        layout: { one: 1 },
      });
      const pid = await client.workspaces.pid.create(ws.key, {
        name: "PID",
        data: { one: 1 },
      });
      expect(pid.name).toEqual("PID");
      expect(pid.key).not.toEqual(ZERO_UUID);
      expect(pid.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "PID",
        layout: { one: 1 },
      });
      const pid = await client.workspaces.pid.create(ws.key, {
        name: "PID",
        data: { one: 1 },
      });
      await client.workspaces.pid.rename(pid.key, "PID2");
      const res = await client.workspaces.pid.retrieve(pid.key);
      expect(res.name).toEqual("PID2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({
        name: "PID",
        layout: { one: 1 },
      });
      const pid = await client.workspaces.pid.create(ws.key, {
        name: "PID",
        data: { one: 1 },
      });
      await client.workspaces.pid.setData(pid.key, { two: 2 });
      const res = await client.workspaces.pid.retrieve(pid.key);
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "PID",
        layout: { one: 1 },
      });
      const pid = await client.workspaces.pid.create(ws.key, {
        name: "PID",
        data: { one: 1 },
      });
      await client.workspaces.pid.delete(pid.key);
      await expect(client.workspaces.pid.retrieve(pid.key)).rejects.toThrow();
    });
  });
  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "PID",
        layout: { one: 1 },
      });
      const pid = await client.workspaces.pid.create(ws.key, {
        name: "PID",
        data: { one: 1 },
      });
      const pid2 = await client.workspaces.pid.copy(pid.key, "PID2", false);
      expect(pid2.name).toEqual("PID2");
      expect(pid2.key).not.toEqual(ZERO_UUID);
      expect(pid2.data.one).toEqual(1);
    });
    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const ws = await client.workspaces.create({
          name: "PID",
          layout: { one: 1 },
        });
        const pid = await client.workspaces.pid.create(ws.key, {
          name: "PID",
          data: { one: 1 },
        });
        const pid2 = await client.workspaces.pid.copy(pid.key, "PID2", true);
        await expect(
          client.workspaces.pid.setData(pid2.key, { two: 2 }),
        ).rejects.toThrow(ValidationError);
      });
    });
  });
});
