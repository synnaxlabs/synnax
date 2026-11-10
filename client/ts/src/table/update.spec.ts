// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, Series } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { table } from "@/table";

const createTable = (): table.Table =>
  table.tableZ.parse({
    name: "Readings",
    cells: {
      UnSv19BHjPB: { variant: "text", value: "Pressure" },
      b: { variant: "text", value: "Flow" },
    },
  });

describe("Update", () => {
  it("should keep the cell keys of a wire update", () => {
    const wire = {
      key: "c211c9f3-0b3c-4f5a-9185-e9cabc09a091",
      dispatch_key: "abc",
      seq: 1,
      actions: [
        {
          type: "update",
          update: {
            value: {
              cells: {
                update: { UnSv19BHjPB: { variant: "text", value: "Temperature" } },
                insert: { SMm7XhRkCJM: { variant: "value", rolling_average: 4 } },
              },
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
    const { next } = table.reduceAll(createTable(), parsed.actions);
    expect(Object.keys(next.cells)).toEqual(["UnSv19BHjPB", "b", "SMm7XhRkCJM"]);
    expect(next.cells.UnSv19BHjPB).toMatchObject({
      variant: "text",
      value: "Temperature",
    });
    expect(next.cells.SMm7XhRkCJM).toMatchObject({
      variant: "value",
      rollingAverage: 4,
    });
  });
});

describe("reduceAll", () => {
  it("should merge, insert, and remove cells by key", () => {
    const prev = createTable();
    const { next, inverse } = table.reduceAll(prev, [
      table.update({
        value: {
          cells: {
            update: { UnSv19BHjPB: { variant: "text", value: "Temperature" } },
            insert: { c: { variant: "text", value: "Level" } },
            remove: ["b"],
          },
        },
      }),
    ]);
    expect(Object.keys(next.cells)).toEqual(["UnSv19BHjPB", "c"]);
    expect(next.cells.UnSv19BHjPB).toEqual({
      ...prev.cells.UnSv19BHjPB,
      value: "Temperature",
    });
    expect(table.reduceAll(next, inverse).next).toEqual(prev);
  });

  it("should throw for an update of a cell that the table does not hold", () => {
    expect(() =>
      table.reduceAll(createTable(), [
        table.update({
          value: { cells: { update: { z: { variant: "text", value: "Level" } } } },
        }),
      ]),
    ).toThrow("z not found");
  });
});
