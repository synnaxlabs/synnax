// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import { NotFoundError } from "@/errors";
import { createTestClient } from "@/testutil";

const client = createTestClient();

describe("Project", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      expect(proj.name).toEqual("Schematic");
      expect(proj.key).not.toEqual(uuid.ZERO);
      expect(proj.layout.one).toEqual(1);
    });
  });
  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      await client.projects.rename(proj.key, "Schematic2");
      const res = await client.projects.retrieve(proj.key);
      expect(res.name).toEqual("Schematic2");
    });
  });
  describe("setLayout", () => {
    test("set layout", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      await client.projects.setLayout(proj.key, { two: 2 });
      const res = await client.projects.retrieve(proj.key);
      expect(res.layout.two).toEqual(2);
    });
  });
  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({
        name: "Schematic",
        layout: { one: 1 },
      });
      await client.projects.delete(proj.key);
      await expect(client.projects.retrieve(proj.key)).rejects.toThrow();
    });
  });
  describe("retrieve", () => {
    test("retrieve projects by search term", async () => {
      const prefix = `searchable-project-${id.create()}`;
      const names = [`${prefix}-1`, `${prefix}-2`];
      await client.projects.create(names.map((name) => ({ name, layout: {} })));
      await expect
        .poll(async () => {
          const results = await client.projects.retrieve({ searchTerm: prefix });
          return results.map((p) => p.name).sort();
        })
        .toEqual(names);
    });
    test("omit missing keys when ignoring not found", async () => {
      const proj = await client.projects.create({ name: "survivor", layout: {} });
      const res = await client.projects.retrieve({
        keys: [proj.key, uuid.create()],
        ignoreNotFoundError: true,
      });
      expect(res.map(({ key }) => key)).toEqual([proj.key]);
    });
    test("throw on a missing key when not ignoring not found", async () => {
      await expect(client.projects.retrieve({ keys: [uuid.create()] })).rejects.toThrow(
        NotFoundError,
      );
    });
  });
  describe("case preservation", () => {
    test("should preserve key casing in layout field on create/retrieve cycle", async () => {
      const proj = await client.projects.create({
        name: "CaseTest",
        layout: {
          camelCaseKey: "value1",
          PascalCaseKey: "value2",
          snake_case_key: "value3",
          nested: {
            innerCamelCase: 123,
            InnerPascalCase: { deepKey: true },
          },
        },
      });

      const retrieved = await client.projects.retrieve(proj.key);

      const layout = retrieved.layout as Record<string, unknown>;
      expect(layout.camelCaseKey).toEqual("value1");
      expect(layout.PascalCaseKey).toEqual("value2");
      expect(layout.snake_case_key).toEqual("value3");
      expect((layout.nested as Record<string, unknown>).innerCamelCase).toEqual(123);
      expect(
        (
          (layout.nested as Record<string, unknown>).InnerPascalCase as Record<
            string,
            unknown
          >
        ).deepKey,
      ).toEqual(true);
      expect(Object.keys(layout)).toContain("camelCaseKey");
      expect(Object.keys(layout)).toContain("PascalCaseKey");
      expect(Object.keys(layout)).toContain("snake_case_key");
      expect(Object.keys(layout)).not.toContain("camel_case_key");
      expect(Object.keys(layout)).not.toContain("pascal_case_key");
    });
  });
});
