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

import { NotFoundError } from "@/index";
import { query } from "@/query";
import { table } from "@/table";
import { createTestClient, expectDeleted } from "@/testutil";

const client = createTestClient();

// mkText and mkValue build fully-populated cell configs by running the variant
// through its schema, filling the declared field defaults. Tests use them so
// fixtures and round-trip assertions agree on the post-default shape.
const mkText = (value?: string): table.CellConfig =>
  table.cellConfigZ.parse(
    value == null ? { variant: "text" } : { variant: "text", value },
  );
const mkValue = (units?: string): table.CellConfig =>
  table.cellConfigZ.parse(
    units == null ? { variant: "value" } : { variant: "value", units },
  );

describe("Table", () => {
  describe("create", () => {
    test("create one", async () => {
      const proj = await client.projects.create({ name: "Table" });
      const t = await client.tables.create(proj.key, {
        name: "Table",
      });
      expect(t.name).toEqual("Table");
      expect(t.key).not.toEqual(uuid.ZERO);
      const retrieved = await client.tables.retrieve(t.key);
      expect(retrieved.key).toEqual(t.key);
    });
  });

  describe("rename", () => {
    test("rename one", async () => {
      const proj = await client.projects.create({ name: "Table" });
      const t = await client.tables.create(proj.key, {
        name: "Table",
      });
      await client.tables.rename(t.key, "Table2");
      const res = await client.tables.retrieve(t.key);
      expect(res.name).toEqual("Table2");
    });
  });

  describe("delete", () => {
    test("delete one", async () => {
      const proj = await client.projects.create({ name: "Table" });
      const t = await client.tables.create(proj.key, {
        name: "Table",
      });
      await client.tables.delete(t.key);
      await expect(client.tables.retrieve(t.key)).rejects.toThrow(NotFoundError);
    });
  });

  describe("cell config round-trip", () => {
    test("round-trips a fully-populated value cell config", async () => {
      const proj = await client.projects.create({ name: "ConfigTest" });
      const t = await client.tables.create(proj.key, {
        name: "ConfigTest",
        cells: {
          a: {
            variant: "value",
            channel: 42,
            rollingAverage: 5,
            precision: 3,
            notation: "scientific",
            redline: {
              bands: [
                { key: "hi", threshold: 90, color: [255, 0, 0, 1], flashing: true },
              ],
              smooth: true,
            },
            fillColor: [0, 255, 0, 1],
            units: "psi",
            stalenessTimeout: 5,
          },
        },
      });
      const retrieved = await client.tables.retrieve(t.key);
      const cfg = retrieved.cells.a;
      expect(cfg.variant).toEqual("value");
      if (cfg.variant !== "value") return;
      expect(cfg.channel).toEqual(42);
      expect(cfg.rollingAverage).toEqual(5);
      expect(cfg.precision).toEqual(3);
      expect(cfg.notation).toEqual("scientific");
      expect(cfg.redline).toEqual({
        bands: [{ key: "hi", threshold: 90, color: [255, 0, 0, 1], flashing: true }],
        smooth: true,
      });
      expect(cfg.fillColor).toEqual([0, 255, 0, 1]);
      expect(cfg.units).toEqual("psi");
      expect(cfg.stalenessTimeout).toEqual(5);
    });

    test("rejects a cell config that matches no variant", async () => {
      const proj = await client.projects.create({ name: "BadConfig" });
      await expect(
        client.tables.create(proj.key, {
          name: "BadConfig",
          cells: {
            a: { variant: "hologram" } as unknown as table.CellConfig,
          },
        }),
      ).rejects.toThrow();
    });
  });

  describe("dispatch", () => {
    const createTable = async () => {
      const proj = await client.projects.create({ name: "Dispatch" });
      return await client.tables.create(proj.key, {
        name: "Dispatch",
        rows: [{ key: "r0", size: 30, cells: ["a", "b"] }],
        columns: [
          { key: "c0", size: 80 },
          { key: "c1", size: 100 },
        ],
        cells: {
          a: { variant: "text", value: "A" },
          b: { variant: "text", value: "B" },
        },
      });
    };

    test("rename applies via dispatch", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [table.rename({ name: "renamed" })]);
      const res = await client.tables.retrieve(t.key);
      expect(res.name).toEqual("renamed");
    });

    test("addRow appends a row and its cells", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.addRow({
          index: 1,
          key: "r1",
          size: 40,
          cells: [
            { key: "c", config: mkText("C") },
            { key: "d", config: mkText("D") },
          ],
        }),
      ]);
      const res = await client.tables.retrieve(t.key);
      expect(res.rows).toHaveLength(2);
      expect(res.rows[1]).toEqual({ key: "r1", size: 40, cells: ["c", "d"] });
      expect(res.cells.c).toEqual(mkText("C"));
    });

    test("addRow derives the same bootstrap column keys as the Core", async () => {
      const proj = await client.projects.create({ name: "Dispatch" });
      const t = await client.tables.create(proj.key, { name: "Dispatch" });
      const action = table.addRow({
        index: 0,
        size: 36,
        cells: [
          { key: "a", config: mkText() },
          { key: "b", config: mkText() },
        ],
      });
      await client.tables.dispatch(t.key, [action]);
      const res = await client.tables.retrieve(t.key);
      expect(res.columns).toEqual(table.reduceAll(t, [action]).next.columns);
    });

    test("removeRow drops the row and its cells", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [table.removeRow({ key: "r0" })]);
      const res = await client.tables.retrieve(t.key);
      expect(res.rows).toEqual([]);
      expect(Object.keys(res.cells)).toEqual([]);
    });

    test("addCol inserts a column at the given index across every row", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.addCol({
          index: 1,
          key: "mid",
          size: 90,
          cells: [{ key: "mid-1", config: mkText("M1") }],
        }),
      ]);
      const res = await client.tables.retrieve(t.key);
      expect(res.columns.map((c) => c.key)).toEqual(["c0", "mid", "c1"]);
      expect(res.rows[0].cells).toEqual(["a", "mid-1", "b"]);
    });

    test("removeCol drops every cell in that column", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [table.removeCol({ key: "c0" })]);
      const res = await client.tables.retrieve(t.key);
      expect(res.columns).toHaveLength(1);
      expect(res.rows[0].cells).toEqual(["b"]);
      expect(res.cells.a).toBeUndefined();
    });

    test("resize actions update size in place", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.resizeRow({ key: "r0", size: 55 }),
        table.resizeCol({ key: "c1", size: 200 }),
      ]);
      const res = await client.tables.retrieve(t.key);
      expect(res.rows[0].size).toEqual(55);
      expect(res.columns[1].size).toEqual(200);
    });

    test("setCell replaces an existing cell", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.setCell({
          cell: { key: "a", config: mkValue("psi") },
        }),
      ]);
      const res = await client.tables.retrieve(t.key);
      expect(res.cells.a).toEqual(mkValue("psi"));
    });

    test("setCell is a no-op for an unknown key", async () => {
      const t = await createTable();
      await client.tables.dispatch(t.key, [
        table.setCell({ cell: { key: "ghost", config: mkText() } }),
      ]);
      const res = await client.tables.retrieve(t.key);
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
            { key: "c", config: mkText() },
            { key: "d", config: mkText() },
          ],
        }),
        table.setCell({
          cell: { key: "c", config: mkValue("psi") },
        }),
      ]);
      const res = await client.tables.retrieve(t.key);
      expect(res.name).toEqual("multi");
      expect(res.rows).toHaveLength(2);
      expect(res.cells.c).toEqual(mkValue("psi"));
    });
  });

  describe("reduceAll", () => {
    const KEY = "00000000-0000-0000-0000-000000000001";
    const TEMPLATE_KEY = "11111111-2222-4333-8444-555555555555";
    const DERIVED_KEY_0 = "11111111-2222-4333-8444-555555550000";
    const DERIVED_KEY_1 = "11111111-2222-4333-8444-555555550001";
    const textTemplate: table.CellConfig = mkText();

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
        { key: "r0", size: 36, cells: ["a", "b"] },
        { key: "r1", size: 36, cells: ["c", "d"] },
      ],
      columns: [
        { key: "c0", size: 80 },
        { key: "c1", size: 100 },
      ],
      cells: {
        a: mkText("A"),
        b: mkText("B"),
        c: mkText("C"),
        d: mkText("D"),
      },
    });

    const createTable3x3 = (): table.Table => {
      const keys = ["a", "b", "c", "d", "e", "f", "g", "h", "i"];
      return {
        key: KEY,
        name: "t",
        rows: [
          { key: "r0", size: 36, cells: ["a", "b", "c"] },
          { key: "r1", size: 36, cells: ["d", "e", "f"] },
          { key: "r2", size: 36, cells: ["g", "h", "i"] },
        ],
        columns: [
          { key: "c0", size: 80 },
          { key: "c1", size: 80 },
          { key: "c2", size: 80 },
        ],
        cells: Object.fromEntries(keys.map((k) => [k, mkValue("psi")])),
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

      test("addRow inverse removes the added row by its key", () => {
        const { inverse } = table.reduceAll(emptyState(), [
          table.addRow({
            index: 0,
            key: "n",
            size: 30,
            cells: [{ key: "x", config: mkText() }],
          }),
        ]);
        expect(inverse).toEqual([table.removeRow({ key: "n" })]);
      });

      test("setCell inverse restores the previous cell value", () => {
        const { next, inverse } = table.reduceAll(
          {
            ...emptyState(),
            rows: [{ key: "r0", size: 30, cells: ["a"] }],
            columns: [{ key: "c0", size: 80 }],
            cells: { a: mkText("1") },
          },
          [
            table.setCell({
              cell: { key: "a", config: mkValue("psi") },
            }),
          ],
        );
        expect(next.cells.a.variant).toEqual("value");
        expect(inverse).toHaveLength(1);
        expect(inverse[0].type).toEqual("set_cell");
        if (inverse[0].type === "set_cell")
          expect(inverse[0].setCell.cell).toEqual({
            key: "a",
            config: mkText("1"),
          });
      });

      test("setCell on an unknown key returns no inverse", () => {
        const { inverse } = table.reduceAll(emptyState(), [
          table.setCell({ cell: { key: "ghost", config: mkText() } }),
        ]);
        expect(inverse).toEqual([]);
      });
    });

    describe("bootstrap", () => {
      test("addRow on an empty table bootstraps one default column per cell", () => {
        const { next } = table.reduceAll(emptyState(), [
          table.addRow({
            index: 0,
            key: "n",
            size: 36,
            cells: [
              { key: "a", config: mkText() },
              { key: "b", config: mkText() },
            ],
          }),
        ]);
        expect(next.rows).toHaveLength(1);
        expect(next.columns).toEqual([
          { key: "n-0000", size: 72 },
          { key: "n-0001", size: 72 },
        ]);
        expect(next.rows[0].cells).toEqual(["a", "b"]);
      });

      test("addCol on an empty table bootstraps one default row per cell", () => {
        const { next } = table.reduceAll(emptyState(), [
          table.addCol({
            index: 0,
            key: "n",
            size: 72,
            cells: [
              { key: "a", config: mkText() },
              { key: "b", config: mkText() },
            ],
          }),
        ]);
        expect(next.columns).toHaveLength(1);
        expect(next.rows).toEqual([
          { key: "n-0000", size: 36, cells: ["a"] },
          { key: "n-0001", size: 36, cells: ["b"] },
        ]);
      });
    });

    describe("cellTemplate", () => {
      const template: table.Cell = {
        key: TEMPLATE_KEY,
        config: mkText("t"),
      };

      test("addRow replicates the template across existing columns", () => {
        const { next } = table.reduceAll(
          {
            ...emptyState(),
            rows: [{ key: "r0", size: 36, cells: ["a", "b"] }],
            columns: [
              { key: "c0", size: 80 },
              { key: "c1", size: 80 },
            ],
            cells: {
              a: mkText(),
              b: mkText(),
            },
          },
          [table.addRow({ index: 1, size: 36, cells: [], cellTemplate: template })],
        );
        expect(next.rows[1].cells).toEqual([DERIVED_KEY_0, DERIVED_KEY_1]);
        expect(next.cells[DERIVED_KEY_0]).toEqual(mkText("t"));
      });

      test("addCol replicates the template across existing rows", () => {
        const { next } = table.reduceAll(
          {
            ...emptyState(),
            rows: [
              { key: "r0", size: 36, cells: ["a"] },
              { key: "r1", size: 36, cells: ["b"] },
            ],
            columns: [{ key: "c0", size: 80 }],
            cells: {
              a: mkText(),
              b: mkText(),
            },
          },
          [table.addCol({ index: 1, size: 80, cells: [], cellTemplate: template })],
        );
        expect(next.rows[0].cells).toEqual(["a", DERIVED_KEY_0]);
        expect(next.rows[1].cells).toEqual(["b", DERIVED_KEY_1]);
      });

      test("addCol with template adds an empty column when rows are empty", () => {
        const { next } = table.reduceAll(
          {
            ...emptyState(),
            columns: [
              { key: "c0", size: 80 },
              { key: "c1", size: 80 },
            ],
          },
          [table.addCol({ index: 2, size: 72, cells: [], cellTemplate: template })],
        );
        expect(next.rows).toHaveLength(0);
        expect(next.columns).toHaveLength(3);
        expect(next.cells).toEqual({});
      });

      test("addRow with template adds an empty row when columns are empty", () => {
        const { next } = table.reduceAll(
          { ...emptyState(), rows: [{ key: "r0", size: 36, cells: [] }] },
          [table.addRow({ index: 1, size: 36, cells: [], cellTemplate: template })],
        );
        expect(next.rows).toHaveLength(2);
        expect(next.rows[1].cells).toEqual([]);
        expect(next.cells).toEqual({});
      });

      test("addRow with template on an empty table produces a 1x1 grid", () => {
        const { next } = table.reduceAll(emptyState(), [
          table.addRow({
            index: 0,
            key: "n",
            size: 36,
            cells: [],
            cellTemplate: template,
          }),
        ]);
        expect(next.columns).toEqual([{ key: "n-0000", size: 72 }]);
        expect(next.rows[0].cells).toEqual([DERIVED_KEY_0]);
      });
    });

    describe("size clamping", () => {
      const MIN = 32;
      const baseWithOneCol: table.Table = {
        ...emptyState(),
        columns: [{ key: "c0", size: 80 }],
      };
      const baseWithOneCell: table.Table = {
        ...emptyState(),
        rows: [{ key: "r0", size: 60, cells: ["a"] }],
        columns: [{ key: "c0", size: 80 }],
        cells: { a: mkText() },
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
            cells: [{ key: "a", config: mkText() }],
          }),
          get: (t) => t.rows[0].size,
        },
        {
          name: "addCol size below minimum",
          state: baseWithOneCell,
          action: table.addCol({
            index: 1,
            size: 10,
            cells: [{ key: "b", config: mkText() }],
          }),
          get: (t) => t.columns[1].size,
        },
        {
          name: "resizeRow size below minimum",
          state: baseWithOneCell,
          action: table.resizeRow({ key: "r0", size: 5 }),
          get: (t) => t.rows[0].size,
        },
        {
          name: "resizeCol size below minimum",
          state: baseWithOneCell,
          action: table.resizeCol({ key: "c0", size: 0 }),
          get: (t) => t.columns[0].size,
        },
      ])("clamps $name to the floor", ({ state, action, get }) => {
        const { next } = table.reduceAll(state, [action]);
        expect(get(next)).toEqual(MIN);
      });
    });

    describe("move", () => {
      const cellsOf = (t: table.Table): string[][] => t.rows.map((r) => r.cells);

      test.for<{ name: string; action: table.Action; expected: string[][] }>([
        {
          name: "moveRow forward",
          action: table.moveRow({ key: "r0", to: 2 }),
          expected: [
            ["d", "e", "f"],
            ["g", "h", "i"],
            ["a", "b", "c"],
          ],
        },
        {
          name: "moveRow backward",
          action: table.moveRow({ key: "r2", to: 0 }),
          expected: [
            ["g", "h", "i"],
            ["a", "b", "c"],
            ["d", "e", "f"],
          ],
        },
        {
          name: "moveRow past the end",
          action: table.moveRow({ key: "r0", to: 99 }),
          expected: [
            ["d", "e", "f"],
            ["g", "h", "i"],
            ["a", "b", "c"],
          ],
        },
        {
          name: "moveCol forward",
          action: table.moveCol({ key: "c0", to: 2 }),
          expected: [
            ["b", "c", "a"],
            ["e", "f", "d"],
            ["h", "i", "g"],
          ],
        },
        {
          name: "moveCol backward",
          action: table.moveCol({ key: "c2", to: 0 }),
          expected: [
            ["c", "a", "b"],
            ["f", "d", "e"],
            ["i", "g", "h"],
          ],
        },
        {
          name: "moveCol past the end",
          action: table.moveCol({ key: "c0", to: 99 }),
          expected: [
            ["b", "c", "a"],
            ["e", "f", "d"],
            ["h", "i", "g"],
          ],
        },
      ])("$name lands at the target index and undoes", ({ action, expected }) => {
        const original = createTable3x3();
        const { next, inverse } = table.reduceAll(original, [action]);
        expect(cellsOf(next)).toEqual(expected);
        expect(next.cells).toEqual(original.cells);
        const { next: restored } = table.reduceAll(next, inverse);
        expect(restored).toEqual(original);
      });

      test("moveRow carries the row's key and size", () => {
        const state = createTable3x3();
        state.rows[0].size = 50;
        const { next } = table.reduceAll(state, [table.moveRow({ key: "r0", to: 2 })]);
        expect(next.rows.map((r) => [r.key, r.size])).toEqual([
          ["r1", 36],
          ["r2", 36],
          ["r0", 50],
        ]);
      });

      test("moveCol carries the column's key and size", () => {
        const state = createTable3x3();
        state.columns[0].size = 120;
        const { next } = table.reduceAll(state, [table.moveCol({ key: "c0", to: 2 })]);
        expect(next.columns).toEqual([
          { key: "c1", size: 80 },
          { key: "c2", size: 80 },
          { key: "c0", size: 120 },
        ]);
      });

      test("undo returns the moved row after another session moves a row", () => {
        const { next, inverse } = table.reduceAll(createTable3x3(), [
          table.moveRow({ key: "r0", to: 2 }),
        ]);
        const remote = table.reduceAll(next, [table.moveRow({ key: "r1", to: 2 })]);
        const { next: undone } = table.reduceAll(remote.next, inverse);
        expect(undone.rows.map((r) => r.key)).toEqual(["r0", "r2", "r1"]);
      });

      test("moveCol leaves a row missing a cell at either index unchanged", () => {
        const state = createTable3x3();
        state.rows[1].cells = ["d", "e"];
        const { next, inverse } = table.reduceAll(state, [
          table.moveCol({ key: "c0", to: 2 }),
        ]);
        expect(cellsOf(next)).toEqual([
          ["b", "c", "a"],
          ["d", "e"],
          ["h", "i", "g"],
        ]);
        expect(table.reduceAll(next, inverse).next).toEqual(state);
      });
    });

    describe("keys", () => {
      test.for<{ name: string; action: table.Action; targets: string[] }>([
        {
          name: "addRow",
          action: table.addRow({
            index: 0,
            key: "n",
            size: 36,
            cells: [{ key: "x", config: mkText() }],
          }),
          targets: ["n", "x"],
        },
        {
          name: "removeRow",
          action: table.removeRow({ key: "r0" }),
          targets: ["a", "b", "c", "r0"],
        },
        {
          name: "removeCol",
          action: table.removeCol({ key: "c0" }),
          targets: ["a", "c0", "d", "g"],
        },
        {
          name: "resizeRow",
          action: table.resizeRow({ key: "r0", size: 50 }),
          targets: ["r0"],
        },
        {
          name: "resizeCol",
          action: table.resizeCol({ key: "c0", size: 50 }),
          targets: ["c0"],
        },
        {
          name: "moveRow",
          action: table.moveRow({ key: "r0", to: 2 }),
          targets: ["r0"],
        },
        {
          name: "moveCol",
          action: table.moveCol({ key: "c0", to: 2 }),
          targets: ["c0"],
        },
      ])("$name targets the keys it touches", ({ action, targets }) => {
        const result = table.reduceAll(createTable3x3(), [action]);
        expect([...result.targets].sort()).toEqual(targets);
      });

      test.for<{ name: string; action: table.Action }>([
        { name: "removeRow", action: table.removeRow({ key: "c0" }) },
        { name: "removeCol", action: table.removeCol({ key: "r0" }) },
        { name: "resizeRow", action: table.resizeRow({ key: "ghost", size: 50 }) },
        { name: "resizeCol", action: table.resizeCol({ key: "ghost", size: 50 }) },
        { name: "moveRow", action: table.moveRow({ key: "ghost", to: 0 }) },
        { name: "moveCol", action: table.moveCol({ key: "ghost", to: 0 }) },
      ])("$name of an unknown key is a no-op", ({ action }) => {
        const state = createTable3x3();
        const { next, inverse } = table.reduceAll(state, [action]);
        expect(next).toEqual(state);
        expect(inverse).toEqual([]);
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
              { key: "e", config: mkText("E") },
              { key: "f", config: mkText("F") },
            ],
          }),
        },
        { name: "removeRow", action: table.removeRow({ key: "r0" }) },
        {
          name: "addCol",
          action: table.addCol({
            index: 1,
            size: 60,
            cells: [
              { key: "m1", config: mkText("M1") },
              { key: "m2", config: mkValue("psi") },
            ],
          }),
        },
        { name: "removeCol", action: table.removeCol({ key: "c0" }) },
        { name: "resizeRow", action: table.resizeRow({ key: "r0", size: 55 }) },
        { name: "resizeCol", action: table.resizeCol({ key: "c1", size: 200 }) },
        {
          name: "setCell",
          action: table.setCell({
            cell: { key: "a", config: mkValue("psi") },
          }),
        },
      ])("$name round-trips", ({ action }) => roundTrip(action));
    });
  });
});

describe("store", () => {
  it("tombstones deletes from live delete signals", async () => {
    await client.connect();
    const project = await client.projects.create({ name: `tbl-${id.create()}` });
    const created = await client.tables.create(project.key, {
      name: `table-${id.create()}`,
    });
    await client.tables.delete(created.key);
    await expect
      .poll(() => query.Deleted.matches(client.tables.getCached(created.key)))
      .toBe(true);
    const cached = expectDeleted(client.tables.getCached(created.key));
    expect(cached.corpse.name).toEqual(created.name);
  });
});
