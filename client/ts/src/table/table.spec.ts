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

    test("addRow on an empty table bootstraps one default column per cell", () => {
      const { next } = table.reduceAll(
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
            size: 36,
            cells: [
              { key: "a", variant: "text", props: {} },
              { key: "b", variant: "text", props: {} },
            ],
          }),
        ],
      );
      expect(next.rows).toHaveLength(1);
      expect(next.columns).toHaveLength(2);
      expect(next.columns[0].size).toEqual(72);
      expect(next.columns[1].size).toEqual(72);
      expect(next.rows[0].cells).toEqual(["a", "b"]);
    });

    test("addCol on an empty table bootstraps one default row per cell", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [],
          columns: [],
          cells: {},
        },
        [
          table.addCol({
            index: 0,
            size: 72,
            cells: [
              { key: "a", variant: "text", props: {} },
              { key: "b", variant: "text", props: {} },
            ],
          }),
        ],
      );
      expect(next.columns).toHaveLength(1);
      expect(next.rows).toHaveLength(2);
      expect(next.rows[0].size).toEqual(36);
      expect(next.rows[1].size).toEqual(36);
      expect(next.rows[0].cells).toEqual(["a"]);
      expect(next.rows[1].cells).toEqual(["b"]);
    });

    test("addRow with cellTemplate replicates the template across existing columns", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [{ size: 36, cells: ["a", "b"] }],
          columns: [{ size: 80 }, { size: 80 }],
          cells: {
            a: { key: "a", variant: "text", props: {} },
            b: { key: "b", variant: "text", props: {} },
          },
        },
        [
          table.addRow({
            index: 1,
            size: 36,
            cells: [],
            cellTemplate: {
              key: "11111111-2222-4333-8444-555555555555",
              variant: "text",
              props: { value: "t" },
            },
          }),
        ],
      );
      expect(next.rows).toHaveLength(2);
      // Derived keys: template[0..32] + "0000", template[0..32] + "0001"
      expect(next.rows[1].cells).toEqual([
        "11111111-2222-4333-8444-555555550000",
        "11111111-2222-4333-8444-555555550001",
      ]);
      expect(next.cells["11111111-2222-4333-8444-555555550000"].variant).toEqual(
        "text",
      );
      expect(next.cells["11111111-2222-4333-8444-555555550000"].props).toEqual({
        value: "t",
      });
    });

    test("addCol with cellTemplate replicates the template across existing rows", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [
            { size: 36, cells: ["a"] },
            { size: 36, cells: ["b"] },
          ],
          columns: [{ size: 80 }],
          cells: {
            a: { key: "a", variant: "text", props: {} },
            b: { key: "b", variant: "text", props: {} },
          },
        },
        [
          table.addCol({
            index: 1,
            size: 80,
            cells: [],
            cellTemplate: {
              key: "11111111-2222-4333-8444-555555555555",
              variant: "text",
              props: {},
            },
          }),
        ],
      );
      expect(next.columns).toHaveLength(2);
      expect(next.rows[0].cells).toEqual(["a", "11111111-2222-4333-8444-555555550000"]);
      expect(next.rows[1].cells).toEqual(["b", "11111111-2222-4333-8444-555555550001"]);
    });

    test("addCol with cellTemplate adds a column with no cells when rows are empty but columns are not", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [],
          columns: [{ size: 80 }, { size: 80 }],
          cells: {},
        },
        [
          table.addCol({
            index: 2,
            size: 72,
            cells: [],
            cellTemplate: {
              key: "11111111-2222-4333-8444-555555555555",
              variant: "text",
              props: {},
            },
          }),
        ],
      );
      expect(next.rows).toHaveLength(0);
      expect(next.columns).toHaveLength(3);
      expect(Object.keys(next.cells)).toHaveLength(0);
    });

    test("addRow with cellTemplate adds a row with no cells when columns are empty but rows are not", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [{ size: 36, cells: [] }],
          columns: [],
          cells: {},
        },
        [
          table.addRow({
            index: 1,
            size: 36,
            cells: [],
            cellTemplate: {
              key: "11111111-2222-4333-8444-555555555555",
              variant: "text",
              props: {},
            },
          }),
        ],
      );
      expect(next.rows).toHaveLength(2);
      expect(next.rows[1].cells).toEqual([]);
      expect(next.columns).toHaveLength(0);
      expect(Object.keys(next.cells)).toHaveLength(0);
    });

    test("addRow with cellTemplate on an empty table creates one column + one replica", () => {
      const { next } = table.reduceAll(
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
            size: 36,
            cells: [],
            cellTemplate: {
              key: "11111111-2222-4333-8444-555555555555",
              variant: "text",
              props: {},
            },
          }),
        ],
      );
      expect(next.columns).toHaveLength(1);
      expect(next.rows).toHaveLength(1);
      expect(next.rows[0].cells).toEqual(["11111111-2222-4333-8444-555555550000"]);
    });

    test("addRow clamps below-minimum sizes to the floor", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [],
          columns: [{ size: 80 }],
          cells: {},
        },
        [
          table.addRow({
            index: 0,
            size: 5,
            cells: [{ key: "a", variant: "text", props: {} }],
          }),
        ],
      );
      expect(next.rows[0].size).toEqual(32);
    });

    test("addCol clamps below-minimum sizes to the floor", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [{ size: 36, cells: ["a"] }],
          columns: [{ size: 80 }],
          cells: { a: { key: "a", variant: "text", props: {} } },
        },
        [
          table.addCol({
            index: 1,
            size: 10,
            cells: [{ key: "b", variant: "text", props: {} }],
          }),
        ],
      );
      expect(next.columns[1].size).toEqual(32);
    });

    const moveSeed = (): table.Table => ({
      key: "00000000-0000-0000-0000-000000000001",
      name: "t",
      rows: [
        { size: 36, cells: ["a", "b", "c"] },
        { size: 40, cells: ["d", "e", "f"] },
        { size: 44, cells: ["g", "h", "i"] },
      ],
      columns: [{ size: 80 }, { size: 90 }, { size: 100 }],
      cells: {
        a: { key: "a", variant: "text", props: {} },
        b: { key: "b", variant: "text", props: {} },
        c: { key: "c", variant: "text", props: {} },
        d: { key: "d", variant: "text", props: {} },
        e: { key: "e", variant: "text", props: {} },
        f: { key: "f", variant: "text", props: {} },
        g: { key: "g", variant: "text", props: {} },
        h: { key: "h", variant: "text", props: {} },
        i: { key: "i", variant: "text", props: {} },
      },
    });

    test("moveRow reorders rows and preserves cell keys", () => {
      const { next } = table.reduceAll(moveSeed(), [table.moveRow({ from: 0, to: 2 })]);
      expect(next.rows.map((r) => r.cells[0])).toEqual(["d", "g", "a"]);
      expect(next.rows.map((r) => r.size)).toEqual([40, 44, 36]);
    });

    test("moveRow clamps an out-of-range target to the last row", () => {
      const { next } = table.reduceAll(moveSeed(), [
        table.moveRow({ from: 0, to: 99 }),
      ]);
      expect(next.rows.map((r) => r.cells[0])).toEqual(["d", "g", "a"]);
    });

    test("moveRow is a no-op when from equals to or from is out of range", () => {
      const seed = moveSeed();
      const same = table.reduceAll(seed, [table.moveRow({ from: 1, to: 1 })]);
      expect(same.next.rows).toEqual(seed.rows);
      const oob = table.reduceAll(seed, [table.moveRow({ from: 99, to: 0 })]);
      expect(oob.next.rows).toEqual(seed.rows);
    });

    test("moveCol reorders columns and the per-row cell at the same offset", () => {
      const { next } = table.reduceAll(moveSeed(), [table.moveCol({ from: 0, to: 2 })]);
      expect(next.columns.map((c) => c.size)).toEqual([90, 100, 80]);
      expect(next.rows[0].cells).toEqual(["b", "c", "a"]);
      expect(next.rows[1].cells).toEqual(["e", "f", "d"]);
      expect(next.rows[2].cells).toEqual(["h", "i", "g"]);
    });

    test("moveRow inverse round-trips", () => {
      const seed = moveSeed();
      const { next, inverse } = table.reduceAll(seed, [
        table.moveRow({ from: 0, to: 2 }),
      ]);
      const back = table.reduceAll(next, inverse);
      expect(back.next.rows).toEqual(seed.rows);
    });

    test("moveCol inverse round-trips", () => {
      const seed = moveSeed();
      const { next, inverse } = table.reduceAll(seed, [
        table.moveCol({ from: 0, to: 2 }),
      ]);
      const back = table.reduceAll(next, inverse);
      expect(back.next.rows).toEqual(seed.rows);
      expect(back.next.columns).toEqual(seed.columns);
    });

    test("resizeRow clamps below-minimum sizes to the floor", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [{ size: 60, cells: ["a"] }],
          columns: [{ size: 80 }],
          cells: { a: { key: "a", variant: "text", props: {} } },
        },
        [table.resizeRow({ index: 0, size: 5 })],
      );
      expect(next.rows[0].size).toEqual(32);
    });

    test("resizeCol clamps below-minimum sizes to the floor", () => {
      const { next } = table.reduceAll(
        {
          key: "00000000-0000-0000-0000-000000000001",
          name: "t",
          rows: [{ size: 30, cells: ["a"] }],
          columns: [{ size: 80 }],
          cells: { a: { key: "a", variant: "text", props: {} } },
        },
        [table.resizeCol({ index: 0, size: 0 })],
      );
      expect(next.columns[0].size).toEqual(32);
    });

    const eraseSeed = (): table.Table => ({
      key: "00000000-0000-0000-0000-000000000001",
      name: "t",
      rows: [
        { size: 36, cells: ["a", "b", "c"] },
        { size: 36, cells: ["d", "e", "f"] },
        { size: 36, cells: ["g", "h", "i"] },
      ],
      columns: [{ size: 80 }, { size: 80 }, { size: 80 }],
      cells: {
        a: { key: "a", variant: "value", props: { units: "psi" } },
        b: { key: "b", variant: "value", props: { units: "psi" } },
        c: { key: "c", variant: "value", props: { units: "psi" } },
        d: { key: "d", variant: "value", props: { units: "psi" } },
        e: { key: "e", variant: "value", props: { units: "psi" } },
        f: { key: "f", variant: "value", props: { units: "psi" } },
        g: { key: "g", variant: "value", props: { units: "psi" } },
        h: { key: "h", variant: "value", props: { units: "psi" } },
        i: { key: "i", variant: "value", props: { units: "psi" } },
      },
    });
    const textTemplate: table.Cell = { key: "", variant: "text", props: {} };

    test("eraseCells resets variant and props on partial selections", () => {
      const { next } = table.reduceAll(eraseSeed(), [
        table.eraseCells({ cells: ["b", "e"], template: textTemplate }),
      ]);
      expect(next.cells.b.variant).toEqual("text");
      expect(next.cells.e.variant).toEqual("text");
      expect(next.cells.a.variant).toEqual("value");
      expect(next.rows).toHaveLength(3);
      expect(next.columns).toHaveLength(3);
    });

    test("eraseCells removes a fully-selected row", () => {
      const { next } = table.reduceAll(eraseSeed(), [
        table.eraseCells({ cells: ["d", "e", "f"], template: textTemplate }),
      ]);
      expect(next.rows).toHaveLength(2);
      expect(next.rows[0].cells).toEqual(["a", "b", "c"]);
      expect(next.rows[1].cells).toEqual(["g", "h", "i"]);
      expect(next.cells.d).toBeUndefined();
    });

    test("eraseCells removes a fully-selected column", () => {
      const { next } = table.reduceAll(eraseSeed(), [
        table.eraseCells({ cells: ["b", "e", "h"], template: textTemplate }),
      ]);
      expect(next.columns).toHaveLength(2);
      expect(next.rows[0].cells).toEqual(["a", "c"]);
      expect(next.cells.b).toBeUndefined();
    });

    test("eraseCells removes a full row and a full column in the same call", () => {
      const { next } = table.reduceAll(eraseSeed(), [
        table.eraseCells({
          cells: ["a", "b", "c", "f", "i"],
          template: textTemplate,
        }),
      ]);
      expect(next.rows).toHaveLength(2);
      expect(next.columns).toHaveLength(2);
      expect(next.rows[0].cells).toEqual(["d", "e"]);
      expect(next.rows[1].cells).toEqual(["g", "h"]);
    });

    test("eraseCells inverse restores the prior state on undo", () => {
      const seed = eraseSeed();
      const { next, inverse } = table.reduceAll(seed, [
        table.eraseCells({ cells: ["a", "b", "c", "e"], template: textTemplate }),
      ]);
      expect(next.rows).toHaveLength(2);
      const { next: restored } = table.reduceAll(next, inverse);
      expect(restored).toEqual(seed);
    });

    test("eraseCells with empty selection returns no inverse", () => {
      const { inverse } = table.reduceAll(eraseSeed(), [
        table.eraseCells({ cells: [], template: textTemplate }),
      ]);
      expect(inverse).toEqual([]);
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
  });

  describe("reduceAll round-trip", () => {
    const seedState = (): table.Table => ({
      key: "00000000-0000-0000-0000-000000000001",
      name: "before",
      rows: [
        { size: 36, cells: ["a", "b"] },
        { size: 36, cells: ["c", "d"] },
      ],
      columns: [{ size: 80 }, { size: 100 }],
      cells: {
        a: { key: "a", variant: "text", props: { value: "A" } },
        b: { key: "b", variant: "text", props: { value: "B" } },
        c: { key: "c", variant: "text", props: { value: "C" } },
        d: { key: "d", variant: "text", props: { value: "D" } },
      },
    });

    const roundTrip = (forward: table.Action) => {
      const original = seedState();
      const { next, inverse } = table.reduceAll(original, [forward]);
      const { next: restored } = table.reduceAll(next, inverse);
      expect(restored).toEqual(original);
    };

    test("rename round-trips", () => roundTrip(table.rename({ name: "after" })));

    test("addRow round-trips", () =>
      roundTrip(
        table.addRow({
          index: 1,
          size: 40,
          cells: [
            { key: "e", variant: "text", props: { value: "E" } },
            { key: "f", variant: "text", props: { value: "F" } },
          ],
        }),
      ));

    test("removeRow round-trips", () => roundTrip(table.removeRow({ index: 0 })));

    test("addCol round-trips", () =>
      roundTrip(
        table.addCol({
          index: 1,
          size: 60,
          cells: [
            { key: "m1", variant: "text", props: { value: "M1" } },
            { key: "m2", variant: "value", props: { units: "psi" } },
          ],
        }),
      ));

    test("removeCol round-trips", () => roundTrip(table.removeCol({ index: 0 })));

    test("resizeRow round-trips", () =>
      roundTrip(table.resizeRow({ index: 0, size: 55 })));

    test("resizeCol round-trips", () =>
      roundTrip(table.resizeCol({ index: 1, size: 200 })));

    test("setCell round-trips", () =>
      roundTrip(
        table.setCell({
          cell: { key: "a", variant: "value", props: { units: "psi" } },
        }),
      ));
  });
});
