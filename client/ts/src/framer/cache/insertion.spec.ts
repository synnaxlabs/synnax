// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { buildInsertionPlan } from "@/framer/cache/insertion";

const plan = (
  removeBefore: number,
  removeAfter: number,
  insertInto: number,
  deleteInBetween: number,
) => ({ removeBefore, removeAfter, insertInto, deleteInBetween });

describe("buildInsertionPlan", () => {
  const SPECS: Array<
    [string, bounds.Crude[], bounds.Crude, ReturnType<typeof plan> | null]
  > = [
    ["insert at the start of an empty array", [], [1, 3], plan(0, 0, 0, 0)],
    ["append past the last bound", [[1, 3]], [4, 6], plan(0, 0, 1, 0)],
    [
      "append past a bound that ends before the value",
      [[3, 7]],
      [8, 9],
      plan(0, 0, 1, 0),
    ],
    ["insert before the first bound", [[3, 7]], [0, 2], plan(0, 0, 0, 0)],
    ["trim the end that overlaps the first bound", [[3, 7]], [1, 5], plan(0, 2, 0, 0)],
    ["skip a value inside the first bound", [[0, 10]], [2, 5], null],
    ["skip a value that starts at a bound's lower edge", [[0, 10]], [0, 5], null],
    [
      "skip a value that starts at a later bound's lower edge",
      [
        [0, 10],
        [20, 30],
      ],
      [20, 25],
      null,
    ],
    ["skip a value that ends at a bound's upper edge", [[0, 10]], [2, 10], null],
    [
      "skip a value inside one bound",
      [
        [1, 3],
        [5, 10],
      ],
      [6, 8],
      null,
    ],
    [
      "insert into the gap between two bounds",
      [
        [1, 3],
        [6, 9],
      ],
      [4, 5],
      plan(0, 0, 1, 0),
    ],
    [
      "trim both ends that overlap two neighbors",
      [
        [1, 3],
        [5, 9],
      ],
      [2, 6],
      plan(1, 1, 1, 0),
    ],
    [
      "replace a bound the value covers",
      [
        [1, 2],
        [3, 5],
        [7, 10],
      ],
      [2, 7],
      plan(0, 0, 1, 1),
    ],
    [
      "replace a bound that starts where the value starts",
      [
        [1, 3],
        [3, 4],
      ],
      [3, 5],
      plan(0, 0, 1, 1),
    ],
    ["trim the start that overlaps the last bound", [[1, 5]], [3, 8], plan(2, 0, 1, 0)],
  ];
  SPECS.forEach(([name, ordered, value, expected]) => {
    it(`should ${name}`, () => {
      expect(buildInsertionPlan(ordered, value)).toEqual(expected);
    });
  });

  it("should plan bigint bounds", () => {
    expect(buildInsertionPlan<bigint>([[0n, 10n]], [5n, 15n])).toEqual(
      plan(5, 0, 1, 0),
    );
  });
});
