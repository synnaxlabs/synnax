// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import * as xy from "@/spatial/xy";

describe("XY", () => {
  describe("construction", () => {
    [
      ["from object", { x: 1, y: 2 }],
      ["from couple", [1, 2]],
      ["from dimensions", { width: 1, height: 2 }],
      ["from signed dimensions", { signedWidth: 1, signedHeight: 2 }],
      ["from client dimensions", { clientX: 1, clientY: 2 }],
    ].forEach(([name, arg]) => {
      test(name as string, () => {
        const p = xy.construct(arg as xy.Crude);
        expect(p.x).toEqual(1);
        expect(p.y).toEqual(2);
      });
    });
  });
});
test("translateX", () => {
  let p = xy.construct([1, 2]);
  p = xy.translateX(p, 5);
  expect(p.x).toEqual(6);
  expect(p.y).toEqual(2);
});
test("translateY", () => {
  let p = xy.construct([1, 2]);
  p = xy.translateY(p, 5);
  expect(p.x).toEqual(1);
  expect(p.y).toEqual(7);
});
test("translate", () => {
  let p = xy.construct([1, 2]);
  p = xy.translate(p, [5, 5]);
  expect(p.x).toEqual(6);
  expect(p.y).toEqual(7);
});
describe("equals", () => {
  const TESTS: Array<[xy.Crude, xy.Crude, boolean]> = [
    [[1, 1], { x: 1, y: 1 }, true],
    [[1, 1], [1, 1], true],
    [{ x: 1, y: 12 }, { x: 1, y: 1 }, false],
    [{ x: 1, y: 12 }, { width: 1, height: 12 }, true],
    [{ x: 1, y: 12 }, { width: 12, height: 1 }, false],
    [{ x: 1, y: 12 }, { signedWidth: 1, signedHeight: 12 }, true],
  ];
  TESTS.forEach(([one, two, expected], i) => {
    test(`equals ${i}`, () => {
      const p = xy.construct(one);
      expect(xy.equals(p, two)).toEqual(expected);
    });
  });
});
test("couple", () => {
  const p = xy.construct([1, 2]);
  expect(xy.couple(p)).toEqual([1, 2]);
});
