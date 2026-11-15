// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { table } from "@/table";

describe("table action wire decode", () => {
  test("decodes the cells of each row through the set-channel schema", () => {
    const wire = {
      key: "c211c9f3-0b3c-4f5a-9185-e9cabc09a091",
      dispatch_key: "abc",
      seq: 1,
      actions: [
        {
          type: "create",
          create: {
            table: {
              key: "c211c9f3-0b3c-4f5a-9185-e9cabc09a091",
              name: "Metrics Table",
              rows: [
                {
                  size: 36,
                  cells: [
                    { key: "UnSv19BHjPB", config: { variant: "value" } },
                    { key: "SMm7XhRkCJM", config: { variant: "text" } },
                  ],
                },
              ],
              columns: [{ size: 72 }, { size: 72 }],
            },
          },
        },
      ],
    };
    const raw = new TextEncoder().encode(JSON.stringify(wire));
    const buf = new ArrayBuffer(4 + raw.byteLength);
    new DataView(buf).setUint32(0, raw.byteLength, true);
    new Uint8Array(buf).set(raw, 4);
    const s = new Series({ data: buf, dataType: DataType.JSON });
    const [parsed] = s.parseJSON(table.scopedActionZ);
    const action = parsed.actions[0];
    expect(action.type).toEqual("create");
    if (action.type !== "create") return;
    const [row] = action.create.table.rows;
    expect(row.cells.map(({ key, config }) => [key, config.variant])).toEqual([
      ["UnSv19BHjPB", "value"],
      ["SMm7XhRkCJM", "text"],
    ]);
  });
});

describe("table path actions", () => {
  const create = (): table.Table =>
    table.tableZ.parse({
      name: "Readings",
      rows: [
        {
          key: "r0",
          cells: [
            { key: "a", config: { variant: "text", value: "Pressure" } },
            { key: "b", config: { variant: "text", value: "Flow" } },
          ],
        },
      ],
    });
  const configPath = (key: string, ...fields: string[]) => [
    "rows",
    "r0",
    "cells",
    key,
    "config",
    ...fields,
  ];
  const set = (path: string[], value: unknown) => table.set({ path, value });

  it("should set a field of a cell config from its wire form", () => {
    const state = create();
    const { next, targets } = table.reduceAll(state, [
      set(configPath("a", "fill_color"), "#ff0000"),
    ]);
    expect(next.rows[0].cells[0].config).toMatchObject({
      variant: "text",
      fillColor: [255, 0, 0, 255],
    });
    expect(next.rows[0].cells[1]).toBe(state.rows[0].cells[1]);
    expect(targets).toEqual(["a"]);
  });

  it("should return a wire-form inverse that restores the cell", () => {
    const state = create();
    const { next, inverse } = table.reduceAll(state, [
      set(configPath("a", "value"), "Temperature"),
    ]);
    expect(inverse).toEqual([
      table.set({ path: configPath("a", "value"), value: "Pressure" }),
    ]);
    expect(table.reduceAll(next, inverse).next).toEqual(state);
  });

  it("should do nothing when no row holds the cell", () => {
    const state = create();
    const { next, inverse } = table.reduceAll(state, [
      set(["rows", "ghost", "cells", "a", "config", "value"], "Temperature"),
    ]);
    expect(next).toBe(state);
    expect(inverse).toEqual([]);
  });

  it.each([
    ["the rows", ["rows"]],
    ["the size of a row", ["rows", "r0", "size"]],
    ["the key of a cell", ["rows", "r0", "cells", "a", "key"]],
  ])("should throw for a path to %s", (_, path) => {
    expect(() => table.reduceAll(create(), [set(path, 1)])).toThrow(
      "[Table] - a path into rows must go into the config of one cell",
    );
  });

  it("should set the name and return an inverse that restores it", () => {
    const state = create();
    const { next, inverse, targets } = table.reduceAll(state, [set(["name"], "Other")]);
    expect(next.name).toEqual("Other");
    expect(targets).toEqual(["name"]);
    expect(table.reduceAll(next, inverse).next).toEqual(state);
  });

  it("should not classify a set of the name as a cell edit", () => {
    expect(table.kindOf([set(["name"], "Other")])).toEqual("set");
  });

  it("should not classify a nested set outside the rows as a cell edit", () => {
    expect(table.kindOf([set(["layout", "a"], 1)])).toEqual("set");
  });
});

describe("table edits after an insert in one batch", () => {
  const text = (value: string): table.CellConfig =>
    table.cellConfigZ.parse({ variant: "text", value });
  const create = (): table.Table =>
    table.tableZ.parse({
      name: "Readings",
      columns: [{ key: "k0" }, { key: "k1" }],
      rows: [
        {
          key: "r0",
          cells: [
            { key: "a", config: text("A") },
            { key: "b", config: text("B") },
          ],
        },
        {
          key: "r1",
          cells: [
            { key: "c", config: text("C") },
            { key: "d", config: text("D") },
          ],
        },
      ],
    });
  const setValue = (row: string, key: string, value: string) =>
    table.set({ path: ["rows", row, "cells", key, "config", "value"], value });

  it("should edit a cell of a row inserted earlier in the batch", () => {
    const insert = table.addRow({
      index: 2,
      key: "r2",
      size: 36,
      cells: [{ key: "e", config: text("Old") }],
    });
    // A dispatch applies the actions before the last one alone to encode the batch.
    table.reduceAll(create(), [insert]);
    const { next } = table.reduceAll(create(), [insert, setValue("r2", "e", "New")]);
    expect(next.rows[2].cells[0].config).toEqual(text("New"));
  });

  it("should edit one replica of a cell template without changing the others", () => {
    const { next } = table.reduceAll(create(), [
      table.addRow({
        index: 2,
        key: "r2",
        size: 36,
        cellTemplate: { key: "t", config: text("Old") },
      }),
      setValue("r2", "t-0000", "New"),
    ]);
    expect(next.rows[2].cells.map((c) => c.config)).toEqual([text("New"), text("Old")]);
  });

  it("should edit one erased cell without changing the others", () => {
    const { next } = table.reduceAll(create(), [
      table.eraseCells({ cells: ["a", "d"], template: text("") }),
      setValue("r0", "a", "New"),
    ]);
    expect(next.rows[0].cells[0].config).toEqual(text("New"));
    expect(next.rows[1].cells[1].config).toEqual(text(""));
  });
});

describe("table setChanged", () => {
  const text = (value: string, extra: object = {}): table.CellConfig =>
    table.cellConfigZ.parse({ variant: "text", value, ...extra });

  it("should keep an edit to another field that base does not hold", () => {
    const state = table.tableZ.parse({
      name: "Readings",
      rows: [
        {
          key: "r0",
          cells: [{ key: "a", config: text("Pressure", { fillColor: "#ff0000" }) }],
        },
      ],
    });
    const actions = table.setChanged({
      row: "r0",
      key: "a",
      base: text("Pressure"),
      next: text("Flow"),
    });
    const { next } = table.reduceAll(
      state,
      actions.map((a) => table.encode(state, a)),
    );
    expect(next.rows[0].cells[0].config).toEqual(
      text("Flow", { fillColor: "#ff0000" }),
    );
  });

  it("should list no action when nothing differs", () => {
    const base = text("Pressure");
    expect(
      table.setChanged({ row: "r0", key: "a", base, next: text("Pressure") }),
    ).toEqual([]);
  });

  it("should classify edits to one cell as one kind", () => {
    const base = text("Pressure");
    const single = table.setChanged({ row: "r0", key: "a", base, next: text("Flow") });
    const both = table.setChanged({
      row: "r0",
      key: "a",
      base,
      next: text("Flow", { fillColor: "#ff0000" }),
    });
    const whole = table.setCell({ cell: { key: "a", config: base } });
    expect(table.kindOf(single)).toEqual("set_cell:a");
    expect(table.kindOf(both)).toEqual("set_cell:a");
    expect(table.kindOf([whole])).toEqual("set_cell:a");
  });

  it("should not classify edits to two cells as one kind", () => {
    const base = text("Pressure");
    const a = table.setChanged({ row: "r0", key: "a", base, next: text("Flow") });
    const b = table.setChanged({ row: "r0", key: "b", base, next: text("Flow") });
    expect(table.kindOf([...a, ...b])).toEqual("transaction");
  });
});

describe("group and project patches", () => {
  const G1 = "11111111-1111-4111-8111-111111111111";
  const G2 = "22222222-2222-4222-8222-222222222222";
  it("should move the document into the group", () => {
    const doc = table.tableZ.parse({ name: "Readings" });
    expect(
      table.reduceAll(doc, [table.set({ path: ["group"], value: G1 })]).next.group,
    ).toEqual(G1);
  });
  it("should move the document out of its group", () => {
    const doc = table.tableZ.parse({ name: "Readings", group: G1 });
    expect(
      table.reduceAll(doc, [table.remove({ key: "group" })]).next.group,
    ).toBeUndefined();
  });
  it("should restore the previous group through its inverse", () => {
    const doc = table.tableZ.parse({ name: "Readings", group: G1 });
    const { next, inverse } = table.reduceAll(doc, [
      table.set({ path: ["group"], value: G2 }),
    ]);
    expect(table.reduceAll(next, inverse).next.group).toEqual(G1);
  });
  it("should move the document into the project", () => {
    const doc = table.tableZ.parse({ name: "Readings", project: G1 });
    expect(
      table.reduceAll(doc, [table.set({ path: ["project"], value: G2 })]).next.project,
    ).toEqual(G2);
  });
});
