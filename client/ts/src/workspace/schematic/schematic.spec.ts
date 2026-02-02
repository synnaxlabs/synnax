// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { NotFoundError, ValidationError } from "@/errors";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Schematic", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        data: { One: 1 },
      });
      expect(schematic.name).toEqual("Schematic");
      expect(schematic.key).not.toEqual(uuid.ZERO);
      expect(schematic.data.One).toEqual(1);
      const retrieved = await client.workspaces.schematics.retrieve({
        key: schematic.key,
      });
      expect(retrieved.data.One).toEqual(1);
    });
  });

  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      await client.workspaces.schematics.rename(schematic.key, "Schematic2");
      const res = await client.workspaces.schematics.retrieve({
        key: schematic.key,
      });
      expect(res.name).toEqual("Schematic2");
    });
  });

  describe("setData", () => {
    test("set data", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      await client.workspaces.schematics.setData(schematic.key, { two: 2 });
      const res = await client.workspaces.schematics.retrieve({
        key: schematic.key,
      });
      expect(res.data.two).toEqual(2);
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      await client.workspaces.schematics.delete(schematic.key);
      await expect(
        client.workspaces.schematics.retrieve({ key: schematic.key }),
      ).rejects.toThrow(NotFoundError);
    });
  });
  describe("case preservation", () => {
    test("should preserve key casing in data field on create/retrieve cycle", async () => {
      const ws = await client.workspaces.create({
        name: "CaseTest",
        layout: {},
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "CaseTest",
        data: {
          camelCaseKey: "value1",
          PascalCaseKey: "value2",
          snake_case_key: "value3",
          nested: {
            innerCamelCase: 123,
            InnerPascalCase: { deepKey: true },
          },
        },
      });

      const retrieved = await client.workspaces.schematics.retrieve({
        key: schematic.key,
      });

      const data = retrieved.data as Record<string, unknown>;
      expect(data.camelCaseKey).toEqual("value1");
      expect(data.PascalCaseKey).toEqual("value2");
      expect(data.snake_case_key).toEqual("value3");
      expect((data.nested as Record<string, unknown>).innerCamelCase).toEqual(123);
      expect(
        (
          (data.nested as Record<string, unknown>).InnerPascalCase as Record<
            string,
            unknown
          >
        ).deepKey,
      ).toEqual(true);
      expect(Object.keys(data)).toContain("camelCaseKey");
      expect(Object.keys(data)).toContain("PascalCaseKey");
      expect(Object.keys(data)).toContain("snake_case_key");
      expect(Object.keys(data)).not.toContain("camel_case_key");
      expect(Object.keys(data)).not.toContain("pascal_case_key");
    });
  });

  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const schematic = await client.workspaces.schematics.create(ws.key, {
        name: "Schematic",
        data: { one: 1 },
      });
      const schematic2 = await client.workspaces.schematics.copy({
        key: schematic.key,
        name: "Schematic2",
        snapshot: false,
      });
      expect(schematic2.name).toEqual("Schematic2");
      expect(schematic2.key).not.toEqual(uuid.ZERO);
      expect(schematic2.data.one).toEqual(1);
    });

    describe("snapshot", () => {
      it("should not allow the caller to edit the snapshot", async () => {
        const ws = await client.workspaces.create({
          name: "Schematic",
          layout: { one: 1 },
        });
        const schematic = await client.workspaces.schematics.create(ws.key, {
          name: "Schematic",
          data: { one: 1 },
        });
        const schematic2 = await client.workspaces.schematics.copy({
          key: schematic.key,
          name: "Schematic2",
          snapshot: true,
        });
        await expect(
          client.workspaces.schematics.setData(schematic2.key, { two: 2 }),
        ).rejects.toThrow(ValidationError);
      });
    });
  });
});
