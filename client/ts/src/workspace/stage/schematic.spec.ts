// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NIL as ZERO_UUID } from "uuid";
import { describe, expect, it, test } from "vitest";

import { ValidationError } from "@/errors";
import { newClient } from "@/setupspecs";

const client = newClient();

describe("Stage", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Stage",
        layout: { one: 1 },
      });
      const stage = await client.workspaces.stage.create(ws.key, {
        name: "Stage",
        data: { one: 1 },
      });
      expect(stage.name).toEqual("Stage");
      expect(stage.key).not.toEqual(ZERO_UUID);
      expect(stage.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Stage",
        layout: { one: 1 },
      });
      const stage = await client.workspaces.stage.create(ws.key, {
        name: "Stage",
        data: { one: 1 },
      });
      await client.workspaces.stage.rename(stage.key, "Stage2");
      const res = await client.workspaces.stage.retrieve(stage.key);
      expect(res.name).toEqual("Stage2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({
        name: "Stage",
        layout: { one: 1 },
      });
      const stage = await client.workspaces.stage.create(ws.key, {
        name: "Stage",
        data: { one: 1 },
      });
      await client.workspaces.stage.setData(stage.key, { two: 2 });
      const res = await client.workspaces.stage.retrieve(stage.key);
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Stage",
        layout: { one: 1 },
      });
      const stage = await client.workspaces.stage.create(ws.key, {
        name: "Stage",
        data: { one: 1 },
      });
      await client.workspaces.stage.delete(stage.key);
      await expect(
        client.workspaces.stage.retrieve(stage.key),
      ).rejects.toThrow();
    });
  });
  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "Stage",
        layout: { one: 1 },
      });
      const stage = await client.workspaces.stage.create(ws.key, {
        name: "Stage",
        data: { one: 1 },
      });
      const stage2 = await client.workspaces.stage.copy(
        stage.key,
        "Stage2",
        false,
      );
      expect(stage2.name).toEqual("Stage2");
      expect(stage2.key).not.toEqual(ZERO_UUID);
      expect(stage2.data.one).toEqual(1);
    });
    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const ws = await client.workspaces.create({
          name: "Stage",
          layout: { one: 1 },
        });
        const stage = await client.workspaces.stage.create(ws.key, {
          name: "Stage",
          data: { one: 1 },
        });
        const stage2 = await client.workspaces.stage.copy(
          stage.key,
          "Stage2",
          true,
        );
        await expect(
          client.workspaces.stage.setData(stage2.key, { two: 2 }),
        ).rejects.toThrow(ValidationError);
      });
    });
  });
});
