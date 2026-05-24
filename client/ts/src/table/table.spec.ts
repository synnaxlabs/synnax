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
import { table } from "@/table";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Table", () => {
  describe("create", () => {
    test("create one", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const t = await client.tables.create(ws.key, {
        ...table.ZERO_NEW,
        name: "Table",
      });
      expect(t.name).toEqual("Table");
      expect(t.key).not.toEqual(uuid.ZERO);
      const retrieved = await client.tables.retrieve({ key: t.key });
      expect(retrieved.key).toEqual(t.key);
    });
  });

  describe("rename", () => {
    test("rename one", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const t = await client.tables.create(ws.key, {
        ...table.ZERO_NEW,
        name: "Table",
      });
      await client.tables.rename(t.key, "Table2");
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.name).toEqual("Table2");
    });
  });

  describe("setData", () => {
    test("set data replaces body fields while preserving key and name", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const t = await client.tables.create(ws.key, {
        ...table.ZERO_NEW,
        name: "Table",
      });
      await client.tables.setData(t.key, {
        rows: [{ size: 40, cells: ["a", "b"] }],
        columns: [{ size: 80 }, { size: 100 }],
        cells: {
          a: { key: "a", variant: "text", props: { value: "hello" } },
          b: { key: "b", variant: "value", props: { units: "psi" } },
        },
      });
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.name).toEqual("Table");
      expect(res.rows).toHaveLength(1);
      expect(res.rows[0].cells).toEqual(["a", "b"]);
      expect(res.columns).toHaveLength(2);
      expect(res.cells.a.variant).toEqual("text");
      expect((res.cells.a.props as Record<string, unknown>).value).toEqual("hello");
      expect(res.cells.b.variant).toEqual("value");
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const ws = await client.workspaces.create({ name: "Table", layout: { one: 1 } });
      const t = await client.tables.create(ws.key, {
        ...table.ZERO_NEW,
        name: "Table",
      });
      await client.tables.delete(t.key);
      await expect(client.tables.retrieve({ key: t.key })).rejects.toThrow(
        NotFoundError,
      );
    });
  });

  describe("cell props case preservation", () => {
    test("preserves arbitrary key casing within cell props values", async () => {
      const ws = await client.workspaces.create({ name: "CaseTest", layout: {} });
      const t = await client.tables.create(ws.key, {
        ...table.ZERO_NEW,
        name: "CaseTest",
        cells: {
          a: {
            key: "a",
            variant: "value",
            props: {
              camelCaseKey: "value1",
              PascalCaseKey: "value2",
              snake_case_key: "value3",
              nested: {
                innerCamelCase: 123,
                InnerPascalCase: { deepKey: true },
              },
            },
          },
        },
      });
      const retrieved = await client.tables.retrieve({ key: t.key });
      const props = retrieved.cells.a.props as Record<string, unknown>;
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
    });
  });
});
