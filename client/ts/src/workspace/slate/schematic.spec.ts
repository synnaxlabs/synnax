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

describe("Slate", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Slate",
        layout: { one: 1 },
      });
      const slate = await client.workspaces.slate.create(ws.key, {
        name: "Slate",
        data: { one: 1 },
      });
      expect(slate.name).toEqual("Slate");
      expect(slate.key).not.toEqual(ZERO_UUID);
      expect(slate.data.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Slate",
        layout: { one: 1 },
      });
      const slate = await client.workspaces.slate.create(ws.key, {
        name: "Slate",
        data: { one: 1 },
      });
      await client.workspaces.slate.rename(slate.key, "Slate2");
      const res = await client.workspaces.slate.retrieve(slate.key);
      expect(res.name).toEqual("Slate2");
    });
  });
  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({
        name: "Slate",
        layout: { one: 1 },
      });
      const slate = await client.workspaces.slate.create(ws.key, {
        name: "Slate",
        data: { one: 1 },
      });
      await client.workspaces.slate.setData(slate.key, { two: 2 });
      const res = await client.workspaces.slate.retrieve(slate.key);
      expect(res.data.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Slate",
        layout: { one: 1 },
      });
      const slate = await client.workspaces.slate.create(ws.key, {
        name: "Slate",
        data: { one: 1 },
      });
      await client.workspaces.slate.delete(slate.key);
      await expect(
        client.workspaces.slate.retrieve(slate.key),
      ).rejects.toThrow();
    });
  });
  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "Slate",
        layout: { one: 1 },
      });
      const slate = await client.workspaces.slate.create(ws.key, {
        name: "Slate",
        data: { one: 1 },
      });
      const slate2 = await client.workspaces.slate.copy(
        slate.key,
        "Slate2",
        false,
      );
      expect(slate2.name).toEqual("Slate2");
      expect(slate2.key).not.toEqual(ZERO_UUID);
      expect(slate2.data.one).toEqual(1);
    });
    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const ws = await client.workspaces.create({
          name: "Slate",
          layout: { one: 1 },
        });
        const slate = await client.workspaces.slate.create(ws.key, {
          name: "Slate",
          data: { one: 1 },
        });
        const slate2 = await client.workspaces.slate.copy(
          slate.key,
          "Slate2",
          true,
        );
        await expect(
          client.workspaces.slate.setData(slate2.key, { two: 2 }),
        ).rejects.toThrow(ValidationError);
      });
    });
  });
});
