// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { type Move, steps } from "@/table/move";

const N = 5;
const RANGE = [...Array(N).keys()];

// Distinct sizes, so a size that fails to travel with its row or column shows.
const createTable = (): table.Table => ({
  key: "00000000-0000-0000-0000-000000000001",
  name: "t",
  rows: RANGE.map((y) => ({
    key: `r${y}`,
    size: 40 + y,
    cells: RANGE.map((x) => `${y}${x}`),
  })),
  columns: RANGE.map((x) => ({ key: `c${x}`, size: 80 + x })),
  cells: Object.fromEntries(
    RANGE.flatMap((y) =>
      RANGE.map((x) => [`${y}${x}`, table.cellConfigZ.parse({ variant: "text" })]),
    ),
  ),
});

// Every selection of the five, at every slot it can land in.
const MOVES: Move[] = [...Array(2 ** N).keys()].slice(1).flatMap((mask) => {
  const indices = RANGE.filter((i) => (mask >> i) & 1);
  return RANGE.slice(0, N - indices.length + 1).map((to) => ({ indices, to }));
});

const orderOf = ({ indices, to }: Move): number[] => {
  const rest = RANGE.filter((i) => !indices.includes(i));
  return [...rest.slice(0, to), ...indices, ...rest.slice(to)];
};

describe("steps", () => {
  it.for(MOVES)("should land rows $indices at $to, and undo", (move) => {
    const state = createTable();
    const { next, inverse } = table.reduceAll(
      state,
      steps(move).map(({ from, to }) =>
        table.moveRow({ key: state.rows[from].key, to }),
      ),
    );
    expect(next.rows).toEqual(orderOf(move).map((y) => state.rows[y]));
    expect(table.reduceAll(next, inverse).next).toEqual(state);
  });

  it.for(MOVES)("should land columns $indices at $to, and undo", (move) => {
    const state = createTable();
    const { next, inverse } = table.reduceAll(
      state,
      steps(move).map(({ from, to }) =>
        table.moveCol({ key: state.columns[from].key, to }),
      ),
    );
    const order = orderOf(move);
    expect(next.columns).toEqual(order.map((x) => state.columns[x]));
    expect(next.rows.map((r) => r.cells)).toEqual(
      state.rows.map((row) => order.map((x) => row.cells[x])),
    );
    expect(table.reduceAll(next, inverse).next).toEqual(state);
  });

  it("should return no steps for a block already in place", () => {
    expect(steps({ indices: [1, 2], to: 1 })).toEqual([]);
  });
});
