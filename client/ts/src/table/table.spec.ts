// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { NotFoundError } from "@/errors";
import { query } from "@/query";
import { table } from "@/table";
import { createTestClient, expectDeleted } from "@/testutil";

const client = createTestClient();

describe("Table", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({ name: "Table", layout: { one: 1 } });
      const t = await client.tables.create(proj.key, {
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
      const proj = await client.projects.create({ name: "Table", layout: { one: 1 } });
      const t = await client.tables.create(proj.key, {
        name: "Table",
      });
      await client.tables.rename(t.key, "Table2");
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.name).toEqual("Table2");
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({ name: "Table", layout: { one: 1 } });
      const t = await client.tables.create(proj.key, {
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
      const proj = await client.projects.create({ name: "CaseTest", layout: {} });
      const t = await client.tables.create(proj.key, {
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
    const createTable = async () => {
      const proj = await client.projects.create({ name: "Dispatch", layout: {} });
      return await client.tables.create(proj.key, {
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
      const t = await createTable();
      await client.tables.dispatch(t.key, [table.rename({ name: "renamed" })]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.name).toEqual("renamed");
    });

    test("addRow appends a row and its cells", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
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
      const t = await createTable();
      await client.tables.dispatch(t.key, [table.removeRow({ index: 0 })]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.rows).toEqual([]);
      expect(Object.keys(res.cells)).toEqual([]);
    });

    test("addCol inserts a column at the given index across every row", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
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
      const t = await createTable();
      await client.tables.dispatch(t.key, [table.removeCol({ index: 0 })]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.columns).toHaveLength(1);
      expect(res.rows[0].cells).toEqual(["b"]);
      expect(res.cells.a).toBeUndefined();
    });

    test("resize actions update size in place", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.resizeRow({ index: 0, size: 55 }),
        table.resizeCol({ index: 1, size: 200 }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.rows[0].size).toEqual(55);
      expect(res.columns[1].size).toEqual(200);
    });

    test("setCell replaces an existing cell", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.setCell({
          cell: { key: "a", variant: "value", props: { units: "psi" } },
        }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.cells.a.variant).toEqual("value");
      expect(res.cells.a.props.units).toEqual("psi");
    });

    test("setCell is a no-op for an unknown key", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.setCell({ cell: { key: "ghost", variant: "text", props: {} } }),
      ]);
      const res = await client.tables.retrieve({ key: t.key });
      expect(res.cells.ghost).toBeUndefined();
      expect(Object.keys(res.cells).sort()).toEqual(["a", "b"]);
    });

    test("multi-action sequence applies atomically", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
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

  describe("reduceAll", () => {
    const KEY = "00000000-0000-0000-0000-000000000001";
    const TEMPLATE_KEY = "11111111-2222-4333-8444-555555555555";
    const DERIVED_KEY_0 = "11111111-2222-4333-8444-555555550000";
    const DERIVED_KEY_1 = "11111111-2222-4333-8444-555555550001";
    const textTemplate: table.CellTemplate = { variant: "text", props: {} };

    const emptyState = (name = "t"): table.Table => ({
      key: KEY,
      name,
      rows: [],
      columns: [],
      cells: {},
    });

    const createTable2x2 = (): table.Table => ({
      key: KEY,
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

    const createTable3x3 = (): table.Table => {
      const keys = ["a", "b", "c", "d", "e", "f", "g", "h", "i"];
      return {
        key: KEY,
        name: "t",
        rows: [
          { size: 36, cells: ["a", "b", "c"] },
          { size: 36, cells: ["d", "e", "f"] },
          { size: 36, cells: ["g", "h", "i"] },
        ],
        columns: [{ size: 80 }, { size: 80 }, { size: 80 }],
        cells: Object.fromEntries(
          keys.map((k) => [k, { key: k, variant: "value", props: { units: "psi" } }]),
        ),
      };
    };

    describe("inverses", () => {
      test("rename produces a rename inverse with the old name", () => {
        const { next, inverse } = table.reduceAll(emptyState("before"), [
          table.rename({ name: "after" }),
        ]);
        expect(next.name).toEqual("after");
        expect(inverse).toHaveLength(1);
        expect(inverse[0].type).toEqual("rename");
        expect(inverse[0].type === "rename" ? inverse[0].rename.name : null).toEqual(
          "before",
        );
      });

      test("addRow inverse is a removeRow at the inserted index", () => {
        const { inverse } = table.reduceAll(emptyState(), [
          table.addRow({
            index: 0,
            size: 30,
            cells: [{ key: "x", variant: "text", props: {} }],
          }),
        ]);
        expect(inverse).toHaveLength(1);
        expect(inverse[0].type).toEqual("remove_row");
      });

      test("setCell inverse restores the previous cell value", () => {
        const { next, inverse } = table.reduceAll(
          {
            ...emptyState(),
            rows: [{ size: 30, cells: ["a"] }],
            columns: [{ size: 80 }],
            cells: { a: { key: "a", variant: "text", props: { v: 1 } } },
          },
          [table.setCell({ cell: { key: "a", variant: "value", props: { v: 2 } } })],
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
        const { inverse } = table.reduceAll(emptyState(), [
          table.setCell({ cell: { key: "ghost", variant: "text", props: {} } }),
        ]);
        expect(inverse).toEqual([]);
      });
    });

    describe("bootstrap", () => {
      test("addRow on an empty table bootstraps one default column per cell", () => {
        const { next } = table.reduceAll(emptyState(), [
          table.addRow({
            index: 0,
            size: 36,
            cells: [
              { key: "a", variant: "text", props: {} },
              { key: "b", variant: "text", props: {} },
            ],
          }),
        ]);
        expect(next.rows).toHaveLength(1);
        expect(next.columns).toHaveLength(2);
        expect(next.columns[0].size).toEqual(72);
        expect(next.rows[0].cells).toEqual(["a", "b"]);
      });

      test("addCol on an empty table bootstraps one default row per cell", () => {
        const { next } = table.reduceAll(emptyState(), [
          table.addCol({
            index: 0,
            size: 72,
            cells: [
              { key: "a", variant: "text", props: {} },
              { key: "b", variant: "text", props: {} },
            ],
          }),
        ]);
        expect(next.columns).toHaveLength(1);
        expect(next.rows).toHaveLength(2);
        expect(next.rows[0].size).toEqual(36);
        expect(next.rows[0].cells).toEqual(["a"]);
        expect(next.rows[1].cells).toEqual(["b"]);
      });
    });

    describe("cellTemplate", () => {
      const template = { key: TEMPLATE_KEY, variant: "text", props: { value: "t" } };

      test("addRow replicates the template across existing columns", () => {
        const { next } = table.reduceAll(
          {
            ...emptyState(),
            rows: [{ size: 36, cells: ["a", "b"] }],
            columns: [{ size: 80 }, { size: 80 }],
            cells: {
              a: { key: "a", variant: "text", props: {} },
              b: { key: "b", variant: "text", props: {} },
            },
          },
          [table.addRow({ index: 1, size: 36, cells: [], cellTemplate: template })],
        );
        expect(next.rows[1].cells).toEqual([DERIVED_KEY_0, DERIVED_KEY_1]);
        expect(next.cells[DERIVED_KEY_0].variant).toEqual("text");
        expect(next.cells[DERIVED_KEY_0].props).toEqual({ value: "t" });
      });

      test("addCol replicates the template across existing rows", () => {
        const { next } = table.reduceAll(
          {
            ...emptyState(),
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
          [table.addCol({ index: 1, size: 80, cells: [], cellTemplate: template })],
        );
        expect(next.rows[0].cells).toEqual(["a", DERIVED_KEY_0]);
        expect(next.rows[1].cells).toEqual(["b", DERIVED_KEY_1]);
      });

      test("addCol with template adds an empty column when rows are empty", () => {
        const { next } = table.reduceAll(
          { ...emptyState(), columns: [{ size: 80 }, { size: 80 }] },
          [table.addCol({ index: 2, size: 72, cells: [], cellTemplate: template })],
        );
        expect(next.rows).toHaveLength(0);
        expect(next.columns).toHaveLength(3);
        expect(next.cells).toEqual({});
      });

      test("addRow with template adds an empty row when columns are empty", () => {
        const { next } = table.reduceAll(
          { ...emptyState(), rows: [{ size: 36, cells: [] }] },
          [table.addRow({ index: 1, size: 36, cells: [], cellTemplate: template })],
        );
        expect(next.rows).toHaveLength(2);
        expect(next.rows[1].cells).toEqual([]);
        expect(next.cells).toEqual({});
      });

      test("addRow with template on an empty table produces a 1x1 grid", () => {
        const { next } = table.reduceAll(emptyState(), [
          table.addRow({ index: 0, size: 36, cells: [], cellTemplate: template }),
        ]);
        expect(next.columns).toHaveLength(1);
        expect(next.rows[0].cells).toEqual([DERIVED_KEY_0]);
      });
    });

    describe("size clamping", () => {
      const MIN = 32;
      const baseWithOneCol = {
        ...emptyState(),
        columns: [{ size: 80 }],
      };
      const baseWithOneCell: table.Table = {
        ...emptyState(),
        rows: [{ size: 60, cells: ["a"] }],
        columns: [{ size: 80 }],
        cells: { a: { key: "a", variant: "text", props: {} } },
      };
      test.for<{
        name: string;
        state: table.Table;
        action: table.Action;
        get: (t: table.Table) => number;
      }>([
        {
          name: "addRow size below minimum",
          state: baseWithOneCol,
          action: table.addRow({
            index: 0,
            size: 5,
            cells: [{ key: "a", variant: "text", props: {} }],
          }),
          get: (t) => t.rows[0].size,
        },
        {
          name: "addCol size below minimum",
          state: baseWithOneCell,
          action: table.addCol({
            index: 1,
            size: 10,
            cells: [{ key: "b", variant: "text", props: {} }],
          }),
          get: (t) => t.columns[1].size,
        },
        {
          name: "resizeRow size below minimum",
          state: baseWithOneCell,
          action: table.resizeRow({ index: 0, size: 5 }),
          get: (t) => t.rows[0].size,
        },
        {
          name: "resizeCol size below minimum",
          state: baseWithOneCell,
          action: table.resizeCol({ index: 0, size: 0 }),
          get: (t) => t.columns[0].size,
        },
      ])("clamps $name to the floor", ({ state, action, get }) => {
        const { next } = table.reduceAll(state, [action]);
        expect(get(next)).toEqual(MIN);
      });
    });

    describe("eraseCells", () => {
      test.for<{
        name: string;
        cells: string[];
        rows: number;
        columns: number;
        row0?: string[];
      }>([
        {
          name: "partial selection leaves axes intact",
          cells: ["b", "e"],
          rows: 3,
          columns: 3,
          row0: ["a", "b", "c"],
        },
        {
          name: "fully-selected row is removed",
          cells: ["d", "e", "f"],
          rows: 2,
          columns: 3,
          row0: ["a", "b", "c"],
        },
        {
          name: "fully-selected column is removed",
          cells: ["b", "e", "h"],
          rows: 3,
          columns: 2,
          row0: ["a", "c"],
        },
        {
          name: "full row + full column in one call",
          cells: ["a", "b", "c", "f", "i"],
          rows: 2,
          columns: 2,
          row0: ["d", "e"],
        },
      ])("$name", ({ cells, rows, columns, row0 }) => {
        const { next } = table.reduceAll(createTable3x3(), [
          table.eraseCells({ cells, template: textTemplate }),
        ]);
        expect(next.rows).toHaveLength(rows);
        expect(next.columns).toHaveLength(columns);
        if (row0 != null) expect(next.rows[0].cells).toEqual(row0);
      });

      test("resets surviving cells to the template variant", () => {
        const { next } = table.reduceAll(createTable3x3(), [
          table.eraseCells({ cells: ["b", "e"], template: textTemplate }),
        ]);
        expect(next.cells.b.variant).toEqual("text");
        expect(next.cells.e.variant).toEqual("text");
        expect(next.cells.a.variant).toEqual("value");
      });

      test("drops cells from the map when their row is fully removed", () => {
        const { next } = table.reduceAll(createTable3x3(), [
          table.eraseCells({ cells: ["d", "e", "f"], template: textTemplate }),
        ]);
        for (const k of ["d", "e", "f"]) expect(next.cells[k]).toBeUndefined();
      });

      test("inverse restores the prior state on undo", () => {
        const state = createTable3x3();
        const { next, inverse } = table.reduceAll(state, [
          table.eraseCells({
            cells: ["a", "b", "c", "e"],
            template: textTemplate,
          }),
        ]);
        expect(next.rows).toHaveLength(2);
        const { next: restored } = table.reduceAll(next, inverse);
        expect(restored).toEqual(state);
      });

      test("empty selection returns no inverse", () => {
        const { inverse } = table.reduceAll(createTable3x3(), [
          table.eraseCells({ cells: [], template: textTemplate }),
        ]);
        expect(inverse).toEqual([]);
      });

      test("inverse round-trips when payload.cells contains duplicates", () => {
        const state = createTable3x3();
        const { next, inverse } = table.reduceAll(state, [
          table.eraseCells({ cells: ["b", "b", "b"], template: textTemplate }),
        ]);
        expect(next.cells.b.variant).toEqual("text");
        const { next: restored } = table.reduceAll(next, inverse);
        expect(restored).toEqual(state);
      });
    });

    describe("round-trip", () => {
      const roundTrip = (forward: table.Action) => {
        const original = createTable2x2();
        const { next, inverse } = table.reduceAll(original, [forward]);
        const { next: restored } = table.reduceAll(next, inverse);
        expect(restored).toEqual(original);
      };

      test.for<{ name: string; action: table.Action }>([
        { name: "rename", action: table.rename({ name: "after" }) },
        {
          name: "addRow",
          action: table.addRow({
            index: 1,
            size: 40,
            cells: [
              { key: "e", variant: "text", props: { value: "E" } },
              { key: "f", variant: "text", props: { value: "F" } },
            ],
          }),
        },
        { name: "removeRow", action: table.removeRow({ index: 0 }) },
        {
          name: "addCol",
          action: table.addCol({
            index: 1,
            size: 60,
            cells: [
              { key: "m1", variant: "text", props: { value: "M1" } },
              { key: "m2", variant: "value", props: { units: "psi" } },
            ],
          }),
        },
        { name: "removeCol", action: table.removeCol({ index: 0 }) },
        { name: "resizeRow", action: table.resizeRow({ index: 0, size: 55 }) },
        { name: "resizeCol", action: table.resizeCol({ index: 1, size: 200 }) },
        {
          name: "setCell",
          action: table.setCell({
            cell: { key: "a", variant: "value", props: { units: "psi" } },
          }),
        },
      ])("$name round-trips", ({ action }) => roundTrip(action));
    });
  });
});

describe("store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.cache.ensureStreaming();
    const project = await client.projects.create({ name: `tbl-${id.create()}` });
    const created = await client.tables.create(project.key, {
      name: `table-${id.create()}`,
    });
    await client.tables.delete(created.key);
    await expect
      .poll(() => query.Deleted.matches(client.tables.getCached({ key: created.key })))
      .toBe(true);
    const cached = expectDeleted(client.tables.getCached({ key: created.key }));
    expect(cached.corpse.name).toEqual(created.name);
  });
});
