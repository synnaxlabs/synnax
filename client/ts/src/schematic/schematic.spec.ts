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

import { type schematic } from "@/schematic";
import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

const newSchematic = (overrides: Partial<schematic.New> = {}): schematic.New => ({
  name: "Test",
  legend: { visible: true, position: { x: 50, y: 50 }, colors: {} },
  nodes: [],
  edges: [],
  props: { myProp: "value" },
  ...overrides,
});

describe("Schematic", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const s = await client.schematics.create(ws.key, newSchematic());
      expect(s.name).toEqual("Test");
      expect(s.key).not.toEqual(uuid.ZERO);
      expect(s.props.myProp).toEqual("value");
      const retrieved = await client.schematics.retrieve({ key: s.key });
      expect(retrieved.props.myProp).toEqual("value");
    });
  });

  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const s = await client.schematics.create(ws.key, newSchematic());
      await client.schematics.rename(s.key, "Schematic2");
      const res = await client.schematics.retrieve({ key: s.key });
      expect(res.name).toEqual("Schematic2");
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const s = await client.schematics.create(ws.key, newSchematic());
      await client.schematics.delete(s.key);
      await expect(client.schematics.retrieve({ key: s.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("case preservation", () => {
    test("should preserve key casing in props field on create/retrieve cycle", async () => {
      const ws = await client.workspaces.create({
        name: "CaseTest",
        layout: {},
      });
      const s = await client.schematics.create(
        ws.key,
        newSchematic({
          name: "CaseTest",
          props: {
            camelCaseKey: "value1",
            PascalCaseKey: "value2",
            snake_case_key: "value3",
            nested: {
              innerCamelCase: 123,
              InnerPascalCase: { deepKey: true },
            },
          },
        }),
      );

      const retrieved = await client.schematics.retrieve({ key: s.key });

      const props = retrieved.props as Record<string, unknown>;
      expect(props.camelCaseKey).toEqual("value1");
      expect(props.PascalCaseKey).toEqual("value2");
      expect(props.snake_case_key).toEqual("value3");
      expect((props.nested as Record<string, unknown>).innerCamelCase).toEqual(123);
      expect(
        (
          (props.nested as Record<string, unknown>).InnerPascalCase as Record<
            string,
            unknown
          >
        ).deepKey,
      ).toEqual(true);
      expect(Object.keys(props)).toContain("camelCaseKey");
      expect(Object.keys(props)).toContain("PascalCaseKey");
      expect(Object.keys(props)).toContain("snake_case_key");
      expect(Object.keys(props)).not.toContain("camel_case_key");
      expect(Object.keys(props)).not.toContain("pascal_case_key");
    });
  });

  describe("copy", () => {
    test("copy one", async () => {
      const ws = await client.workspaces.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      const s = await client.schematics.create(
        ws.key,
        newSchematic({ props: { one: 1 } }),
      );
      const s2 = await client.schematics.copy({
        key: s.key,
        name: "Schematic2",
        snapshot: false,
      });
      expect(s2.name).toEqual("Schematic2");
      expect(s2.key).not.toEqual(uuid.ZERO);
      expect(s2.props.one).toEqual(1);
    });
  });
});
