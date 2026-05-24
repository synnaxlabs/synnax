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

  describe("dispatch", () => {
    const seed = async () => {
      const ws = await client.workspaces.create({ name: "Dispatch", layout: {} });
      return client.tables.create(ws.key, {
        ...table.ZERO_NEW,
        name: "Dispatch",
        rows: [{ size: 30, cells: ["a", "b"] }],
        columns: [{ size: 80 }, { size: 100 }],
        cells: {
          a: { key: "a", variant: "text", props: { value: "A" } },
          b: { key: "b", variant: "text", props: { value: "B" } },
        },
      });
    };

    test("rename applies via dispatch", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [table.rename({ name: "renamed" })]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.name).toEqual("renamed");
    });

    test("addRow appends a row and its cells", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [
        table.addRow({
          index: 1,
          size: 40,
          cells: [
            { key: "c", variant: "text", props: { value: "C" } },
            { key: "d", variant: "text", props: { value: "D" } },
          ],
        }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.rows).toHaveLength(2);
      expect(res.rows[1].cells).toEqual(["c", "d"]);
      expect(res.cells.c.props.value).toEqual("C");
    });

    test("removeRow drops the row and its cells", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [table.removeRow({ index: 0 })]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.rows).toEqual([]);
      expect(Object.keys(res.cells)).toEqual([]);
    });

    test("addCol inserts a column at the given index across every row", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [
        table.addCol({
          index: 1,
          size: 90,
          cells: [{ key: "mid-1", variant: "text", props: { value: "M1" } }],
        }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.columns).toHaveLength(3);
      expect(res.rows[0].cells).toEqual(["a", "mid-1", "b"]);
    });

    test("removeCol drops every cell in that column", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [table.removeCol({ index: 0 })]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.columns).toHaveLength(1);
      expect(res.rows[0].cells).toEqual(["b"]);
      expect(res.cells.a).toBeUndefined();
    });

    test("resize actions update size in place", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [
        table.resizeRow({ index: 0, size: 55 }),
        table.resizeCol({ index: 1, size: 200 }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.rows[0].size).toEqual(55);
      expect(res.columns[1].size).toEqual(200);
    });

    test("setCell replaces an existing cell", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [
        table.setCell({
          cell: { key: "a", variant: "value", props: { units: "psi" } },
        }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.cells.a.variant).toEqual("value");
      expect(res.cells.a.props.units).toEqual("psi");
    });

    test("setCell is a no-op for an unknown key", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [
        table.setCell({ cell: { key: "ghost", variant: "text", props: {} } }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.cells.ghost).toBeUndefined();
      expect(Object.keys(res.cells).sort()).toEqual(["a", "b"]);
    });

    test("multi-action sequence applies atomically", async () => {
      const t = await seed();
      await client.tables.dispatch(t.key, "dk-1", [
        table.rename({ name: "multi" }),
        table.addRow({
          index: 1,
          size: 40,
          cells: [
            { key: "c", variant: "text", props: {} },
            { key: "d", variant: "text", props: {} },
          ],
        }),
        table.setCell({
          cell: { key: "c", variant: "value", props: { telem: "ch1" } },
        }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.name).toEqual("multi");
      expect(res.rows).toHaveLength(2);
      expect(res.cells.c.variant).toEqual("value");
      expect(res.cells.c.props.telem).toEqual("ch1");
    });
  });

  describe("reduceAll inverse", () => {
    test("rename produces a rename inverse with the old name", () => {
      const { next, inverse } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "before",
          rows: [],
          columns: [],
          cells: {},
        },
        [table.rename({ name: "after" })],
      );
      expect(next.name).toEqual("after");
      expect(inverse).toHaveLength(1);
      expect(inverse[0].type).toEqual("rename");
      expect(inverse[0].type === "rename" ? inverse[0].rename.name : null).toEqual(
        "before",
      );
    });

    test("addRow inverse is a removeRow at the inserted index", () => {
      const { inverse } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [],
          columns: [],
          cells: {},
        },
        [
          table.addRow({
            index: 0,
            size: 30,
            cells: [{ key: "x", variant: "text", props: {} }],
          }),
        ],
      );
      expect(inverse).toHaveLength(1);
      expect(inverse[0].type).toEqual("remove_row");
    });

    test("setCell inverse restores the previous cell value", () => {
      const { next, inverse } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [{ size: 30, cells: ["a"] }],
          columns: [{ size: 80 }],
          cells: { a: { key: "a", variant: "text", props: { v: 1 } } },
        },
        [
          table.setCell({
            cell: { key: "a", variant: "value", props: { v: 2 } },
          }),
        ],
      );
      expect(next.cells.a.variant).toEqual("value");
      expect(inverse).toHaveLength(1);
      expect(inverse[0].type).toEqual("set_cell");
      if (inverse[0].type === "set_cell") {
        expect(inverse[0].setCell.cell.variant).toEqual("text");
        expect(inverse[0].setCell.cell.props).toEqual({ v: 1 });
      }
    });

    test("setCell on an unknown key returns no inverse", () => {
      const { inverse } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [],
          columns: [],
          cells: {},
        },
        [table.setCell({ cell: { key: "ghost", variant: "text", props: {} } })],
      );
      expect(inverse).toEqual([]);
    });

    test("isUndoable returns true for every action variant", () => {
      const actions: table.Action[] = [
        table.rename({ name: "x" }),
        table.addRow({ index: 0, size: 30, cells: [] }),
        table.removeRow({ index: 0 }),
        table.addCol({ index: 0, size: 80, cells: [] }),
        table.removeCol({ index: 0 }),
        table.resizeRow({ index: 0, size: 40 }),
        table.resizeCol({ index: 0, size: 100 }),
        table.setCell({ cell: { key: "a", variant: "text", props: {} } }),
      ];
      for (const a of actions) expect(table.isUndoable(a)).toBe(true);
    });
  });
});
