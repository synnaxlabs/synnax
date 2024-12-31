// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { color } from "@/color/core";

describe("color.Color", () => {
  describe("constructor", () => {
    test("from hex", () => {
      const c = new color.Color("#7a2c26");
      expect(c.r).toEqual(122);
      expect(c.g).toEqual(44);
      expect(c.b).toEqual(38);
    });
    test("from hex with alpha", () => {
      const c = new color.Color("#7a2c26", 0.5);
      expect(c.r).toEqual(122);
      expect(c.g).toEqual(44);
      expect(c.b).toEqual(38);
      expect(c.a).toEqual(0.5);
    });
    describe("from eight digit hex", () => {
      test("case 1", () => {
        const c = new color.Color("#7a2c26ff");
        expect(c.r).toEqual(122);
        expect(c.g).toEqual(44);
        expect(c.b).toEqual(38);
        expect(c.a).toEqual(1);
      });
      test("case 2", () => {
        const c = new color.Color("#7a2c2605");
        expect(c.r).toEqual(122);
        expect(c.g).toEqual(44);
        expect(c.b).toEqual(38);
        expect(c.a).toEqual(5 / 255);
      });
    });
    test("from rgb", () => {
      const c = new color.Color([122, 44, 38]);
      expect(c.r).toEqual(122);
      expect(c.g).toEqual(44);
      expect(c.b).toEqual(38);
    });
    test("from rgba", () => {
      const c = new color.Color([122, 44, 38, 0.5]);
      expect(c.r).toEqual(122);
      expect(c.g).toEqual(44);
      expect(c.b).toEqual(38);
      expect(c.a).toEqual(0.5);
    });
    test("from c", () => {
      const c = new color.Color(new color.Color("#7a2c26"));
      expect(c.r).toEqual(122);
      expect(c.g).toEqual(44);
      expect(c.b).toEqual(38);
    });
  });
  describe("to hex", () => {
    test("without alpha", () => {
      const c = new color.Color("#7a2c26");
      expect(c.hex).toEqual("#7a2c26");
    });
    test("with alpha", () => {
      const c = new color.Color("#7a2c26", 0.5);
      expect(c.hex).toEqual("#7a2c267f");
    });
  });
  describe("to RGBA255", () => {
    test("with alpha", () => {
      const c = new color.Color("#7a2c26", 0.5);
      const expected = [122, 44, 38, 0.5];
      expect(c.rgba255).toEqual(expected);
    });
    test("without alpha", () => {
      const c = new color.Color("#7a2c26");
      const expected = [122, 44, 38, 1];
      expect(c.rgba255).toEqual(expected);
    });
  });
  describe("to RGBA1", () => {
    test("with alpha", () => {
      const c = new color.Color("#7a2c26", 0.5);
      const expected = [122 / 255, 44 / 255, 38 / 255, 0.5];
      expected.forEach((v, i) => {
        expect(c.rgba1[i]).toBeCloseTo(v);
      });
    });
    test("without alpha", () => {
      const c = new color.Color("#7a2c26");
      const expected = [122 / 255, 44 / 255, 38 / 255, 1];
      expected.forEach((v, i) => {
        expect(c.rgba1[i]).toBeCloseTo(v);
      });
    });
  });
  describe("luminance", () => {
    const tests: Array<[string, number]> = [
      ["#000000", 0],
      ["#ffffff", 1],
    ];
    tests.forEach(([hex, expected]) => {
      test(hex, () => {
        const c = new color.Color(hex);
        expect(c.luminance).toBeCloseTo(expected);
      });
    });
  });
  describe("contrast", () => {
    const tests: Array<[string, string, number]> = [
      ["#000000", "#ffffff", 3],
      ["#ffffff", "#000000", 3],
      ["#000000", "#000000", 1],
      ["#ffffff", "#ffffff", 1],
    ];
    tests.forEach(([hex1, hex2, expected]) => {
      test(`${hex1} ${hex2}`, () => {
        const c1 = new color.Color(hex1);
        const c2 = new color.Color(hex2);
        expect(c1.contrast(c2)).toBeCloseTo(expected);
      });
    });
    test("pick c with highest contrast", () => {
      const c = new color.Color("#000000");
      const c1 = new color.Color("#ffffff");
      const c2 = new color.Color("#0000ff");
      expect(c.pickByContrast(c1, c2)).toEqual(c1);
    });
  });
  describe("grayness", () => {
    const tests: Array<[string, number]> = [
      ["#000000", 0],
      ["#ffffff", 1],
      ["#0000ff", 0],
      ["#00ff00", 0],
      ["#ff0000", 0],
      ["#ffff00", 0],
      ["#fefed4", 0.786],
    ];
    tests.forEach(([hex, expected]) => {
      test(hex, () => {
        const c = new color.Color(hex);
        expect(c.grayness).toBeCloseTo(expected);
      });
    });
  });
});
