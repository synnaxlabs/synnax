// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { color } from "@/color";

describe("color.Color", () => {
  describe("constructor", () => {
    test("from hex", () => {
      const c = color.construct("#7a2c26");
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
    });
    test("from hex with alpha", () => {
      const c = color.construct("#7a2c26", 0.5);
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
      expect(color.aValue(c)).toEqual(0.5);
    });
    describe("from eight digit hex", () => {
      test("case 1", () => {
        const c = color.construct("#7a2c26ff");
        expect(color.rValue(c)).toEqual(122);
        expect(color.gValue(c)).toEqual(44);
        expect(color.bValue(c)).toEqual(38);
        expect(color.aValue(c)).toEqual(1);
      });
      test("case 2", () => {
        const c = color.construct("#7a2c2605");
        expect(color.rValue(c)).toEqual(122);
        expect(color.gValue(c)).toEqual(44);
        expect(color.bValue(c)).toEqual(38);
        expect(color.aValue(c)).toEqual(5 / 255);
      });
    });
    test("from rgb", () => {
      const c = color.construct([122, 44, 38]);
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
    });
    test("from rgba", () => {
      const c = color.construct([122, 44, 38, 0.5]);
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
      expect(color.aValue(c)).toEqual(0.5);
    });
    test("from c", () => {
      const c = color.construct(color.construct("#7a2c26"));
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
    });
  });
  describe("to hex", () => {
    test("without alpha", () => {
      const c = color.construct("#7a2c26");
      expect(color.hex(c)).toEqual("#7a2c26");
    });
    test("with alpha", () => {
      const c = color.construct("#7a2c26", 0.5);
      expect(color.hex(c)).toEqual("#7a2c267f");
    });
  });
  describe("to RGBA255", () => {
    test("with alpha", () => {
      const c = color.construct("#7a2c26", 0.5);
      const expected = [122, 44, 38, 0.5];
      expect(color.construct(c)).toEqual(expected);
    });
    test("without alpha", () => {
      const c = color.construct("#7a2c26");
      const expected = [122, 44, 38, 1];
      expect(color.construct(c)).toEqual(expected);
    });
  });
  describe("to RGBA1", () => {
    test("with alpha", () => {
      const c = color.construct("#7a2c26", 0.5);
      const expected = [122 / 255, 44 / 255, 38 / 255, 0.5];
      expected.forEach((v, i) => {
        expect(color.rgba1(c)[i]).toBeCloseTo(v);
      });
    });
    test("without alpha", () => {
      const c = color.construct("#7a2c26");
      const expected = [122 / 255, 44 / 255, 38 / 255, 1];
      expected.forEach((v, i) => {
        expect(color.rgba1(c)[i]).toBeCloseTo(v);
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
        const c = color.construct(hex);
        expect(color.luminance(c)).toBeCloseTo(expected);
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
        const c1 = color.construct(hex1);
        const c2 = color.construct(hex2);
        expect(color.contrast(c1, c2)).toBeCloseTo(expected);
      });
    });
    test("pick c with highest contrast", () => {
      const c = color.construct("#000000");
      const c1 = color.construct("#ffffff");
      const c2 = color.construct("#0000ff");
      expect(color.pickByContrast(c, c1, c2)).toEqual(c1);
    });
  });
  describe("grayness", () => {
    const tests: Array<[string, number]> = [
      ["#000000", 1],
      ["#ffffff", 1],
      ["#0000ff", 0],
      ["#00ff00", 0],
      ["#ff0000", 0],
      ["#ffff00", 0],
      ["#fefed4", 0.834],
      ["#5c6670", 0.92],
      ["#d3c5c5", 0.945],
    ];
    tests.forEach(([hex, expected]) => {
      test(hex, () => {
        const c = color.construct(hex);
        expect(color.grayness(c)).toBeCloseTo(expected);
      });
    });
  });
});
