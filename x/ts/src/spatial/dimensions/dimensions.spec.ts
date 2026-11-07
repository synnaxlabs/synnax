// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { dimensions } from "@/spatial/dimensions";

describe("Dimensions", () => {
  describe("construction", () => {
    [
      ["from object", { width: 1, height: 2 }],
      ["from couple", [1, 2]],
      ["from dimensions", { width: 1, height: 2 }],
      ["from signed dimensions", { signedWidth: 1, signedHeight: 2 }],
      ["from XY", { x: 1, y: 2 }],
    ].forEach(([name, arg]) => {
      test(name as string, () => {
        const xy = dimensions.construct(arg as dimensions.Crude);
        expect(xy.width).toEqual(1);
        expect(xy.height).toEqual(2);
      });
    });
  });
  test("couple", () => {
    const d = dimensions.construct([1, 2]);
    expect(dimensions.couple(d)).toEqual([1, 2]);
  });
  describe("equals", () => {
    type T = [dimensions.Crude, dimensions.Crude, boolean];
    const TESTS: T[] = [
      [[1, 1], { width: 1, height: 1 }, true],
      [[1, 1], [1, 1], true],
      [{ width: 1, height: 12 }, { width: 1, height: 1 }, false],
      [{ width: 1, height: 12 }, { width: 12, height: 1 }, false],
      [{ width: 1, height: 12 }, { signedWidth: 1, signedHeight: 12 }, true],
    ];
    TESTS.forEach(([one, two, expected], i) =>
      test(`equals ${i}`, () => expect(dimensions.equals(one, two)).toEqual(expected)),
    );
  });
});
